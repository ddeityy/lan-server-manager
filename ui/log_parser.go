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

func chatColor(team logparse.Team) color.Color {
	switch team {
	case logparse.TeamRed:
		return color.NRGBA{R: 167, G: 88, B: 75, A: 220}
	case logparse.TeamBlu:
		return color.NRGBA{R: 84, G: 125, B: 140, A: 220}
	case logparse.TeamSpec:
		return color.NRGBA{R: 160, G: 160, B: 160, A: 255}
	default:
		return color.NRGBA{R: 160, G: 160, B: 160, A: 255}
	}
}

// chatFromLogEvent converts a parsed chat event into a display-ready chatMessage.
func chatFromLogEvent(evt logparse.Event) (chatMessage, bool) {
	if evt.Type != logparse.EventChat && evt.Type != logparse.EventChatTeam {
		return chatMessage{}, false
	}

	return chatMessage{
		Time:    evt.Timestamp.Format(time.TimeOnly),
		Color:   chatColor(evt.Source.Team),
		Name:    evt.Source.Name,
		Message: evt.Data["message"],
	}, true
}
