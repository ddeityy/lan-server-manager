package rcon

import (
	"fmt"
	"net"
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
	Address   string
}

// SourceTV holds the parsed SourceTV line from the status output.
type SourceTV struct {
	Address string
	Delay   string
	Local   string
}

// Address holds the addresses reported by the Source engine status command:
// the reported UDP/IP endpoint and the local bind.
type Address struct {
	IP    string
	Local string
}

// ServerInfo holds the parsed output of the rcon "status" command.
type ServerInfo struct {
	Hostname          string
	ConfiguredAddress string
	Address           Address
	Map               string
	SourceTV          SourceTV
	HumanPlayers      int
	MaxPlayers        int
	Players           []Player
}

// GameConnectAddress returns the best address to give to TF2 clients that want
// to connect to the game rcon.
//
// The reported UDP/IP address from status is preferred. Fallbacks are only
// used when the server reports a placeholder such as ?.?.?.?:?.
func (i ServerInfo) GameConnectAddress() string {
	if AddressIsValid(i.Address.IP) {
		return i.Address.IP
	}
	if AddressIsValid(i.Address.Local) {
		return i.Address.Local
	}
	if AddressIsValid(i.ConfiguredAddress) {
		return i.ConfiguredAddress
	}
	return ""
}

// STVConnectAddress returns the best SourceTV connect address.
//
// The server's reported STV address is preferred. If it is unavailable we
// derive one from the configured game host and the STV port reported by status.
func (i ServerInfo) STVConnectAddress() string {
	if AddressIsValid(i.SourceTV.Address) {
		return i.SourceTV.Address
	}
	if AddressIsValid(i.SourceTV.Local) {
		return i.SourceTV.Local
	}
	if !AddressIsValid(i.ConfiguredAddress) {
		return ""
	}
	host, _, err := net.SplitHostPort(i.ConfiguredAddress)
	if err != nil {
		return ""
	}

	stvPort := ""
	if i.SourceTV.Local != "" {
		if _, port, err := net.SplitHostPort(i.SourceTV.Local); err == nil && port != "" && !strings.Contains(port, "?") {
			stvPort = port
		}
	}
	if stvPort == "" && i.SourceTV.Address != "" {
		if _, port, err := net.SplitHostPort(i.SourceTV.Address); err == nil && port != "" && !strings.Contains(port, "?") {
			stvPort = port
		}
	}
	if stvPort == "" {
		return ""
	}
	return net.JoinHostPort(host, stvPort)
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

	info.ConfiguredAddress = s.address

	// Fall back to the configured address if the server did not report a usable one.
	if !AddressIsValid(info.Address.IP) {
		info.Address.IP = s.address
	}
	if info.Address.Local == "" {
		info.Address.Local = s.address
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

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "hostname"):
			if _, after, ok := strings.Cut(line, ":"); ok {
				info.Hostname = stripQuotes(strings.TrimSpace(after))
			}

		case strings.HasPrefix(line, "udp/ip"):
			if m := regexp.MustCompile(`udp/ip\s*:\s*(\S+(?::\d+)?)\s*(?:\(local:\s*([^)]+)\))?`).FindStringSubmatch(line); len(m) > 1 {
				info.Address.IP = m[1]
				if len(m) > 2 {
					info.Address.Local = m[2]
				}
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

		case strings.HasPrefix(line, "sourcetv"):
			if m := regexp.MustCompile(`sourcetv:\s*([^,]+),\s*delay\s+([0-9.]+s)(?:\s*\(local:\s*([^)]+)\))?`).FindStringSubmatch(line); len(m) > 2 {
				info.SourceTV.Address = m[1]
				info.SourceTV.Delay = m[2]
				if len(m) > 3 {
					info.SourceTV.Local = m[3]
				}
			}

		case strings.HasPrefix(line, "players"):
			info.HumanPlayers, info.MaxPlayers = parsePlayersLine(line)
		}
	}

	info.Players = parsePlayersTable(lines)

	return info, nil
}

func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	if len(s) >= 1 && (s[0] == '"' || s[0] == '\'') {
		return s[1:]
	}
	if len(s) >= 1 && (s[len(s)-1] == '"' || s[len(s)-1] == '\'') {
		return s[:len(s)-1]
	}
	return s
}

// AddressIsValid reports whether an address string is a real endpoint and not
// an empty or unknown placeholder like "?.?.?.?:?".
func AddressIsValid(s string) bool {
	if s == "" {
		return false
	}
	host, _, err := net.SplitHostPort(s)
	if err != nil {
		host = s
	}
	return !strings.Contains(host, "?")
}

func parsePlayersLine(line string) (humans, max int) {
	// "players : 0 humans, 1 bots (25 max)"
	if m := regexp.MustCompile(`(\d+)\s+humans`).FindStringSubmatch(line); len(m) > 1 {
		humans, _ = strconv.Atoi(m[1])
	}
	if m := regexp.MustCompile(`\((\d+)\s+max\)`).FindStringSubmatch(line); len(m) > 1 {
		max, _ = strconv.Atoi(m[1])
	}
	return
}

func parsePlayersTable(lines []string) []Player {
	var players []Player

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if m := regexp.MustCompile(`^#\s+(\d+)\s+"([^"]*)"\s+(\S+)\s+(\S+)\s+(\d+)\s+(\d+)\s+(\S+)(?:\s+(\S+))?`).FindStringSubmatch(line); m != nil {
			uniqueID := m[3]
			// Skip bot entries.
			if uniqueID == "BOT" {
				continue
			}
			id, _ := strconv.Atoi(m[1])
			ping, _ := strconv.Atoi(m[5])
			loss, _ := strconv.Atoi(m[6])
			address := ""
			if len(m) > 8 {
				address = m[8]
			}
			players = append(players, Player{
				UserID:    id,
				Name:      m[2],
				UniqueID:  uniqueID,
				Connected: m[4],
				Ping:      ping,
				Loss:      loss,
				State:     m[7],
				Address:   address,
			})
		}
	}

	return players
}
