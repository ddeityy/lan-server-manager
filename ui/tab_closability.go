package ui

import (
	"reflect"
	"unsafe"

	"fyne.io/fyne/v2/container"
)

// updateTabClosability hides the close button on the last remaining server tab
// and on the "+" tab. Fyne assigns every DocTabs item a close handler, so we nil
// it out via reflection when it should not be shown.
func (m *Manager) updateTabClosability() {
	closable := len(m.panels) > 1
	for _, panel := range m.panels {
		item := panel.TabItem()
		if closable {
			setTabCloseHandler(item, func() { m.onTabCloseIntercept(item) })
		} else {
			setTabCloseHandler(item, nil)
		}
	}
	if m.plusTab != nil {
		setTabCloseHandler(m.plusTab, nil)
	}
	m.tabs.Refresh()
}

// setTabCloseHandler modifies the unexported onClosed callback of a tab button.
// handler == nil hides the close button entirely.
func setTabCloseHandler(item *container.TabItem, handler func()) {
	rv := reflect.ValueOf(item).Elem()
	buttonPtr := rv.FieldByName("button")
	if !buttonPtr.IsValid() || buttonPtr.IsNil() {
		return
	}

	button := reflect.NewAt(buttonPtr.Type().Elem(), unsafe.Pointer(buttonPtr.Pointer())).Elem()
	onClosed := button.FieldByName("onClosed")
	if !onClosed.IsValid() {
		return
	}

	ptr := (*func())(unsafe.Pointer(onClosed.UnsafeAddr()))
	*ptr = handler
}
