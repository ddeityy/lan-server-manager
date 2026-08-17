package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const sidebarMinWidth = float32(320)

func (p *ServerPanel) buildUI(title string) {
	left := container.NewVBox(
		p.serverInfo.View(p.connection.View()),
		widget.NewCard("Actions", "", p.actions.View()),
	)
	leftScroll := container.NewVScroll(left)
	leftScroll.SetMinSize(fyne.NewSize(sidebarMinWidth, 0))

	chat := container.NewBorder(
		nil, p.sendMessage.View(), nil, nil,
		p.logs.View(),
	)
	right := container.NewVSplit(p.players.View(), chat)
	right.Offset = 0.5

	split := container.NewHSplit(leftScroll, right)
	split.Offset = 0.65

	p.tabItem = container.NewTabItem(title, split)
}
