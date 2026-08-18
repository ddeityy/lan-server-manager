package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ServerTabs is a custom tab bar that renders server tabs plus a "+" tab.
//
// Unlike container.DocTabs it does not add an overflow menu or force every
// tab to be closable, so no reflection or unsafe code is required.
type ServerTabs struct {
	widget.BaseWidget

	items    []*container.TabItem
	selected int

	onSelected func(*container.TabItem)
	onClosed   func(*container.TabItem)
	onAdd      func()
}

// NewServerTabs creates a tab bar with the provided callbacks.
func NewServerTabs(onSelected, onClosed func(*container.TabItem), onAdd func()) *ServerTabs {
	tabs := &ServerTabs{
		onSelected: onSelected,
		onClosed:   onClosed,
		onAdd:      onAdd,
	}
	tabs.ExtendBaseWidget(tabs)
	return tabs
}

// Items returns the current tab items.
func (tabs *ServerTabs) Items() []*container.TabItem {
	return tabs.items
}

// SetItems replaces the full set of tabs and refreshes the bar.
func (tabs *ServerTabs) SetItems(items []*container.TabItem) {
	tabs.items = items
	if tabs.selected >= len(tabs.items) {
		tabs.selected = len(tabs.items) - 1
	}
	if tabs.selected < 0 {
		tabs.selected = 0
	}
	tabs.Refresh()
}

// SelectIndex selects the tab at the given index.
func (tabs *ServerTabs) SelectIndex(index int) {
	if index < 0 || index >= len(tabs.items) {
		return
	}
	if index == tabs.selected {
		return
	}
	tabs.selected = index
	if tabs.onSelected != nil {
		tabs.onSelected(tabs.items[index])
	}
	tabs.Refresh()
}

// SelectedIndex returns the index of the currently selected tab.
func (tabs *ServerTabs) SelectedIndex() int {
	return tabs.selected
}

// Remove removes a tab by value.
func (tabs *ServerTabs) Remove(item *container.TabItem) {
	for i, it := range tabs.items {
		if it == item {
			tabs.items = append(tabs.items[:i], tabs.items[i+1:]...)
			if tabs.selected >= len(tabs.items) {
				tabs.selected = len(tabs.items) - 1
			}
			if tabs.selected < 0 {
				tabs.selected = 0
			}
			tabs.Refresh()
			return
		}
	}
}

// CreateRenderer builds the tab bar renderer.
func (tabs *ServerTabs) CreateRenderer() fyne.WidgetRenderer {
	r := &serverTabsRenderer{tabs: tabs}
	r.bar = container.NewHBox()
	r.divider = canvas.NewRectangle(theme.Color(theme.ColorNameShadow))
	r.updateBar()
	return r
}

type serverTabsRenderer struct {
	tabs    *ServerTabs
	bar     *fyne.Container
	divider *canvas.Rectangle
	size    fyne.Size
}

func (r *serverTabsRenderer) Destroy() {}

func (r *serverTabsRenderer) Objects() []fyne.CanvasObject {
	objs := []fyne.CanvasObject{r.bar, r.divider}
	if sel := r.tabs.SelectedIndex(); sel >= 0 && sel < len(r.tabs.items) {
		objs = append(objs, r.tabs.items[sel].Content)
	}
	return objs
}

func (r *serverTabsRenderer) MinSize() fyne.Size {
	barMin := r.bar.MinSize()
	contentMin := fyne.NewSize(0, 0)
	for _, item := range r.tabs.items {
		if item.Content != nil {
			contentMin = contentMin.Max(item.Content.MinSize())
		}
	}
	return fyne.NewSize(fyne.Max(barMin.Width, contentMin.Width), barMin.Height+contentMin.Height)
}

func (r *serverTabsRenderer) Layout(size fyne.Size) {
	r.size = size
	th := r.tabs.Theme()
	dividerThickness := th.Size(theme.SizeNameSeparatorThickness)
	barHeight := r.bar.MinSize().Height

	r.bar.Move(fyne.NewPos(0, 0))
	r.bar.Resize(fyne.NewSize(size.Width, barHeight))

	r.divider.Move(fyne.NewPos(0, barHeight))
	r.divider.Resize(fyne.NewSize(size.Width, dividerThickness))

	if sel := r.tabs.SelectedIndex(); sel >= 0 && sel < len(r.tabs.items) {
		contentPos := fyne.NewPos(0, barHeight+dividerThickness)
		contentSize := fyne.NewSize(size.Width, size.Height-barHeight-dividerThickness)
		item := r.tabs.items[sel]
		item.Content.Move(contentPos)
		item.Content.Resize(contentSize)
	}
}

func (r *serverTabsRenderer) Refresh() {
	r.divider.FillColor = theme.Color(theme.ColorNameShadow)
	r.divider.Refresh()
	r.updateBar()
	if !r.size.IsZero() {
		r.Layout(r.size)
	}
}

