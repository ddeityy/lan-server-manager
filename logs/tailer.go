package logs

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/encoding/charmap"

	"lan-server-manager/logger"
)

// Target describes where a container's logs live.
type Target struct {
	ContainerName string
	SSHHost       string
	SSHUser       string
	SSHPassword   string
	SSHKeyPath    string
}

// Stream delivers lines and errors from a running docker logs tail.
type Stream struct {
	Lines  <-chan string
	Errors <-chan error
	Stop   func()
}

// Tail runs docker logs -f for the target container. If SSHHost is empty it
// runs docker locally.
func Tail(target Target) (*Stream, error) {
	logger.Infof("Starting log tail for container %q (ssh_host=%q)", target.ContainerName, target.SSHHost)
	if target.SSHHost == "" {
		return tailLocal(target.ContainerName)
	}
	return tailSSH(target)
}

func tailLocal(pattern string) (*Stream, error) {
	logger.Infof("Tailing container matching %q locally", pattern)

	ctx, cancel := context.WithCancel(context.Background())

	containerID, err := resolveContainerIDLocal(ctx, pattern)
	if err != nil {
		cancel()
		logger.Errorf("Local container resolve failed for %q: %v", pattern, err)
		return nil, err
	}
	logger.Infof("Resolved local container ID %s", containerID)

	cmd := exec.CommandContext(ctx, "docker", "logs", "-f", "--tail", "100", containerID)
	return runCommand(ctx, cmd, cancel)
}

func tailSSH(target Target) (*Stream, error) {
	logger.Infof("Preparing SSH config for %s as user %q", target.SSHHost, target.SSHUser)
	sshConfig, err := sshClientConfig(target)
	if err != nil {
		logger.Errorf("SSH config failed: %v", err)
		return nil, err
	}

	host := target.SSHHost
	if _, _, splitErr := net.SplitHostPort(host); splitErr != nil {
		host = net.JoinHostPort(host, "22")
	}

	logger.Infof("Dialing %s", host)
	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		logger.Errorf("SSH dial to %s failed: %v", host, err)
		return nil, fmt.Errorf("ssh dial: %w", err)
	}
	logger.Infof("SSH connection to %s established", host)

	logger.Infof("Resolving container matching %q", target.ContainerName)
	containerID, err := resolveContainerID(client, target.ContainerName)
	if err != nil {
		closeSSHClient(client)
		return nil, err
	}
	logger.Infof("Resolved container ID %s", containerID)

	session, err := client.NewSession()
	if err != nil {
		closeSSHClient(client)
		return nil, fmt.Errorf("ssh session: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		closeSSHSession(session)
		closeSSHClient(client)
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		closeSSHSession(session)
		closeSSHClient(client)
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	cmd := "docker logs -f --tail 100 " + containerID
	logger.Infof("Running remote command: %s", cmd)
	if startErr := session.Start(cmd); startErr != nil {
		closeSSHSession(session)
		closeSSHClient(client)
		return nil, fmt.Errorf("start remote command: %w", startErr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	stop := func() {
		cancel()
		closeSSHSession(session)
		closeSSHClient(client)
	}

	stream := startStream(ctx, stdout, stderr, stop)

	go func() {
		if waitErr := session.Wait(); waitErr != nil {
			logger.Errorf("Remote command exited: %v", waitErr)
		}
		stop()
	}()

	return stream, nil
}

func closeSSHClient(client *ssh.Client) {
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		logger.Warnf("SSH client close failed: %v", err)
	}
}

func closeSSHSession(session *ssh.Session) {
	if session == nil {
		return
	}
	if err := session.Close(); err != nil {
		logger.Warnf("SSH session close failed: %v", err)
	}
}

func sshClientConfig(target Target) (*ssh.ClientConfig, error) {
	var auth []ssh.AuthMethod

	if target.SSHPassword != "" {
		logger.Infof("Using SSH password auth")
		auth = append(auth, ssh.Password(target.SSHPassword))
	}

	if target.SSHKeyPath != "" || target.SSHPassword == "" {
		keyPath := target.SSHKeyPath
		if keyPath == "" {
			usr, err := user.Current()
			if err != nil {
				return nil, fmt.Errorf("current user: %w", err)
			}
			keyPath = filepath.Join(usr.HomeDir, ".ssh", "id_rsa")
		}
		logger.Infof("Trying SSH key %s", keyPath)
		signer, err := loadPrivateKey(keyPath)
		if err != nil {
			logger.Warnf("Failed to load SSH key %s: %v", keyPath, err)
			if target.SSHPassword == "" {
				return nil, err
			}
		}
		if err == nil {
			auth = append(auth, ssh.PublicKeys(signer))
		}
	}

	if len(auth) == 0 {
		return nil, fmt.Errorf("no SSH authentication method available")
	}
	logger.Infof("%d SSH auth method(s) configured", len(auth))

	return &ssh.ClientConfig{
		User: target.SSHUser,
		Auth: auth,
		// The LAN environment uses ephemeral servers with no stable host keys.
		//nolint:gosec // G106: host key verification is intentionally skipped for LAN SSH.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}, nil
}

func loadPrivateKey(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key %s: %w", path, err)
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse key %s: %w", path, err)
	}
	return signer, nil
}

func resolveContainerID(client *ssh.Client, pattern string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("resolve session: %w", err)
	}
	defer closeSSHSession(session)

	logger.Infof("Querying remote docker ps for pattern %q", pattern)
	output, err := session.Output("docker ps --format '{{.ID}} {{.Names}}'")
	if err != nil {
		return "", fmt.Errorf("docker ps: %w", err)
	}

	return findContainerByPattern(string(output), pattern)
}

func resolveContainerIDLocal(ctx context.Context, pattern string) (string, error) {
	logger.Infof("Querying local docker ps for pattern %q", pattern)
	out, err := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.ID}} {{.Names}}").Output()
	if err != nil {
		return "", fmt.Errorf("docker ps: %w", err)
	}
	return findContainerByPattern(string(out), pattern)
}

