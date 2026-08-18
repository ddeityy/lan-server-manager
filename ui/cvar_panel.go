package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// trackedCVar describes one CVar shown in the scoreboard header panel.
type trackedCVar struct {
	name  string
	label string
}

var trackedCVars = []trackedCVar{
	{name: "mp_timelimit", label: "mp_timelimit"},
	{name: "mp_winlimit", label: "mp_winlimit"},
	{name: "mp_windifference", label: "mp_windifference"},
	{name: "tv_delay", label: "tv_delay"},
}

// TrackedCVarNames returns the names of all CVars shown in the panel.
func TrackedCVarNames() []string {
	names := make([]string, len(trackedCVars))
	for i, cv := range trackedCVars {
		names[i] = cv.name
	}
	return names
}

// CVarPanel displays the current values of tracked CVars and highlights values
// that were recently changed from the UI until a confirming log echo arrives.
type CVarPanel struct {
	container *fyne.Container
	values    map[string]*canvas.Text
	pending   map[string]bool
}

func newCVarPanel() *CVarPanel {
	cp := &CVarPanel{
		values:  make(map[string]*canvas.Text),
		pending: make(map[string]bool),
	}

	objects := make([]fyne.CanvasObject, 0, len(trackedCVars))
	for _, cv := range trackedCVars {
		valueText := canvas.NewText("-", color.White)
		valueText.TextStyle = fyne.TextStyle{Bold: true}
		valueText.Alignment = fyne.TextAlignCenter
		cp.values[cv.name] = valueText
		objects = append(objects, container.NewHBox(
			widget.NewLabelWithStyle(cv.label+":", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			valueText,
		))
	}

	cp.container = container.NewGridWithColumns(len(trackedCVars), objects...)
	return cp
}

// View returns the CVar panel as a single canvas object.
func (cp *CVarPanel) View() fyne.CanvasObject {
	return cp.container
}

// Set updates the displayed value for a tracked CVar and clears its pending state.
func (cp *CVarPanel) Set(name, value string) {
	lbl, ok := cp.values[name]
	if !ok {
		return
	}
	delete(cp.pending, name)
	lbl.Text = value
	lbl.Color = color.White
	lbl.Refresh()
}

// MarkPending highlights the listed tracked CVars to indicate a UI-initiated
// change is in flight. The highlight is cleared when Set receives the new value.
func (cp *CVarPanel) MarkPending(names ...string) {
	pendingColor := color.NRGBA{R: 255, G: 200, B: 80, A: 255}
	for _, name := range names {
		lbl, ok := cp.values[name]
		if !ok {
			continue
		}
		cp.pending[name] = true
		lbl.Color = pendingColor
		lbl.Refresh()
	}
}

// ClearPending removes the pending highlight from the listed CVars without
// changing their values.
func (cp *CVarPanel) ClearPending(names ...string) {
	for _, name := range names {
		if _, ok := cp.pending[name]; !ok {
			continue
		}
		delete(cp.pending, name)
		if lbl, ok := cp.values[name]; ok {
			lbl.Color = color.White
			lbl.Refresh()
		}
	}
}

// Reset clears all displayed values and pending highlights.
func (cp *CVarPanel) Reset() {
	for name, lbl := range cp.values {
		delete(cp.pending, name)
		lbl.Text = "-"
		lbl.Color = color.White
		lbl.Refresh()
	}
}

// trackedCVarNamesFromCommand inspects a raw RCON command and returns any
// tracked CVars that appear to be set by it. Only simple "cvar value" forms are
// recognized; config execs are handled separately by marking every tracked CVar.
func trackedCVarNamesFromCommand(cmd string) []string {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}
	token := cmd
	if i := strings.IndexAny(cmd, " \t"); i >= 0 {
		token = cmd[:i]
	}
	for _, cv := range trackedCVars {
		if strings.EqualFold(token, cv.name) {
			return []string{cv.name}
		}
	}
	return nil
}
