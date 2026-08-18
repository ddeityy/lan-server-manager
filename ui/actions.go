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
	actions := &Actions{}

	actions.serverPasswordEntry = widget.NewPasswordEntry()

	actions.mapSelect = widget.NewSelect(appConfig.MapsOrDefault(), func(value string) {
		actions.setMapSelection(value)
	})
	actions.setMapSelection(appConfig.MapsOrDefault()[0])

	actions.configSelect = widget.NewSelect(appConfig.ConfigsOrDefault(), nil)
	actions.configSelect.SetSelected(appConfig.ConfigsOrDefault()[0])

	actions.customCommandEntry = widget.NewEntry()

	actions.changePasswordButton = widget.NewButton("Send", onChangePassword)
	actions.changeLevelButton = widget.NewButton("Send", onChangeLevel)
	actions.execConfigButton = widget.NewButton("Send", onExecConfig)
	actions.customCommandButton = widget.NewButton("Send", onSendCustom)

	actions.bindEnter(actions.serverPasswordEntry, actions.changePasswordButton, onChangePassword)
	actions.bindEnter(actions.customCommandEntry, actions.customCommandButton, onSendCustom)

	actions.statusLabel = widget.NewLabel("")

	actions.SetEnabled(false)
	return actions
}

// bindEnter fires handler when Enter is pressed in the entry, as long as
// the paired button is enabled (mirrors clicking it).
func (actions *Actions) bindEnter(entry *widget.Entry, button *widget.Button, handler func()) {
	entry.OnSubmitted = func(string) {
		if !button.Disabled() {
			handler()
		}
	}
}

// setMapSelection sets the map dropdown's current value and renders the
// remaining pool options so the currently selected map is not shown twice.
func (actions *Actions) setMapSelection(value string) {
	maps := appConfig.MapsOrDefault()
	valid := slices.Contains(maps, value)
	if !valid {
		actions.mapSelect.Selected = ""
		actions.mapSelect.Options = append([]string(nil), maps...)
		actions.mapSelect.Refresh()
		return
	}

	opts := make([]string, 0, len(maps)-1)
	for _, m := range maps {
		if m != value {
			opts = append(opts, m)
		}
	}
	actions.mapSelect.Selected = value
	actions.mapSelect.Options = opts
	actions.mapSelect.Refresh()
}

// View returns the actions form and status as actions single canvas object.
func (actions *Actions) View() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabelWithStyle("Change server password", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(actions.serverPasswordEntry, actions.changePasswordButton),
		widget.NewLabelWithStyle("Change map", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(actions.mapSelect, actions.changeLevelButton),
		widget.NewLabelWithStyle("Exec config", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(actions.configSelect, actions.execConfigButton),
		widget.NewLabelWithStyle("Custom command", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(actions.customCommandEntry, actions.customCommandButton),
		actions.statusLabel,
	)
}

// SetEnabled enables or disables all action send buttons. Inputs remain editable.
func (actions *Actions) SetEnabled(enabled bool) {
	setButtonsEnabled(enabled,
		actions.changePasswordButton,
		actions.changeLevelButton,
		actions.execConfigButton,
		actions.customCommandButton,
	)
}

// ClearServerPassword clears the server password entry.
func (actions *Actions) ClearServerPassword() { actions.serverPasswordEntry.SetText("") }

// ClearCustomCommand clears the custom command entry.
func (actions *Actions) ClearCustomCommand() { actions.customCommandEntry.SetText("") }

// SetStatus updates the status label at the bottom of the actions card.
func (actions *Actions) SetStatus(text string) { actions.statusLabel.SetText(text) }

// SelectedMap returns the currently selected map.
func (actions *Actions) SelectedMap() string { return actions.mapSelect.Selected }

// SelectedConfig returns the currently selected exec config.
func (actions *Actions) SelectedConfig() string { return actions.configSelect.Selected }

// ServerPassword returns the server password field value.
func (actions *Actions) ServerPassword() string { return actions.serverPasswordEntry.Text }

// SetServerPassword sets the server password field value.
func (actions *Actions) SetServerPassword(password string) {
	actions.serverPasswordEntry.SetText(password)
}