func findContainerByPattern(output, pattern string) (string, error) {
	for line := range strings.SplitSeq(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		id, name := fields[0], fields[1]
		if matchGlob(pattern, name) {
			logger.Infof("Container %q matched pattern %q (ID %s)", name, pattern, id)
			return id, nil
		}
	}
	return "", fmt.Errorf("no container matching %q", pattern)
}

// matchGlob reports whether name matches pattern. A pattern without '*' is
// treated as a substring match; otherwise '*' matches any sequence of
// characters.
func matchGlob(pattern, name string) bool {
	if !strings.Contains(pattern, "*") {
		return strings.Contains(name, pattern)
	}
	parts := strings.Split(pattern, "*")
	if !strings.HasPrefix(name, parts[0]) {
		return false
	}
	name = name[len(parts[0]):]
	for _, part := range parts[1:] {
		idx := strings.Index(name, part)
		if idx == -1 {
			return false
		}
		name = name[idx+len(part):]
	}
	return true
}

func runCommand(ctx context.Context, cmd *exec.Cmd, cancel context.CancelFunc) (*Stream, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if startErr := cmd.Start(); startErr != nil {
		cancel()
		return nil, fmt.Errorf("start command: %w", startErr)
	}

	stop := func() {
		cancel()
		_ = cmd.Wait()
	}

	stream := startStream(ctx, stdout, stderr, stop)

	go func() {
		_ = cmd.Wait()
		stop()
	}()

	return stream, nil
}

func startStream(ctx context.Context, stdout, stderr io.Reader, stop func()) *Stream {
	lines := make(chan string)
	errs := make(chan error)
	var errGroup errgroup.Group

	errGroup.Go(func() error { return streamLines(ctx, stdout, lines) })
	errGroup.Go(func() error { return streamErrors(ctx, stderr, errs) })

	go func() {
		<-ctx.Done()
		_ = errGroup.Wait()
		close(lines)
		close(errs)
	}()

	return &Stream{
		Lines:  lines,
		Errors: errs,
		Stop:   stop,
	}
}

func streamLines(ctx context.Context, r io.Reader, out chan<- string) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		select {
		case out <- normalizeText(scanner.Text()):
		case <-ctx.Done():
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read log lines: %w", err)
	}
	return nil
}

// normalizeText returns s as valid UTF-8. Lines that are not valid UTF-8 are
// assumed to be windows-1251 (game servers often run with a legacy Russian
// locale) and transcoded accordingly; otherwise the bytes are passed through.
func normalizeText(text string) string {
	if utf8.ValidString(text) {
		return text
	}
	decoded, err := charmap.Windows1251.NewDecoder().Bytes([]byte(text))
	if err != nil {
		return text
	}
	return string(decoded)
}

func streamErrors(ctx context.Context, r io.Reader, out chan<- error) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		text := scanner.Text()
		logger.Errorf("Docker logs stderr: %s", text)
		select {
		case out <- errors.New(text):
		case <-ctx.Done():
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read log errors: %w", err)
	}
	return nil
}
