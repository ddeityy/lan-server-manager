package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Connection holds the widgets for connecting to a server over RCON and for
// configuring the docker log tail target.
type Connection struct {
	addressEntry       *widget.Entry
	passwordEntry      *widget.Entry
	sshHostEntry       *widget.Entry
	sshPasswordEntry   *widget.Entry
	containerNameEntry *widget.Entry
	statusLabel        *widget.Label
	connectButton      *widget.Button
	disconnectButton   *widget.Button
}

func newConnection(onConnect, onDisconnect func()) *Connection {
	c := &Connection{}

	c.addressEntry = widget.NewEntry()
	c.addressEntry.SetText("0.0.0.0:27015")

	c.passwordEntry = widget.NewPasswordEntry()
	c.passwordEntry.SetText("test")

	c.sshHostEntry = widget.NewEntry()
	c.sshHostEntry.SetPlaceHolder("ssh host (optional)")

	c.sshPasswordEntry = widget.NewPasswordEntry()
	c.sshPasswordEntry.SetPlaceHolder("ssh password (optional)")

	c.containerNameEntry = widget.NewEntry()
	c.containerNameEntry.SetPlaceHolder("container name or pattern")

	c.connectButton = widget.NewButton("Connect", onConnect)
	c.disconnectButton = widget.NewButton("Disconnect", onDisconnect)
	c.disconnectButton.Disable()

	c.statusLabel = widget.NewLabel("")

	return c
}

// View returns the connection form, buttons, and status as a single canvas object.
func (c *Connection) View() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Address       ", c.addressEntry),
			widget.NewFormItem("RCON Password", c.passwordEntry),
			widget.NewFormItem("Container     ", c.containerNameEntry),
			widget.NewFormItem("SSH Host      ", c.sshHostEntry),
			widget.NewFormItem("SSH Password  ", c.sshPasswordEntry),
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

// SSHHost returns the configured SSH host for log tailing.
func (c *Connection) SSHHost() string { return c.sshHostEntry.Text }

// SSHPassword returns the configured SSH password for log tailing.
func (c *Connection) SSHPassword() string { return c.sshPasswordEntry.Text }

// ContainerName returns the configured container name or pattern for log tailing.
func (c *Connection) ContainerName() string { return c.containerNameEntry.Text }

// SetSSHHost sets the SSH host field.
func (c *Connection) SetSSHHost(host string) { c.sshHostEntry.SetText(host) }

// SetSSHPassword sets the SSH password field.
func (c *Connection) SetSSHPassword(pw string) { c.sshPasswordEntry.SetText(pw) }

// SetContainerName sets the container name or pattern field.
func (c *Connection) SetContainerName(name string) { c.containerNameEntry.SetText(name) }

// SetStatus updates the status label at the bottom of the connection form.
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
