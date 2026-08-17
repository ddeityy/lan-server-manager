package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/rcon"
)

// PlayerSection holds the player table and the "kick all" action.
type PlayerSection struct {
	kickAllButton    *widget.Button
	playerList       *widget.List
	playersAccordion *widget.Accordion
	statusLabel      *widget.Label
}

func newPlayerSection(onKickAll func()) *PlayerSection {
	ps := &PlayerSection{}

	ps.kickAllButton = widget.NewButton("Kick All Players", onKickAll)
	ps.kickAllButton.Disable()

	ps.playerList = widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			return container.New(
				&playerRowLayout{},
				fixedLabel(),
				fixedLabel(),
				fixedLabel(),
				fixedLabel(),
				fixedLabel(),
				fixedLabel(),
				fixedLabel(),
				widget.NewButton("Kick", nil),
			)
		},
		func(_ widget.ListItemID, _ fyne.CanvasObject) {},
	)

	headerLabel := func(text string) *widget.Label {
		lbl := widget.NewLabel(text)
		lbl.Wrapping = fyne.TextWrapOff
		lbl.Truncation = fyne.TextTruncateEllipsis
		lbl.TextStyle = fyne.TextStyle{Bold: true}
		return lbl
	}
	playerHeader := container.New(
		&playerRowLayout{},
		headerLabel("ID"),
		headerLabel("Name"),
		headerLabel("UniqueID"),
		headerLabel("Conn"),
		headerLabel("Ping"),
		headerLabel("Loss"),
		headerLabel("State"),
		headerLabel(""),
	)

	ps.playersAccordion = widget.NewAccordion(
		widget.NewAccordionItem("Players", container.NewBorder(playerHeader, nil, nil, nil, ps.playerList)),
	)
	ps.playersAccordion.Open(0)

	ps.statusLabel = widget.NewLabel("")

	return ps
}

// View returns the kick-all button, player accordion, and status label.
func (ps *PlayerSection) View() fyne.CanvasObject {
	return container.NewBorder(ps.kickAllButton, ps.statusLabel, nil, nil, ps.playersAccordion)
}

// SetStatus updates the status label at the bottom of the player section.
func (ps *PlayerSection) SetStatus(text string) { ps.statusLabel.SetText(text) }

// Update refreshes the player table and accordion title.
func (ps *PlayerSection) Update(players []rcon.Player, kick func(int)) {
	ps.playerList.Length = func() int { return len(players) }
	ps.playerList.UpdateItem = func(i widget.ListItemID, o fyne.CanvasObject) {
		row := o.(*fyne.Container)
		player := players[i]

		row.Objects[0].(*widget.Label).SetText(fmt.Sprintf("%d", player.UserID))
		row.Objects[1].(*widget.Label).SetText(player.Name)
		row.Objects[2].(*widget.Label).SetText(player.UniqueID)
		row.Objects[3].(*widget.Label).SetText(player.Connected)
		row.Objects[4].(*widget.Label).SetText(fmt.Sprintf("%d", player.Ping))
		row.Objects[5].(*widget.Label).SetText(fmt.Sprintf("%d", player.Loss))
		row.Objects[6].(*widget.Label).SetText(player.State)

		btn := row.Objects[7].(*widget.Button)
		btn.SetText("Kick")
		btn.OnTapped = func() { kick(player.UserID) }
		btn.Enable()
	}
	ps.playerList.Refresh()

	ps.playersAccordion.Items[0].Title = fmt.Sprintf("Players (%d)", len(players))
	ps.playersAccordion.Refresh()

	ps.kickAllButton.Disable()
	if len(players) > 0 {
		ps.kickAllButton.Enable()
	}
}

// Reset clears the player list and disables the kick-all button.
func (ps *PlayerSection) Reset() {
	ps.playerList.Length = func() int { return 0 }
	ps.playerList.UpdateItem = func(_ widget.ListItemID, _ fyne.CanvasObject) {}
	ps.playerList.Refresh()
	ps.playersAccordion.Items[0].Title = "Players"
	ps.playersAccordion.Refresh()
	ps.kickAllButton.Disable()
}

// SetEnabled enables or disables the kick-all button wholesale. It does not
// override Update, which re-evaluates the button based on the player count.
func (ps *PlayerSection) SetEnabled(enabled bool) {
	if enabled {
		ps.kickAllButton.Enable()
		return
	}
	ps.kickAllButton.Disable()
}
