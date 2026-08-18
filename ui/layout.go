package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
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
	var maxHeight, totalW float32
	for idx, obj := range objects {
		if minH := obj.MinSize().Height; minH > maxHeight {
			maxHeight = minH
		}
		totalW += obj.MinSize().Width
		if idx < len(objects)-1 {
			totalW += actionRowGap
		}
	}
	return fyne.NewSize(totalW, maxHeight)
}

func newActionRow(left, right fyne.CanvasObject) *fyne.Container {
	return container.New(&actionRowLayout{}, left, right)
}

// setButtonsEnabled enables or disables a slice of buttons.
func setButtonsEnabled(enabled bool, buttons ...*widget.Button) {
	for _, b := range buttons {
		if enabled {
			b.Enable()
		} else {
			b.Disable()
		}
	}
}
