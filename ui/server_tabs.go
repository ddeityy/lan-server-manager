package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
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

// tabButton renders a single tab in the tab bar.
type tabButton struct {
	widget.BaseWidget

	text     string
	selected bool
	hovered  bool

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

	// Transparent background is always visible so the tab keeps a mouse-hit
	// area for hover events even when the hover fill is not shown.
	background := canvas.NewRectangle(color.Transparent)
	background.CornerRadius = th.Size(theme.SizeNameSelectionRadius)

	indicator := canvas.NewRectangle(th.Color(theme.ColorNamePrimary, v))
	indicator.Hide()

	label := canvas.NewText(b.text, th.Color(theme.ColorNameForeground, v))
	label.TextStyle.Bold = true

	objects := []fyne.CanvasObject{background, label, indicator}
	var closeBtn *tabCloseButton
	if b.onClosed != nil {
		closeBtn = newTabCloseButton(func() { b.onClosed() })
		closeBtn.Hide()
		objects = append(objects, closeBtn)
	}

	return &tabButtonRenderer{
		button:     b,
		background: background,
		indicator:  indicator,
		label:      label,
		closeBtn:   closeBtn,
		objects:    objects,
	}
}

func (b *tabButton) Tapped(*fyne.PointEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

func (b *tabButton) MouseIn(*desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
}

func (b *tabButton) MouseOut() {
	b.hovered = false
	b.Refresh()
}

type tabButtonRenderer struct {
	button     *tabButton
	background *canvas.Rectangle
	indicator  *canvas.Rectangle
	label      *canvas.Text
	closeBtn   *tabCloseButton
	objects    []fyne.CanvasObject
}

func (r *tabButtonRenderer) Destroy() {}

func (r *tabButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *tabButtonRenderer) MinSize() fyne.Size {
	th := r.button.Theme()
	padding := th.Size(theme.SizeNameInnerPadding)
	labelSize := r.label.MinSize()
	width := labelSize.Width + padding*2
	height := labelSize.Height + padding

	if r.button.onClosed != nil {
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
	pad := th.Size(theme.SizeNamePadding)

	r.background.Resize(size)

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

	indicatorHeight := th.Size(theme.SizeNameSelectionRadius)
	r.indicator.Move(fyne.NewPos(0, size.Height-indicatorHeight))
	r.indicator.Resize(fyne.NewSize(size.Width, indicatorHeight))
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

	if !r.button.selected && r.button.hovered {
		r.background.FillColor = th.Color(theme.ColorNameHover, v)
	} else {
		r.background.FillColor = color.Transparent
	}

	r.background.Refresh()
	r.indicator.Refresh()

	r.label.Text = r.button.text
	r.label.Refresh()

	if r.closeBtn != nil {
		// Hover delivery is unreliable for our custom widget at the moment,
		// so keep the close button visible on every closable tab.
		r.closeBtn.Show()
		r.closeBtn.Refresh()
	}
}

// tabCloseButton is the small "x" shown on hover for closable tabs.
type tabCloseButton struct {
	widget.BaseWidget

	hovered  bool
	onTapped func()
}

func newTabCloseButton(onTapped func()) *tabCloseButton {
	b := &tabCloseButton{onTapped: onTapped}
	b.ExtendBaseWidget(b)
	return b
}

func (b *tabCloseButton) CreateRenderer() fyne.WidgetRenderer {
	th := b.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	background := canvas.NewRectangle(th.Color(theme.ColorNameHover, v))
	background.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
	background.Hide()

	icon := canvas.NewImageFromResource(theme.CancelIcon())

	return &tabCloseButtonRenderer{
		button:     b,
		background: background,
		icon:       icon,
		objects:    []fyne.CanvasObject{background, icon},
	}
}

func (b *tabCloseButton) Tapped(*fyne.PointEvent) {
	if b.onTapped != nil {
		b.onTapped()
	}
}

func (b *tabCloseButton) MouseIn(*desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
}

func (b *tabCloseButton) MouseOut() {
	b.hovered = false
	b.Refresh()
}

type tabCloseButtonRenderer struct {
	button     *tabCloseButton
	background *canvas.Rectangle
	icon       *canvas.Image
	objects    []fyne.CanvasObject
}

func (r *tabCloseButtonRenderer) Destroy() {}

func (r *tabCloseButtonRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *tabCloseButtonRenderer) MinSize() fyne.Size {
	return fyne.NewSquareSize(r.button.Theme().Size(theme.SizeNameInlineIcon))
}

func (r *tabCloseButtonRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
	r.icon.Resize(size)
}

func (r *tabCloseButtonRenderer) Refresh() {
	th := r.button.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	if r.button.hovered {
		r.background.FillColor = th.Color(theme.ColorNameHover, v)
		r.background.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
		r.background.Show()
	} else {
		r.background.Hide()
	}
	r.background.Refresh()
	r.icon.Refresh()
}
