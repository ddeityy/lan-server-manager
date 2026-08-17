package ui

import (
	"fmt"
	"regexp"
	"strings"
)

// logPrefixLen is the length of the leading "L 08/14/2026 - 11:49:16: " prefix
// that game log lines start with.
const logPrefixLen = 25

// logParser filters and formats Source engine log lines.
type logParser struct {
	chatMessageRegex *regexp.Regexp
}

// newLogParser creates a parser with a compiled chat-line regex.
func newLogParser() *logParser {
	return &logParser{
		chatMessageRegex: regexp.MustCompile(`^.*:\s*"([^"<]*)<\d+>.*?<([^>]+)>"\s+say\s+"([^"]*)"`),
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

// parseChat extracts team, name, and message from a say line and returns a
// formatted chat string. If the line does not match a say command, an error is
// returned.
func (p *logParser) parseChat(line string) (string, error) {
	m := p.chatMessageRegex.FindStringSubmatch(line)
	if len(m) != 4 {
		return "", fmt.Errorf("no match: %q", line)
	}

	name := m[1]
	rawTeam := strings.ToLower(m[2])
	msg := m[3]

	var team string
	switch rawTeam {
	case "red":
		team = "RED"
	case "blue":
		team = "BLU"
	case "console":
		team = "CON"
	case "spectator":
		team = "SPC"
	default:
		return "", fmt.Errorf("unknown team %q in line: %q", m[2], line)
	}

	return team + ": " + name + ": " + msg, nil
}

// formatLogLine strips the leading "L <date> - " prefix and keeps the time
// followed by the log message. Non-game log lines are returned unchanged.
func (p *logParser) formatLogLine(line string) string {
	if !p.isGameLogLine(line) {
		return line
	}
	return line[15:23] + ": " + line[logPrefixLen:]
}
