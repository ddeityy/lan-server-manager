package ui

import (
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (p *ServerPanel) buildUI(title string) {
	sidebar := container.NewVBox(
		widget.NewCard("RCON Connection", "", p.connection.View()),
		widget.NewCard("RCON Actions", "", p.actions.View()),
	)

	horizontalSplit := container.NewHSplit(
		withMinWidth(sidebar, sidebarMinWidth),
		p.serverInfo.View(),
	)
	horizontalSplit.Offset = 0.35

	verticalSplit := container.NewVSplit(
		horizontalSplit,
		p.players.View(),
	)
	verticalSplit.Offset = 0.65

	p.tabItem = container.NewTabItem(title, verticalSplit)
}
