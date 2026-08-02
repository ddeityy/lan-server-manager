package rcon

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	gorcon "github.com/gorcon/rcon"
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
	return &Client{
		address:  address,
		password: password,
	}
}

// Connect dials the RCON endpoint and stores the connection.
func (s *Client) Connect() error {
	conn, err := gorcon.Dial(s.address, s.password)
	if err != nil {
		return fmt.Errorf("rcon dial: %w", err)
	}
	s.conn = conn
	return nil
}

// Close releases the RCON connection.
func (s *Client) Close() error {
	if s.conn != nil {
		err := s.conn.Close()
		s.conn = nil
		return err
	}
	return nil
}

// Kick removes a player from the server by their user ID.
func (s *Client) Kick(userID int) error {
	_, err := s.send(fmt.Sprintf("kickid %d", userID))
	return err
}

// Execute sends an arbitrary RCON command to the rcon.
func (s *Client) Execute(cmd string) error {
	return s.execute(cmd)
}

// ChangeLevel sends the Source changelevel command for the given map.
func (s *Client) ChangeLevel(level string) error {
	return s.execute("changelevel " + level)
}

// SetPassword sends the Source sv_password command for the given password.
func (s *Client) SetPassword(password string) error {
	return s.execute("sv_password " + password)
}

// ExecConfig sends the Source exec command for the given config name.
func (s *Client) ExecConfig(config string) error {
	return s.execute("exec " + config)
}

func (s *Client) execute(cmd string) error {
	_, err := s.send(cmd)
	return err
}

// send executes a single RCON command, reconnecting once on failure.
func (s *Client) send(cmd string) (string, error) {
	if strings.TrimSpace(cmd) == "" {
		return "", fmt.Errorf("empty command")
	}
	if s.conn == nil {
		return "", fmt.Errorf("not connected")
	}

	resp, err := s.conn.Execute(cmd)
	if err != nil {
		// Connection may have dropped; try once more with a fresh connection.
		s.Close()
		if err := s.Connect(); err != nil {
			return "", err
		}
		resp, err = s.conn.Execute(cmd)
		if err != nil {
			return "", fmt.Errorf("rcon execute: %w", err)
		}
	}
	return resp, nil
}

// Refresh runs the rcon "status" command and parses the response.
func (s *Client) Refresh() error {
	if s.conn == nil {
		if err := s.Connect(); err != nil {
			return err
		}
	}

	resp, err := s.send("status")
	if err != nil {
		return err
	}

	info, err := ParseStatus(resp)
	if err != nil {
		return fmt.Errorf("parse status: %w", err)
	}

	s.lastInfo = info
	return nil
}

// Info returns the most recently parsed server information.
func (s *Client) Info() ServerInfo {
	return s.lastInfo
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
			// Skip bot entries.
			if uniqueID == "BOT" {
				continue
			}
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

	return info, nil
}
