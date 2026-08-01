package ui

import (
	"errors"
	"io"
	"net"
	"strings"

	"github.com/gorcon/rcon"
)

// formatError converts low-level network and RCON errors into short, readable
// status messages. If the error is not recognized, the original text is returned.
func formatError(err error) string {
	if err == nil {
		return ""
	}

	switch {
	case errors.Is(err, rcon.ErrAuthFailed):
		return "RCON password rejected"
	case errors.Is(err, rcon.ErrAuthNotRCON):
		return "Server is not an RCON endpoint"
	case errors.Is(err, rcon.ErrInvalidAuthResponse):
		return "Invalid RCON authentication response"
	case errors.Is(err, rcon.ErrCommandEmpty):
		return "Empty command"
	case errors.Is(err, rcon.ErrCommandTooLong):
		return "Command too long"
	}

	msg := err.Error()
	if msg == "" {
		return "Unknown error"
	}

	switch {
	case isNetworkError(err, "connection refused"):
		return "Connection refused"
	case isNetworkError(err, "connection reset by peer"):
		return "Connection reset by peer"
	case isNetworkError(err, "no route to host"):
		return "No route to host"
	case isNetworkError(err, "network is unreachable"):
		return "Network unreachable"
	case isNetworkError(err, "i/o timeout"):
		return "Server timed out"
	case isNetworkError(err, "broken pipe"):
		return "Connection broken"
	case errors.Is(err, io.EOF):
		return "Connection closed unexpectedly"
	}

	return msg
}

func isNetworkError(err error, text string) bool {
	if _, ok := errors.AsType[net.Error](err); ok {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), strings.ToLower(text))
}
