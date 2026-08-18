package rcon

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	gorcon "github.com/gorcon/rcon"

	"lan-server-manager/logger"
)

// Player holds the fields we care about from the status player table.
type Player struct {
	UserID    int
	Name      string
	UniqueID  string
	Connected string
	Ping      int
	Loss      int
	State     string
}

// ServerInfo holds the parsed output of the rcon "status" command.
type ServerInfo struct {
	Hostname     string
	Map          string
	HumanPlayers int
	MaxPlayers   int
	Players      []Player
}

// request is a single RCON command queued to the client goroutine.
type request struct {
	cmd  string
	resp chan<- result
}

// result carries the response or error back from the client goroutine.
type result struct {
	resp string
	err  error
}

// Client wraps a gorcon RCON connection behind a request channel so all TCP
// traffic is serialized on one goroutine.
type Client struct {
	address  string
	password string
	conn     *gorcon.Conn
	lastInfo ServerInfo

	reqs chan request
	done chan struct{}
	wg   sync.WaitGroup
}

const requestQueueSize = 8

// NewClient creates a Client without connecting and starts the command queue.
func NewClient(address, password string) *Client {
	logger.Infof("Created RCON client for %s", address)
	client := &Client{
		address:  address,
		password: password,
		reqs:     make(chan request, requestQueueSize),
		done:     make(chan struct{}),
	}
	client.wg.Add(1)
	go client.loop()
	return client
}

// Connect dials the RCON endpoint and stores the connection.
func (s *Client) Connect() error {
	logger.Infof("Connecting to RCON at %s", s.address)
	conn, err := gorcon.Dial(s.address, s.password)
	if err != nil {
		logger.Errorf("RCON dial to %s failed: %v", s.address, err)
		return fmt.Errorf("rcon dial: %w", err)
	}
	s.conn = conn
	logger.Infof("RCON connection to %s established", s.address)
	return nil
}

// Close releases the RCON connection and stops the command queue.
func (s *Client) Close() error {
	logger.Infof("Closing RCON connection to %s", s.address)
	close(s.done)
	s.wg.Wait()
	if s.conn != nil {
		err := s.conn.Close()
		s.conn = nil
		if err != nil {
			return fmt.Errorf("close rcon: %w", err)
		}
	}
	return nil
}

func (s *Client) loop() {
	defer s.wg.Done()
	for {
		select {
		case req, ok := <-s.reqs:
			if !ok {
				return
			}
			s.processRequest(req)
		case <-s.done:
			s.drainAndExit()
			return
		}
	}
}

func (s *Client) processRequest(req request) {
	resp, err := s.send(req.cmd)
	select {
	case req.resp <- result{resp: resp, err: err}:
	default:
	}
}

func (s *Client) drainAndExit() {
	for {
		select {
		case req, ok := <-s.reqs:
			if !ok {
				return
			}
			s.processRequest(req)
		default:
			return
		}
	}
}

func (s *Client) execute(cmd string) error {
	_, err := s.ExecuteWithResponse(cmd)
	return err
}

// Execute sends an arbitrary RCON command to the rcon.
func (s *Client) Execute(cmd string) error {
	logger.Infof("Executing RCON command on %s: %s", s.address, cmd)
	return s.execute(cmd)
}

// ExecuteWithResponse sends an arbitrary RCON command and returns the server's
// text response. Useful for commands like "mp_timelimit" whose output is only
// returned over RCON and not echoed to the console log.
func (s *Client) ExecuteWithResponse(cmd string) (string, error) {
	logger.Infof("Executing RCON command on %s: %s", s.address, cmd)
	respC := make(chan result, 1)
	select {
	case s.reqs <- request{cmd: cmd, resp: respC}:
	case <-s.done:
		return "", fmt.Errorf("rcon client closed")
	}
	r := <-respC
	return r.resp, r.err
}

// Kick removes a player from the server by their user ID.
func (s *Client) Kick(userID int) error {
	logger.Infof("Kicking player %d on %s", userID, s.address)
	_, err := s.ExecuteWithResponse(fmt.Sprintf("kickid %d", userID))
	if err != nil {
		logger.Errorf("Kick player %d on %s failed: %v", userID, s.address, err)
	}
	return err
}

// ChangeLevel sends the Source changelevel command for the given map.
func (s *Client) ChangeLevel(level string) error {
	logger.Infof("Changing level on %s to %s", s.address, level)
	return s.execute("changelevel " + level)
}

