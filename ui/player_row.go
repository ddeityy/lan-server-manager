package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Player list column minimum widths.
var playerColumnWidths = []float32{50, 200, 160, 65, 55, 55, 80, 75}

// Player list flex weights.
var playerColumnFlex = []int{0, 3, 1, 0, 0, 0, 0, 0}

const playerColumnGap = float32(8)

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
