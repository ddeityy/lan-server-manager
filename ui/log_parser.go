package ui

import (
	"fmt"
	"regexp"
	"strings"
)

// logPrefixLen is the length of the leading "L 08/14/2026 - 11:49:16: " prefix
// that game log lines start with.
const logPrefixLen = 25

// chatMessage holds the parsed pieces of a Source engine say line.
type chatMessage struct {
	Time    string
	Team    string
	Name    string
	Message string
}

// logParser filters and formats Source engine log lines.
type logParser struct {
	sayRE *regexp.Regexp
}

// newLogParser creates a parser with a compiled chat-line regex.
func newLogParser() *logParser {
	return &logParser{
		sayRE: regexp.MustCompile(`^.*:\s*"([^"<]*)<\d+>.*?<([^>]+)>"\s+say\s+"([^"]*)"`),
	}
}

// isGameLogLine reports whether line starts with the Source engine log prefix.
func (p *logParser) isGameLogLine(line string) bool {
	if len(line) < logPrefixLen {
		return false
	}
	return line[0] == 'L' &&
		line[1] == ' ' &&
		line[12] == ' ' &&
		line[13] == '-' &&
		line[14] == ' ' &&
		line[23] == ':' &&
		line[24] == ' '
}

// isChatLogLine reports whether line contains a player say command.
func (p *logParser) isChatLogLine(line string) bool {
	return strings.Contains(line, `say "`)
}

// parseChat extracts team, name, and message from a say line. If the line does
// not match a say command, an error is returned.
func (p *logParser) parseChat(line string) (chatMessage, error) {
	m := p.sayRE.FindStringSubmatch(line)
	if len(m) != 4 {
		return chatMessage{}, fmt.Errorf("no match: %q", line)
	}

	var team string
	switch strings.ToLower(m[2]) {
	case "red":
		team = "RED"
	case "blue":
		team = "BLU"
	case "console":
		team = "CON"
	case "spectator":
		team = "SPC"
	default:
		return chatMessage{}, fmt.Errorf("unknown team %q in line: %q", m[2], line)
	}

	msg := chatMessage{
		Time:    line[15:23],
		Team:    team,
		Name:    m[1],
		Message: m[3],
	}
	return msg, nil
}

// formatLogLine strips the leading "L <date> - " prefix and keeps the time
// followed by the log message. Non-game log lines are returned unchanged.
func (p *logParser) formatLogLine(line string) string {
	if !p.isGameLogLine(line) {
		return line
	}
	return line[15:23] + ": " + line[logPrefixLen:]
}

// chatDisplayText returns the plain-text representation of a chat message.
func (m chatMessage) String() string {
	return fmt.Sprintf("%s %s: %s: %s", m.Time, m.Team, m.Name, m.Message)
}