// SetPassword sends the Source sv_password command for the given password.
func (s *Client) SetPassword(password string) error {
	logger.Infof("Setting server password on %s", s.address)
	return s.execute("sv_password " + password)
}

// ExecConfig sends the Source exec command for the given config name.
func (s *Client) ExecConfig(config string) error {
	logger.Infof("Executing config on %s: %s", s.address, config)
	return s.execute("exec " + config)
}

// send executes a single RCON command, reconnecting once on failure.
func (s *Client) send(cmd string) (string, error) {
	if strings.TrimSpace(cmd) == "" {
		logger.Errorf("Refusing to send empty RCON command to %s", s.address)
		return "", fmt.Errorf("empty command")
	}
	if s.conn == nil {
		logger.Errorf("Not connected to %s", s.address)
		return "", fmt.Errorf("not connected")
	}

	resp, err := s.conn.Execute(cmd)
	if err != nil {
		logger.Warnf("RCON command failed on %s, attempting reconnect: %v", s.address, err)
		// Connection may have dropped; close only the socket and reconnect.
		if s.conn != nil {
			_ = s.conn.Close()
			s.conn = nil
		}
		if connectErr := s.Connect(); connectErr != nil {
			return "", connectErr
		}
		resp, err = s.conn.Execute(cmd)
		if err != nil {
			logger.Errorf("RCON command failed on %s after reconnect: %v", s.address, err)
			return "", fmt.Errorf("rcon execute: %w", err)
		}
	}
	return resp, nil
}

// Refresh runs the rcon "status" command and parses the response.
func (s *Client) Refresh() error {
	if s.conn == nil {
		logger.Infof("No RCON connection for %s, connecting before refresh", s.address)
		if err := s.Connect(); err != nil {
			return err
		}
	}

	resp, err := s.ExecuteWithResponse("status")
	if err != nil {
		return err
	}

	info, err := ParseStatus(resp)
	if err != nil {
		logger.Errorf("Failed to parse status from %s: %v", s.address, err)
		return fmt.Errorf("parse status: %w", err)
	}

	s.lastInfo = info
	return nil
}

// Info returns the most recently parsed server information.
func (s *Client) Info() ServerInfo {
	return s.lastInfo
}

// unquote removes matching surrounding quotes from s.
func unquote(value string) string {
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return value[1 : len(value)-1]
		}
	}
	if len(value) >= 1 {
		if value[0] == '"' || value[0] == '\'' {
			return value[1:]
		}
		if value[len(value)-1] == '"' || value[len(value)-1] == '\'' {
			return value[:len(value)-1]
		}
	}
	return value
}

// ParseStatus extracts the fields we need from the Source engine status text.
func ParseStatus(status string) (ServerInfo, error) {
	var info ServerInfo

	lines := strings.Split(status, "\n")

	reHumans := regexp.MustCompile(`(\d+)\s+humans`)
	reMax := regexp.MustCompile(`\((\d+)\s+max\)`)
	rePlayer := regexp.MustCompile(`^#\s+(\d+)\s+"([^"]*)"\s+(\S+)\s+(\S+)\s+(\d+)\s+(\d+)\s+(\S+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "hostname"):
			if _, after, ok := strings.Cut(line, ":"); ok {
				info.Hostname = unquote(strings.TrimSpace(after))
			}

		case strings.HasPrefix(line, "map"):
			if _, after, ok := strings.Cut(line, ":"); ok {
				value := strings.TrimSpace(after)
				// "cp_badlands at: 0 x, 0 y, 0 z"
				if at := strings.Index(value, " at:"); at != -1 {
					value = value[:at]
				}
				info.Map = value
			}

		case strings.HasPrefix(line, "players"):
			if m := reHumans.FindStringSubmatch(line); len(m) > 1 {
				info.HumanPlayers, _ = strconv.Atoi(m[1])
			}
			if m := reMax.FindStringSubmatch(line); len(m) > 1 {
				info.MaxPlayers, _ = strconv.Atoi(m[1])
			}
		}
	}

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if matches := rePlayer.FindStringSubmatch(line); matches != nil {
			uniqueID := matches[3]
			id, _ := strconv.Atoi(matches[1])
			ping, _ := strconv.Atoi(matches[5])
			loss, _ := strconv.Atoi(matches[6])
			info.Players = append(info.Players, Player{
				UserID:    id,
				Name:      matches[2],
				UniqueID:  uniqueID,
				Connected: matches[4],
				Ping:      ping,
				Loss:      loss,
				State:     matches[7],
			})
		}
	}

	return info, nil
}
