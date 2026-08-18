package ui

import (
	"image/color"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// trackedCVar describes one CVar shown in the CVar panel.
type trackedCVar struct {
	name   string
	label  string
	active bool
}

func trackedCVarList() []trackedCVar {
	return []trackedCVar{
		{name: "mp_timelimit", label: "mp_timelimit", active: true},
		{name: "mp_winlimit", label: "mp_winlimit", active: true},
		{name: "mp_windifference", label: "mp_windifference", active: true},
		{name: "tv_delay", label: "tv_delay", active: false},
	}
}

// TrackedCVarNames returns the names of all CVars shown in the panel.
func TrackedCVarNames() []string {
	cvars := trackedCVarList()
	names := make([]string, len(cvars))
	for i, cv := range cvars {
		names[i] = cv.name
	}
	return names
}

// ActiveCVarNames returns the names of CVars that are actively queried after
// config execs. Passive CVars (e.g. tv_delay) are left to update from server_cvar
// log lines instead of being stuck in a pending state.
func ActiveCVarNames() []string {
	var names []string
	for _, cv := range trackedCVarList() {
		if cv.active {
			names = append(names, cv.name)
		}
	}
	return names
}

// IsTrackedCVar reports whether name is one of the CVars displayed in the panel.
func IsTrackedCVar(name string) bool {
	return slices.ContainsFunc(trackedCVarList(), func(cv trackedCVar) bool {
		return cv.name == name
	})
}

// CVarPanel displays the current values of tracked CVars and highlights values
// that were recently changed from the UI until a confirming log echo arrives.
type CVarPanel struct {
	container *fyne.Container
	values    map[string]*canvas.Text
	pending   map[string]bool
}

func newCVarPanel() *CVarPanel {
	panel := &CVarPanel{
		values:  make(map[string]*canvas.Text),
		pending: make(map[string]bool),
	}

	panel.container = container.NewVBox()
	for _, cvar := range trackedCVarList() {
		valueText := canvas.NewText("-", color.White)
		valueText.TextStyle = fyne.TextStyle{Bold: true}
		valueText.Alignment = fyne.TextAlignLeading
		panel.values[cvar.name] = valueText
		row := container.NewHBox(
			widget.NewLabelWithStyle(cvar.label+":", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			valueText,
		)
		panel.container.Add(row)
	}
	return panel
}

// View returns the CVar panel as a single canvas object.
func (panel *CVarPanel) View() fyne.CanvasObject {
	return panel.container
}

// Value returns the currently displayed value for a tracked CVar.
func (panel *CVarPanel) Value(name string) string {
	lbl, ok := panel.values[name]
	if !ok {
		return ""
	}
	return lbl.Text
}

// Set updates the displayed value for a tracked CVar and clears its pending state.
func (panel *CVarPanel) Set(name, value string) {
	lbl, ok := panel.values[name]
	if !ok {
		return
	}
	delete(panel.pending, name)
	lbl.Text = value
	lbl.Color = color.White
	lbl.Refresh()
}

// MarkPending highlights the listed tracked CVars to indicate a UI-initiated
// change is in flight. The highlight is cleared when Set receives the new value.
func (panel *CVarPanel) MarkPending(names ...string) {
	pendingColor := pendingHighlightColor()
	for _, name := range names {
		lbl, ok := panel.values[name]
		if !ok {
			continue
		}
		panel.pending[name] = true
		lbl.Color = pendingColor
		lbl.Refresh()
	}
}

// ClearPending removes the pending highlight from the listed CVars without
// changing their values.
func (panel *CVarPanel) ClearPending(names ...string) {
	for _, name := range names {
		if _, ok := panel.pending[name]; !ok {
			continue
		}
		delete(panel.pending, name)
		if lbl, ok := panel.values[name]; ok {
			lbl.Color = color.White
			lbl.Refresh()
		}
	}
}

// Reset clears all displayed values and pending highlights.
func (panel *CVarPanel) Reset() {
	for name, lbl := range panel.values {
		delete(panel.pending, name)
		lbl.Text = "-"
		lbl.Color = color.White
		lbl.Refresh()
	}
}

const (
	pendingRed   = 255
	pendingGreen = 200
	pendingBlue  = 80
	opaqueAlpha  = 255
)

func pendingHighlightColor() color.NRGBA {
	return color.NRGBA{R: pendingRed, G: pendingGreen, B: pendingBlue, A: opaqueAlpha}
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
	for _, cv := range trackedCVarList() {
		if strings.EqualFold(token, cv.name) {
			return []string{cv.name}
		}
	}
	return nil
}
