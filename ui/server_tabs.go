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
	t := &ServerTabs{
		onSelected: onSelected,
		onClosed:   onClosed,
		onAdd:      onAdd,
	}
	t.ExtendBaseWidget(t)
	return t
}

// Items returns the current tab items.
func (t *ServerTabs) Items() []*container.TabItem {
	return t.items
}

// SetItems replaces the full set of tabs and refreshes the bar.
func (t *ServerTabs) SetItems(items []*container.TabItem) {
	t.items = items
	if t.selected >= len(t.items) {
		t.selected = len(t.items) - 1
	}
	if t.selected < 0 {
		t.selected = 0
	}
	t.Refresh()
}

// SelectIndex selects the tab at the given index.
func (t *ServerTabs) SelectIndex(index int) {
	if index < 0 || index >= len(t.items) {
		return
	}
	if index == t.selected {
		return
	}
	t.selected = index
	if t.onSelected != nil {
		t.onSelected(t.items[index])
	}
	t.Refresh()
}

// SelectedIndex returns the index of the currently selected tab.
func (t *ServerTabs) SelectedIndex() int {
	return t.selected
}

// Remove removes a tab by value.
func (t *ServerTabs) Remove(item *container.TabItem) {
	for i, it := range t.items {
		if it == item {
			t.items = append(t.items[:i], t.items[i+1:]...)
			if t.selected >= len(t.items) {
				t.selected = len(t.items) - 1
			}
			if t.selected < 0 {
				t.selected = 0
			}
			t.Refresh()
			return
		}
	}
}

// CreateRenderer builds the tab bar renderer.
func (t *ServerTabs) CreateRenderer() fyne.WidgetRenderer {
	r := &serverTabsRenderer{tabs: t}
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

// tabButton renders a single tab in the tab bar. It is backed by a standard
// widget.Button so hover/focus effects work reliably, with a custom label,
// selection indicator, and optional close button layered on top.
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
	th := b.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	// The standard button provides hover/tap feedback and hit testing.
	bg := widget.NewButton("", func() {
		if b.onTapped != nil {
			b.onTapped()
		}
	})
	bg.Importance = widget.LowImportance

	indicator := canvas.NewRectangle(th.Color(theme.ColorNamePrimary, v))
	indicator.Hide()

	label := canvas.NewText(b.text, th.Color(theme.ColorNameForeground, v))
	label.TextStyle.Bold = true
	if strings.Contains(b.text, "＋") {
		label.TextSize = th.Size(theme.SizeNameText) * 1.4
	}

	objects := []fyne.CanvasObject{bg, indicator, label}
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
		bg:        bg,
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
	th := r.button.Theme()
	padding := th.Size(theme.SizeNameInnerPadding) + th.Size(theme.SizeNamePadding)
	labelSize := r.label.MinSize()
	width := labelSize.Width + padding*2
	height := labelSize.Height + padding

	if r.closeBtn != nil {
		closeSize := th.Size(theme.SizeNameInlineIcon)
		width += closeSize + padding
		if closeSize > height-padding {
			height = closeSize + padding
		}
	}
	return fyne.NewSize(width, height)
}

func (r *tabButtonRenderer) Layout(size fyne.Size) {
	th := r.button.Theme()
	pad := th.Size(theme.SizeNameInnerPadding) + th.Size(theme.SizeNamePadding)

	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(size)

	indicatorHeight := th.Size(theme.SizeNameSelectionRadius)
	r.indicator.Move(fyne.NewPos(0, size.Height-indicatorHeight))
	r.indicator.Resize(fyne.NewSize(size.Width, indicatorHeight))

	labelWidth := size.Width - pad*2
	if r.closeBtn != nil {
		inlineIconSize := th.Size(theme.SizeNameInlineIcon)
		labelWidth -= inlineIconSize + pad
	}
	r.label.Move(fyne.NewPos(pad, (size.Height-r.label.MinSize().Height)/2))
	r.label.Resize(fyne.NewSize(labelWidth, r.label.MinSize().Height))

	if r.closeBtn != nil {
		inlineIconSize := th.Size(theme.SizeNameInlineIcon)
		r.closeBtn.Move(fyne.NewPos(size.Width-inlineIconSize-pad, (size.Height-inlineIconSize)/2))
		r.closeBtn.Resize(fyne.NewSquareSize(inlineIconSize))
	}
}

func (r *tabButtonRenderer) Refresh() {
	th := r.button.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	if r.button.selected {
		r.label.Color = th.Color(theme.ColorNamePrimary, v)
		r.indicator.FillColor = th.Color(theme.ColorNamePrimary, v)
		r.indicator.Show()
	} else {
		r.label.Color = th.Color(theme.ColorNameForeground, v)
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
