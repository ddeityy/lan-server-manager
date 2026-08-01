package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/rcon"
)

// Player list column minimum widths.
var playerColumnWidths = []float32{50, 200, 160, 65, 55, 55, 80, 200, 75}

// Player list flex weights.
var playerColumnFlex = []int{0, 3, 1, 0, 0, 0, 0, 3, 0}

const playerColumnGap = float32(8)

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
				fixedLabel(),
				widget.NewButton("Kick", nil),
			)
		},
		func(_ widget.ListItemID, _ fyne.CanvasObject) {},
	)

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
		row.Objects[7].(*widget.Label).SetText(player.Address)

		btn := row.Objects[8].(*widget.Button)
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

// playerRowLayout lays out player table cells at fixed minimum widths, letting
// flexible columns absorb any extra horizontal space when the window grows.
type playerRowLayout struct{}

func (l *playerRowLayout) totalFlexWeight() int {
	total := 0
	for _, w := range playerColumnFlex {
		total += w
	}
	return total
}

func (l *playerRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	minWidth := l.MinSize(objects).Width
	extra := size.Width - minWidth
	if extra < 0 {
		extra = 0
	}
	flexWeight := l.totalFlexWeight()

	x := float32(0)
	for i, obj := range objects {
		w := playerColumnWidths[i]
		if flexWeight > 0 && playerColumnFlex[i] > 0 {
			w += extra * float32(playerColumnFlex[i]) / float32(flexWeight)
		}
		obj.Move(fyne.NewPos(x, 0))
		obj.Resize(fyne.NewSize(w, size.Height))
		x += w
		if i < len(objects)-1 {
			x += playerColumnGap
		}
	}
}

func (l *playerRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	totalWidth := float32(0)
	maxHeight := float32(0)
	for i, obj := range objects {
		totalWidth += playerColumnWidths[i]
		if i < len(objects)-1 {
			totalWidth += playerColumnGap
		}
		if h := obj.MinSize().Height; h > maxHeight {
			maxHeight = h
		}
	}
	return fyne.NewSize(totalWidth, maxHeight)
}

func fixedLabel() *widget.Label {
	lbl := widget.NewLabel("")
	lbl.Wrapping = fyne.TextWrapOff
	lbl.Truncation = fyne.TextTruncateEllipsis
	return lbl
}

func playerHeader() fyne.CanvasObject {
	headerLabel := func(text string) *widget.Label {
		lbl := widget.NewLabel(text)
		lbl.Wrapping = fyne.TextWrapOff
		lbl.Truncation = fyne.TextTruncateEllipsis
		lbl.TextStyle = fyne.TextStyle{Bold: true}
		return lbl
	}

	return container.New(
		&playerRowLayout{},
		headerLabel("ID"),
		headerLabel("Name"),
		headerLabel("UniqueID"),
		headerLabel("Conn"),
		headerLabel("Ping"),
		headerLabel("Loss"),
		headerLabel("State"),
		headerLabel("Adr"),
		headerLabel(""),
	)
}
