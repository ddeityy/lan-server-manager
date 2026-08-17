package ui

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"
	"time"
)

// logPrefixLen is the length of the leading "L 08/14/2026 - 11:49:16: " prefix
// that game log lines start with.
const logPrefixLen = 25

// chatMessage holds the parsed pieces of a Source engine say line.
type chatMessage struct {
	Time    string
	Color   color.Color
	Name    string
	Message string
}

// isServerLogLine reports whether line starts with the Source engine log prefix.
func isServerLogLine(line string) bool {
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
func isChatLogLine(line string) bool {
	return strings.Contains(line, `say "`)
}

var (
	ColorRed       color.Color = color.NRGBA{R: 167, G: 88, B: 75, A: 220}
	ColorBlue      color.Color = color.NRGBA{R: 84, G: 125, B: 140, A: 220}
	ColorConsole   color.Color = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
	ColorSpectator color.Color = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
)

// parseChatLine runs the chat regex against line and returns the captured
// fields. If line does not match a say command, it returns an error.
func parseChatLine(line string) (dateTime, name, team, message string, err error) {
	reg := regexp.MustCompile(`^L (\d{2}/\d{2}/\d{4} - \d{2}:\d{2}:\d{2}):\s*"([^"]*?)<[^>]+><[^>]+><([^>]+)>"\s+say\s+"([^"]*)"`)
	m := reg.FindStringSubmatch(line)
	if len(m) != 5 {
		return "", "", "", "", fmt.Errorf("no match: %q", line)
	}
	return m[1], m[2], m[3], m[4], nil
}

// parseChatMessage extracts team, name, and message from a say line. If the line does
// not match a say command, an error is returned.
func parseChatMessage(line string) (chatMessage, error) {
	dateTime, name, team, message, err := parseChatLine(line)
	if err != nil {
		return chatMessage{}, err
	}

	var c color.Color
	switch strings.ToLower(team) {
	case "red":
		c = ColorRed
	case "blue":
		c = ColorBlue
	case "console":
		c = ColorConsole
	case "spectator":
		c = ColorSpectator
	default:
		c = ColorConsole
	}

	msg := chatMessage{
		Time:    localizeTime(dateTime),
		Color:   c,
		Name:    name,
		Message: message,
	}
	return msg, nil
}

func localizeTime(ts string) string {
	parsed, err := time.Parse("01/02/2006 - 15:04:05", ts)
	if err != nil {
		return ts
	}
	return parsed.In(time.Local).Format(time.TimeOnly)
}

// formatLogLine strips the leading "L <date> - " prefix and keeps the time
// followed by the log message. Non-game log lines are returned unchanged.
func formatLogLine(line string) string {
	if !isServerLogLine(line) {
		return line
	}
	return line[15:23] + ": " + line[logPrefixLen:]
}
