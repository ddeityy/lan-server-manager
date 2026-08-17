package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

const actionRowGap = float32(8)

// actionRowLayout sizes two children in a 3:1 ratio (75% input / 25% button)
// with a small gap between them.
type actionRowLayout struct{}

func (l *actionRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 2 {
		return
	}
	available := size.Width - actionRowGap
	leftW := available * 0.75
	rightW := available * 0.25
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(leftW, size.Height))
	objects[1].Move(fyne.NewPos(leftW+actionRowGap, 0))
	objects[1].Resize(fyne.NewSize(rightW, size.Height))
}

func (l *actionRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var h, totalW float32
	for i, obj := range objects {
		if mh := obj.MinSize().Height; mh > h {
			h = mh
		}
		totalW += obj.MinSize().Width
		if i < len(objects)-1 {
			totalW += actionRowGap
		}
	}
	return fyne.NewSize(totalW, h)
}

func newActionRow(left, right fyne.CanvasObject) *fyne.Container {
	return container.New(&actionRowLayout{}, left, right)
}

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
