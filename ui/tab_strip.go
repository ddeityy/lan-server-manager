package ui

import (
	"reflect"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// TabStrip wraps container.DocTabs so we can hide the overflow "more" button.
type TabStrip struct {
	*container.DocTabs
}

func newTabStrip(docTabs *container.DocTabs) *TabStrip {
	return &TabStrip{DocTabs: docTabs}
}

// CreateRenderer returns the standard DocTabs renderer wrapped so we can hide
// the overflow action button after every layout/refresh.
func (t *TabStrip) CreateRenderer() fyne.WidgetRenderer {
	inner := t.DocTabs.CreateRenderer()
	hideOverflowAction(inner)
	return &tabStripRenderer{inner: inner}
}

type tabStripRenderer struct {
	inner fyne.WidgetRenderer
}

func (r *tabStripRenderer) Destroy() {
	r.inner.Destroy()
}

func (r *tabStripRenderer) Layout(size fyne.Size) {
	r.inner.Layout(size)
	hideOverflowAction(r.inner)
}

func (r *tabStripRenderer) MinSize() fyne.Size {
	return r.inner.MinSize()
}

func (r *tabStripRenderer) Objects() []fyne.CanvasObject {
	return r.inner.Objects()
}

func (r *tabStripRenderer) Refresh() {
	r.inner.Refresh()
	hideOverflowAction(r.inner)
}

// hideOverflowAction locates the unexported "action" button on the inner
// DocTabs renderer and hides it.
func hideOverflowAction(renderer fyne.WidgetRenderer) {
	rv := reflect.ValueOf(renderer)
	if rv.Kind() == reflect.Interface {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Ptr {
		return
	}
	rv = rv.Elem()

	actionField := rv.FieldByName("action")
	if !actionField.IsValid() || actionField.IsNil() {
		return
	}

	// action is a *widget.Button; cast directly and hide it.
	btn := (*widget.Button)(unsafe.Pointer(actionField.Pointer()))
	btn.Hide()
}