func (r *serverTabsRenderer) updateBar() {
	r.bar.Objects = nil
	items := r.tabs.items
	// The last item is the "+" tab and is never closable.
	serverCount := len(items) - 1
	closable := r.tabs.onClosed != nil && serverCount > 1

	for i, item := range items {
		idx := i
		isPlus := i == len(items)-1
		btn := newTabButton(item.Text, i == r.tabs.selected)

		if isPlus {
			btn.onTapped = func() {
				if r.tabs.onAdd != nil {
					r.tabs.onAdd()
				}
			}
		} else {
			btn.onTapped = func() { r.tabs.SelectIndex(idx) }
			if closable {
				btn.onClosed = func() {
					if r.tabs.onClosed != nil {
						r.tabs.onClosed(items[idx])
					}
				}
			}
		}
		r.bar.Add(btn)
	}
	r.bar.Refresh()
}

// tabButton renders a single tab in the tab bar.
//
// It is backed by a standard widget.Button so hover/focus effects work reliably,
// with a custom label, selection indicator, and optional close button layered on top.
type tabButton struct {
	widget.BaseWidget

	text     string
	selected bool

	onTapped func()
	onClosed func()
}

func newTabButton(text string, selected bool) *tabButton {
	b := &tabButton{text: text, selected: selected}
	b.ExtendBaseWidget(b)
	return b
}

func (b *tabButton) CreateRenderer() fyne.WidgetRenderer {
	currentTheme := b.Theme()
	variant := fyne.CurrentApp().Settings().ThemeVariant()

	// The standard button provides hover/tap feedback and hit testing.
	button := widget.NewButton("", func() {
		if b.onTapped != nil {
			b.onTapped()
		}
	})
	button.Importance = widget.LowImportance

	indicator := canvas.NewRectangle(currentTheme.Color(theme.ColorNamePrimary, variant))
	indicator.Hide()

	label := canvas.NewText(b.text, currentTheme.Color(theme.ColorNameForeground, variant))
	label.TextStyle.Bold = true
	if strings.Contains(b.text, "＋") {
		label.TextSize = currentTheme.Size(theme.SizeNameText) * 1.4
	}

	objects := []fyne.CanvasObject{button, indicator, label}
	var closeBtn *widget.Button
	if b.onClosed != nil {
		closeBtn = widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
			if b.onClosed != nil {
				b.onClosed()
			}
		})
		closeBtn.Importance = widget.LowImportance
		objects = append(objects, closeBtn)
	}

	return &tabButtonRenderer{
		button:    b,
		bg:        button,
		indicator: indicator,
		label:     label,
		closeBtn:  closeBtn,
		objects:   objects,
	}
}

type tabButtonRenderer struct {
	button    *tabButton
	bg        *widget.Button
	indicator *canvas.Rectangle
	label     *canvas.Text
	closeBtn  *widget.Button
	objects   []fyne.CanvasObject
}

func (r *tabButtonRenderer) Destroy() {}

func (r *tabButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *tabButtonRenderer) MinSize() fyne.Size {
	currentTheme := r.button.Theme()
	padding := currentTheme.Size(theme.SizeNameInnerPadding) + currentTheme.Size(theme.SizeNamePadding)
	labelSize := r.label.MinSize()
	width := labelSize.Width + padding*2
	height := labelSize.Height + padding

	if r.closeBtn != nil {
		closeSize := currentTheme.Size(theme.SizeNameInlineIcon)
		width += closeSize + padding
		if closeSize > height-padding {
			height = closeSize + padding
		}
	}
	return fyne.NewSize(width, height)
}

func (r *tabButtonRenderer) Layout(size fyne.Size) {
	currentTheme := r.button.Theme()
	pad := currentTheme.Size(theme.SizeNameInnerPadding) + currentTheme.Size(theme.SizeNamePadding)

	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(size)

	indicatorHeight := currentTheme.Size(theme.SizeNameSelectionRadius)
	r.indicator.Move(fyne.NewPos(0, size.Height-indicatorHeight))
	r.indicator.Resize(fyne.NewSize(size.Width, indicatorHeight))

	labelWidth := size.Width - pad*2
	if r.closeBtn != nil {
		inlineIconSize := currentTheme.Size(theme.SizeNameInlineIcon)
		labelWidth -= inlineIconSize + pad
	}
	r.label.Move(fyne.NewPos(pad, (size.Height-r.label.MinSize().Height)/2))
	r.label.Resize(fyne.NewSize(labelWidth, r.label.MinSize().Height))

	if r.closeBtn != nil {
		inlineIconSize := currentTheme.Size(theme.SizeNameInlineIcon)
		r.closeBtn.Move(fyne.NewPos(size.Width-inlineIconSize-pad, (size.Height-inlineIconSize)/2))
		r.closeBtn.Resize(fyne.NewSquareSize(inlineIconSize))
	}
}

func (r *tabButtonRenderer) Refresh() {
	currentTheme := r.button.Theme()
	variant := fyne.CurrentApp().Settings().ThemeVariant()

	if r.button.selected {
		r.label.Color = currentTheme.Color(theme.ColorNamePrimary, variant)
		r.indicator.FillColor = currentTheme.Color(theme.ColorNamePrimary, variant)
		r.indicator.Show()
	} else {
		r.label.Color = currentTheme.Color(theme.ColorNameForeground, variant)
		r.indicator.Hide()
	}

	r.bg.Refresh()
	r.indicator.Refresh()

	r.label.Text = r.button.text
	r.label.Refresh()

	if r.closeBtn != nil {
		r.closeBtn.Refresh()
	}
}
