package ui

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Actions holds the widgets for RCON actions: password, map, config, and custom commands.
type Actions struct {
	serverPasswordEntry  *widget.Entry
	changePasswordButton *widget.Button
	mapSelect            *widget.Select
	changeLevelButton    *widget.Button
	configSelect         *widget.Select
	execConfigButton     *widget.Button
	customCommandEntry   *widget.Entry
	customCommandButton  *widget.Button
	statusLabel          *widget.Label
}

func newActions(
	onChangePassword,
	onChangeLevel,
	onExecConfig,
	onSendCustom func(),
) *Actions {
	a := &Actions{}

	a.serverPasswordEntry = widget.NewPasswordEntry()

	a.mapSelect = widget.NewSelect(appConfig.MapsOrDefault(), func(value string) {
		a.setMapSelection(value)
	})
	a.setMapSelection(appConfig.MapsOrDefault()[0])

	a.configSelect = widget.NewSelect(appConfig.ConfigsOrDefault(), nil)
	a.configSelect.SetSelected(appConfig.ConfigsOrDefault()[0])

	a.customCommandEntry = widget.NewEntry()

	a.changePasswordButton = widget.NewButton("Send", onChangePassword)
	a.changeLevelButton = widget.NewButton("Send", onChangeLevel)
	a.execConfigButton = widget.NewButton("Send", onExecConfig)
	a.customCommandButton = widget.NewButton("Send", onSendCustom)

	a.bindEnter(a.serverPasswordEntry, a.changePasswordButton, onChangePassword)
	a.bindEnter(a.customCommandEntry, a.customCommandButton, onSendCustom)

	a.statusLabel = widget.NewLabel("")

	a.SetEnabled(false)
	return a
}

// bindEnter fires handler when Enter is pressed in the entry, as long as
// the paired button is enabled (mirrors clicking it).
func (a *Actions) bindEnter(entry *widget.Entry, button *widget.Button, handler func()) {
	entry.OnSubmitted = func(string) {
		if !button.Disabled() {
			handler()
		}
	}
}

// setMapSelection sets the map dropdown's current value and renders the
// remaining pool options so the currently selected map is not shown twice.
func (a *Actions) setMapSelection(value string) {
	maps := appConfig.MapsOrDefault()
	valid := slices.Contains(maps, value)
	if !valid {
		a.mapSelect.Selected = ""
		a.mapSelect.Options = append([]string(nil), maps...)
		a.mapSelect.Refresh()
		return
	}

	opts := make([]string, 0, len(maps)-1)
	for _, m := range maps {
		if m != value {
			opts = append(opts, m)
		}
	}
	a.mapSelect.Selected = value
	a.mapSelect.Options = opts
	a.mapSelect.Refresh()
}

// View returns the actions form and status as a single canvas object.
func (a *Actions) View() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabelWithStyle("Change server password", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(a.serverPasswordEntry, a.changePasswordButton),
		widget.NewLabelWithStyle("Change map", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(a.mapSelect, a.changeLevelButton),
		widget.NewLabelWithStyle("Exec config", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(a.configSelect, a.execConfigButton),
		widget.NewLabelWithStyle("Custom command", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(a.customCommandEntry, a.customCommandButton),
		a.statusLabel,
	)
}

// SetEnabled enables or disables all action send buttons. Inputs remain editable.
func (a *Actions) SetEnabled(enabled bool) {
	if enabled {
		a.changePasswordButton.Enable()
		a.changeLevelButton.Enable()
		a.execConfigButton.Enable()
		a.customCommandButton.Enable()
		return
	}
	a.changePasswordButton.Disable()
	a.changeLevelButton.Disable()
	a.execConfigButton.Disable()
	a.customCommandButton.Disable()
}

// ClearServerPassword clears the server password entry.
func (a *Actions) ClearServerPassword() { a.serverPasswordEntry.SetText("") }

// ClearCustomCommand clears the custom command entry.
func (a *Actions) ClearCustomCommand() { a.customCommandEntry.SetText("") }

// SetStatus updates the status label at the bottom of the actions card.
func (a *Actions) SetStatus(text string) { a.statusLabel.SetText(text) }

// SelectedMap returns the currently selected map.
func (a *Actions) SelectedMap() string { return a.mapSelect.Selected }

// SelectedConfig returns the currently selected exec config.
func (a *Actions) SelectedConfig() string { return a.configSelect.Selected }

// ServerPassword returns the server password field value.
func (a *Actions) ServerPassword() string { return a.serverPasswordEntry.Text }

// SetServerPassword sets the server password field value.
func (a *Actions) SetServerPassword(password string) { a.serverPasswordEntry.SetText(password) }
