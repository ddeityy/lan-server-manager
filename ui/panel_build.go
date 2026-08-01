package ui

import (
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (p *ServerPanel) buildUI(title string) {
	sidebar := container.NewVBox(
		widget.NewCard("Connection", "", p.connection.View()),
		widget.NewCard("Actions", "", p.actions.View()),
	)

	topRow := container.NewBorder(nil, nil, withMinWidth(sidebar, sidebarMinWidth), nil, p.serverInfo.View())
	bottomRow := p.players.View()

	content := container.NewBorder(topRow, nil, nil, nil, bottomRow)
	p.tabItem = container.NewTabItem(title, content)
}
