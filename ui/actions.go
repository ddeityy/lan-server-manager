package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Actions holds the widgets for RCON actions: password, map, config, message, and custom commands.
type Actions struct {
	serverPasswordEntry  *widget.Entry
	changePasswordButton *widget.Button
	mapSelect            *widget.Select
	changeLevelButton    *widget.Button
	configSelect         *widget.Select
	execConfigButton     *widget.Button
	messageEntry         *widget.Entry
	sendMessageButton    *widget.Button
	customCommandEntry   *widget.Entry
	customCommandButton  *widget.Button
	statusLabel          *widget.Label
}

func newActions(
	onChangePassword,
	onChangeLevel,
	onExecConfig,
	onSendMessage,
	onSendCustom,
	onMapSelected,
	onConfigSelected,
	onChanged func(),
) *Actions {
	a := &Actions{}

	a.serverPasswordEntry = widget.NewPasswordEntry()
	a.serverPasswordEntry.OnChanged = func(string) { onChanged() }

	a.mapSelect = widget.NewSelect(mapList(), func(value string) {
		setMapSelection(a.mapSelect, value)
		onMapSelected()
	})
	setMapSelection(a.mapSelect, mapList()[0])

	a.configSelect = widget.NewSelect(configList(), func(string) { onConfigSelected() })
	a.configSelect.SetSelected(configList()[0])

	a.messageEntry = widget.NewEntry()
	a.messageEntry.OnChanged = func(string) { onChanged() }

	a.customCommandEntry = widget.NewEntry()
	a.customCommandEntry.OnChanged = func(string) { onChanged() }

	a.changePasswordButton = widget.NewButton("Send", onChangePassword)
	a.changeLevelButton = widget.NewButton("Send", onChangeLevel)
	a.execConfigButton = widget.NewButton("Send", onExecConfig)
	a.sendMessageButton = widget.NewButton("Send", onSendMessage)
	a.customCommandButton = widget.NewButton("Send", onSendCustom)

	a.statusLabel = widget.NewLabel("")

	a.SetEnabled(false)
	return a
}

// View returns the actions form as a single canvas object.
func (a *Actions) View() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabelWithStyle("Change server password", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(a.serverPasswordEntry, a.changePasswordButton),
		widget.NewLabelWithStyle("Change map", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(a.mapSelect, a.changeLevelButton),
		widget.NewLabelWithStyle("Exec config", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(a.configSelect, a.execConfigButton),
		widget.NewLabelWithStyle("Send message", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(a.messageEntry, a.sendMessageButton),
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
		a.sendMessageButton.Enable()
		a.customCommandButton.Enable()
		return
	}
	a.changePasswordButton.Disable()
	a.changeLevelButton.Disable()
	a.execConfigButton.Disable()
	a.sendMessageButton.Disable()
	a.customCommandButton.Disable()
}

// SetStatus updates the status label under the actions box.
func (a *Actions) SetStatus(text string) { a.statusLabel.SetText(text) }

// SelectedMap returns the currently selected map.
func (a *Actions) SelectedMap() string { return a.mapSelect.Selected }

// SetMap sets the map dropdown selection while avoiding duplicate entries.
func (a *Actions) SetMap(value string) { setMapSelection(a.mapSelect, value) }

// SelectedConfig returns the currently selected exec config.
func (a *Actions) SelectedConfig() string { return a.configSelect.Selected }

// ServerPassword returns the server password field value.
func (a *Actions) ServerPassword() string { return a.serverPasswordEntry.Text }

// SetServerPassword sets the server password field value.
func (a *Actions) SetServerPassword(password string) { a.serverPasswordEntry.SetText(password) }

// Message returns the send-message field value.
func (a *Actions) Message() string { return a.messageEntry.Text }

// CustomCommand returns the custom command field value, trimmed.
func (a *Actions) CustomCommand() string {
	// TrimSpace is performed by callers; expose raw if needed for display.
	return a.customCommandEntry.Text
}
