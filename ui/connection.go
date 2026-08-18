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

const (
	defaultAddress  = "0.0.0.0:27015"
	defaultPassword = "test"
)

func newConnection(onConnect, onDisconnect func()) *Connection {
	connection := &Connection{}

	connection.addressEntry = widget.NewEntry()
	connection.addressEntry.SetText(defaultAddress)

	connection.passwordEntry = widget.NewPasswordEntry()
	connection.passwordEntry.SetText(defaultPassword)

	connection.sshHostEntry = widget.NewEntry()
	connection.sshHostEntry.SetPlaceHolder("ssh host (optional)")

	connection.sshPasswordEntry = widget.NewPasswordEntry()
	connection.sshPasswordEntry.SetPlaceHolder("ssh password (optional)")

	connection.containerNameEntry = widget.NewEntry()
	connection.containerNameEntry.SetPlaceHolder("container name or pattern")

	connection.connectButton = widget.NewButton("Connect", onConnect)
	connection.disconnectButton = widget.NewButton("Disconnect", onDisconnect)
	connection.disconnectButton.Disable()

	connection.statusLabel = widget.NewLabel("")

	return connection
}

// View returns the connection form, buttons, and status as a single canvas object.
func (connection *Connection) View() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Address       ", connection.addressEntry),
			widget.NewFormItem("RCON Password", connection.passwordEntry),
			widget.NewFormItem("Container     ", connection.containerNameEntry),
			widget.NewFormItem("SSH Host      ", connection.sshHostEntry),
			widget.NewFormItem("SSH Password  ", connection.sshPasswordEntry),
		),
		container.NewGridWithColumns(2, connection.connectButton, connection.disconnectButton),
		connection.statusLabel,
	)
}

// Address returns the configured RCON address.
func (connection *Connection) Address() string { return connection.addressEntry.Text }

// Password returns the configured RCON password.
func (connection *Connection) Password() string { return connection.passwordEntry.Text }

// SetAddress sets the RCON address field.
func (connection *Connection) SetAddress(addr string) { connection.addressEntry.SetText(addr) }

// SetPassword sets the RCON password field.
func (connection *Connection) SetPassword(pw string) { connection.passwordEntry.SetText(pw) }

// SSHHost returns the configured SSH host for log tailing.
func (connection *Connection) SSHHost() string { return connection.sshHostEntry.Text }

// SSHPassword returns the configured SSH password for log tailing.
func (connection *Connection) SSHPassword() string { return connection.sshPasswordEntry.Text }

// ContainerName returns the configured container name or pattern for log tailing.
func (connection *Connection) ContainerName() string { return connection.containerNameEntry.Text }

// SetSSHHost sets the SSH host field.
func (connection *Connection) SetSSHHost(host string) { connection.sshHostEntry.SetText(host) }

// SetSSHPassword sets the SSH password field.
func (connection *Connection) SetSSHPassword(pw string) { connection.sshPasswordEntry.SetText(pw) }

// SetContainerName sets the container name or pattern field.
func (connection *Connection) SetContainerName(name string) {
	connection.containerNameEntry.SetText(name)
}

// SetStatus updates the status label at the bottom of the connection form.
func (connection *Connection) SetStatus(text string) { connection.statusLabel.SetText(text) }

// SetConnecting shows a connecting state and disables both buttons.
func (connection *Connection) SetConnecting() {
	connection.connectButton.Disable()
	connection.disconnectButton.Disable()
	connection.SetStatus("Connecting...")
}

// SetConnected enables or disables the connect/disconnect buttons and updates
// the status text. When connected the status reads "Connected".
func (connection *Connection) SetConnected(connected bool) {
	setButtonsEnabled(!connected, connection.connectButton)
	setButtonsEnabled(connected, connection.disconnectButton)
	if connected {
		connection.SetStatus("Connected")
	}
}
