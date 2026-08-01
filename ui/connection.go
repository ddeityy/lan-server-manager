package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Connection holds the widgets for connecting to a server over RCON.
type Connection struct {
	addressEntry     *widget.Entry
	passwordEntry    *widget.Entry
	statusLabel      *widget.Label
	connectButton    *widget.Button
	disconnectButton *widget.Button
}

func newConnection(onConnect, onDisconnect, onChanged func()) *Connection {
	c := &Connection{}

	c.addressEntry = widget.NewEntry()
	c.addressEntry.SetText("0.0.0.0:27015")
	c.addressEntry.OnChanged = func(string) { onChanged() }

	c.passwordEntry = widget.NewPasswordEntry()
	c.passwordEntry.SetText("test")
	c.passwordEntry.OnChanged = func(string) { onChanged() }

	c.statusLabel = widget.NewLabel("")

	c.connectButton = widget.NewButton("Connect", onConnect)
	c.disconnectButton = widget.NewButton("Disconnect", onDisconnect)
	c.disconnectButton.Disable()

	return c
}

// View returns the connection form and buttons as a single canvas object.
func (c *Connection) View() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Address   ", c.addressEntry),
			widget.NewFormItem("Password", c.passwordEntry),
		),
		container.NewGridWithColumns(2, c.connectButton, c.disconnectButton),
		c.statusLabel,
	)
}

// Address returns the configured RCON address.
func (c *Connection) Address() string { return c.addressEntry.Text }

// Password returns the configured RCON password.
func (c *Connection) Password() string { return c.passwordEntry.Text }

// SetAddress sets the RCON address field.
func (c *Connection) SetAddress(addr string) { c.addressEntry.SetText(addr) }

// SetPassword sets the RCON password field.
func (c *Connection) SetPassword(pw string) { c.passwordEntry.SetText(pw) }

// SetStatus updates the status label under the connection box.
func (c *Connection) SetStatus(text string) { c.statusLabel.SetText(text) }

// SetConnecting shows a connecting state and disables both buttons.
func (c *Connection) SetConnecting() {
	c.connectButton.Disable()
	c.disconnectButton.Disable()
	c.SetStatus("Connecting...")
}

// SetConnected enables or disables the connect/disconnect buttons and updates
// the status text. When connected the status reads "Connected".
func (c *Connection) SetConnected(connected bool) {
	if connected {
		c.connectButton.Disable()
		c.disconnectButton.Enable()
		c.SetStatus("Connected")
		return
	}
	c.connectButton.Enable()
	c.disconnectButton.Disable()
}
