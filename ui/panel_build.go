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
	sidebar := container.NewVBox(
		widget.NewCard("Actions", "", p.actions.View()),
	)

	horizontalSplit := container.NewHSplit(
		container.New(&minWidthLayout{width: sidebarMinWidth}, sidebar),
		p.serverInfo.View(p.connection.View()),
	)
	horizontalSplit.Offset = 0.35

	verticalSplit := container.NewVSplit(
		horizontalSplit,
		p.players.View(),
	)
	verticalSplit.Offset = 0.75

	p.tabItem = container.NewTabItem(title, verticalSplit)
}
