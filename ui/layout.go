package ui

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/config"
)

// mapList returns the configured map list, falling back to the compiled-in
// defaults when no runtime config is loaded.
func mapList() []string {
	if len(appConfig.Maps) > 0 {
		return appConfig.Maps
	}
	return config.Default().Maps
}

// configList returns the configured exec config list, falling back to the
// compiled-in defaults when no runtime config is loaded.
func configList() []string {
	if len(appConfig.Configs) > 0 {
		return appConfig.Configs
	}
	return config.Default().Configs
}

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

func withMinWidth(obj fyne.CanvasObject, width float32) fyne.CanvasObject {
	return container.New(&minWidthLayout{width: width}, obj)
}

// setMapSelection sets the map dropdown's current value and renders the
// remaining pool options so the currently selected map is not shown twice.
func setMapSelection(sel *widget.Select, value string) {
	maps := mapList()
	valid := slices.Contains(maps, value)
	if !valid {
		sel.Selected = ""
		sel.Options = append([]string(nil), maps...)
		sel.Refresh()
		return
	}

	opts := make([]string, 0, len(maps)-1)
	for _, m := range maps {
		if m != value {
			opts = append(opts, m)
		}
	}
	sel.Selected = value
	sel.Options = opts
	sel.Refresh()
}
