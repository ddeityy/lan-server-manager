package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/server"
)

// PlayerSection holds the player table and the "kick all" action.
type PlayerSection struct {
	kickAllButton    *widget.Button
	playerList       *widget.List
	playersAccordion *widget.Accordion
}

func newPlayerSection(onKickAll func()) *PlayerSection {
	ps := &PlayerSection{}

	ps.kickAllButton = widget.NewButton("Kick All Players", onKickAll)
	ps.kickAllButton.Disable()

	ps.playerList = newPlayerList()
	ps.playersAccordion = widget.NewAccordion(
		widget.NewAccordionItem("Players", container.NewBorder(playerHeader(), nil, nil, nil, ps.playerList)),
	)
	ps.playersAccordion.Open(0)

	return ps
}

// View returns the kick-all button and player accordion.
func (ps *PlayerSection) View() fyne.CanvasObject {
	return container.NewBorder(ps.kickAllButton, nil, nil, nil, ps.playersAccordion)
}

// Update refreshes the player table and accordion title.
func (ps *PlayerSection) Update(players []server.Player, kick func(int)) {
	updatePlayerList(ps.playerList, players, kick)
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
