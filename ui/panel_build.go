package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const sidebarMinWidth = float32(320)

// minWidthLayout wraps a single child and enforces a minimum width.
type minWidthLayout struct {
	width float32
}

func (l *minWidthLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(size)
}

func (l *minWidthLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(l.width, 0)
	}
	s := objects[0].MinSize()
	if s.Width < l.width {
		s.Width = l.width
	}
	return s
}

func (p *ServerPanel) buildUI(title string) {
	left := container.NewVBox(
		p.serverInfo.View(p.connection.View()),
		widget.NewCard("Actions", "", p.actions.View()),
	)
	leftScroll := container.NewVScroll(left)
	leftScroll.SetMinSize(fyne.NewSize(sidebarMinWidth, 0))

	right := container.NewVSplit(p.players.View(), p.logs.View())
	right.Offset = 0.5

	split := container.NewHSplit(leftScroll, right)
	split.Offset = 0.65

	p.tabItem = container.NewTabItem(title, split)
}
