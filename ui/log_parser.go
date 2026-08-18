package ui

import (
	"image/color"
	"time"

	"lan-server-manager/game/logparse"
)

// chatMessage holds the parsed pieces of a Source engine say line.
type chatMessage struct {
	Time    string
	Color   color.Color
	Name    string
	Message string
}

var (
	ColorRed       color.Color = color.NRGBA{R: 167, G: 88, B: 75, A: 220}
	ColorBlue      color.Color = color.NRGBA{R: 84, G: 125, B: 140, A: 220}
	ColorConsole   color.Color = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
	ColorSpectator color.Color = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
)

// chatFromLogEvent converts a parsed chat event into a display-ready chatMessage.
func chatFromLogEvent(evt logparse.Event) (chatMessage, bool) {
	if evt.Type != logparse.EventChat && evt.Type != logparse.EventChatTeam {
		return chatMessage{}, false
	}

	var c color.Color
	switch evt.Source.Team {
	case logparse.TeamRed:
		c = ColorRed
	case logparse.TeamBlu:
		c = ColorBlue
	case logparse.TeamSpec:
		c = ColorSpectator
	default:
		c = ColorConsole
	}

	return chatMessage{
		Time:    evt.Timestamp.Format(time.TimeOnly),
		Color:   c,
		Name:    evt.Source.Name,
		Message: evt.Data["message"],
	}, true
}
