package rcon

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	gorcon "github.com/gorcon/rcon"

	"lan-server-manager/internal/logger"
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

// Client wraps an RCON connection to a Source engine game rcon.
type Client struct {
	address  string
	password string
	conn     *gorcon.Conn
	lastInfo ServerInfo
}

// NewClient creates a Client without connecting.
func NewClient(address, password string) *Client {
	logger.Infof("Created RCON client for %s", address)
	return &Client{
		address:  address,
		password: password,
	}
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

// Close releases the RCON connection.
func (s *Client) Close() error {
	logger.Infof("Closing RCON connection to %s", s.address)
	if s.conn != nil {
		err := s.conn.Close()
		s.conn = nil
		return err
	}
	return nil
}

// Kick removes a player from the server by their user ID.
func (s *Client) Kick(userID int) error {
	logger.Infof("Kicking player %d on %s", userID, s.address)
	_, err := s.send(fmt.Sprintf("kickid %d", userID))
	if err != nil {
		logger.Errorf("Kick player %d on %s failed: %v", userID, s.address, err)
	}
	return err
}

// Execute sends an arbitrary RCON command to the rcon.
func (s *Client) Execute(cmd string) error {
	logger.Infof("Executing RCON command on %s: %s", s.address, cmd)
	return s.execute(cmd)
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

func (s *Client) execute(cmd string) error {
	_, err := s.send(cmd)
	return err
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
		// Connection may have dropped; try once more with a fresh connection.
		if err := s.Close(); err != nil {
			logger.Warnf("RCON close on %s failed: %v", s.address, err)
		}
		if err := s.Connect(); err != nil {
			return "", err
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

	// logger.Infof("Refreshing status on %s", s.address)
	resp, err := s.send("status")
	if err != nil {
		return err
	}

	info, err := ParseStatus(resp)
	if err != nil {
		logger.Errorf("Failed to parse status from %s: %v", s.address, err)
		return fmt.Errorf("parse status: %w", err)
	}

	s.lastInfo = info
	// logger.Infof("Status refreshed on %s: hostname=%q map=%q players=%d/%d", s.address, info.Hostname, info.Map, info.HumanPlayers, info.MaxPlayers)
	return nil
}

// Info returns the most recently parsed server information.
func (s *Client) Info() ServerInfo {
	return s.lastInfo
}

// ParseStatus extracts the fields we need from the Source engine status text.
func ParseStatus(status string) (ServerInfo, error) {
	// logger.Infof("Parsing status output (%d bytes)", len(status))
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
				s := strings.TrimSpace(after)
				if len(s) >= 2 {
					if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
						s = s[1 : len(s)-1]
					}
				} else if len(s) >= 1 && (s[0] == '"' || s[0] == '\'') {
					s = s[1:]
				} else if len(s) >= 1 && (s[len(s)-1] == '"' || s[len(s)-1] == '\'') {
					s = s[:len(s)-1]
				}
				info.Hostname = s
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
		if m := rePlayer.FindStringSubmatch(line); m != nil {
			uniqueID := m[3]
			id, _ := strconv.Atoi(m[1])
			ping, _ := strconv.Atoi(m[5])
			loss, _ := strconv.Atoi(m[6])
			info.Players = append(info.Players, Player{
				UserID:    id,
				Name:      m[2],
				UniqueID:  uniqueID,
				Connected: m[4],
				Ping:      ping,
				Loss:      loss,
				State:     m[7],
			})
		}
	}

	// logger.Infof("Parsed status: hostname=%q map=%q players=%d/%d human_players=%d", info.Hostname, info.Map, len(info.Players), info.MaxPlayers, info.HumanPlayers)
	return info, nil
}
