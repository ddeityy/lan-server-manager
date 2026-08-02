package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/rcon"
)

// ServerInfo holds the widgets that display parsed status output and the
// controls that refresh it.
type ServerInfo struct {
	serverNameLabel      *widget.Label
	addressLabel         *widget.Label
	sourceTVLabel        *widget.Label
	mapLabel             *widget.Label
	connectLabel         *widget.Label
	stvLabel             *widget.Label
	copyConnectButton    *widget.Button
	copySTVButton        *widget.Button
	refreshIntervalEntry *widget.Entry
	statusLabel          *widget.Label

	lastInfo rcon.ServerInfo
}

func newServerInfo(
	title string,
	onCopyConnect,
	onCopySTV,
	onIntervalChanged func(),
) *ServerInfo {
	si := &ServerInfo{}

	si.serverNameLabel = widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	si.addressLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	si.sourceTVLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	si.mapLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	si.connectLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	si.stvLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	si.copyConnectButton = widget.NewButtonWithIcon("", theme.ContentCopyIcon(), onCopyConnect)
	si.copyConnectButton.Disable()
	si.copySTVButton = widget.NewButtonWithIcon("", theme.ContentCopyIcon(), onCopySTV)
	si.copySTVButton.Disable()

	si.refreshIntervalEntry = widget.NewEntry()
	si.refreshIntervalEntry.SetText(fmt.Sprintf("%d", defaultRefreshInterval))
	si.refreshIntervalEntry.SetPlaceHolder("seconds")
	si.refreshIntervalEntry.OnSubmitted = func(string) { onIntervalChanged() }
	si.refreshIntervalEntry.OnChanged = func(string) { onIntervalChanged() }

	si.statusLabel = widget.NewLabel("")

	return si
}

// View returns the server info card as a single canvas object.
func (si *ServerInfo) View(connection fyne.CanvasObject) fyne.CanvasObject {
	connectionTile := widget.NewCard("Connection", "", connection)
	nameTile := widget.NewCard("Name", "", si.serverNameLabel)
	mapTile := widget.NewCard("Map", "", si.mapLabel)
	addressTile := widget.NewCard("Address", "", si.addressLabel)
	sourceTVTile := widget.NewCard("SourceTV", "", si.sourceTVLabel)

	header := container.NewHBox(
		widget.NewLabelWithStyle("Server Info", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Refresh interval (s):"),
		si.refreshIntervalEntry,
	)

	return widget.NewCard("", "", container.NewVBox(
		header,
		connectionTile,
		nameTile,
		mapTile,
		addressTile,
		sourceTVTile,
		si.statusLabel,
	))
}

// SetRefreshing sets the status text shown while a refresh is in flight.
func (si *ServerInfo) SetRefreshing() { si.SetStatus("Refreshing...") }

// SetStatus updates the status label at the bottom of the server info card.
func (si *ServerInfo) SetStatus(text string) { si.statusLabel.SetText(text) }

// SetName updates the name displayed at the top of the server info card.
func (si *ServerInfo) SetName(name string) { si.serverNameLabel.SetText(name) }

// Reset clears all displayed server info and disables copy buttons.
func (si *ServerInfo) Reset() {
	si.lastInfo = rcon.ServerInfo{}
	si.serverNameLabel.SetText("")
	si.addressLabel.SetText("")
	si.mapLabel.SetText("")
	si.sourceTVLabel.SetText("")
	si.connectLabel.SetText("")
	si.stvLabel.SetText("")
	si.SetStatus("")
	si.copyConnectButton.Disable()
	si.copySTVButton.Disable()
}

// RefreshInterval returns the refresh interval in seconds. Invalid values fall
// back to the default.
func (si *ServerInfo) RefreshInterval() int {
	s := si.refreshIntervalEntry.Text
	if s == "" {
		return defaultRefreshInterval
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return defaultRefreshInterval
	}
	return n
}

// SetInfo updates the displayed server info and the copyable connect strings.
func (si *ServerInfo) SetInfo(info rcon.ServerInfo, configuredAddress, password string) {
	si.lastInfo = info

	si.addressLabel.SetText(formatAddress(info.Address, configuredAddress))

	if info.SourceTV.Address != "" {
		tvText := fmt.Sprintf("%s, delay %s", info.SourceTV.Address, info.SourceTV.Delay)
		if rcon.AddressIsValid(info.SourceTV.Local) {
			tvText += "\nlocal " + info.SourceTV.Local
		}
		si.sourceTVLabel.SetText(tvText)
	} else {
		si.sourceTVLabel.SetText("")
	}

	si.mapLabel.SetText(info.Map)

	si.updateConnectStrings(password)
}

// LastInfo returns the most recently displayed server information.
func (si *ServerInfo) LastInfo() rcon.ServerInfo { return si.lastInfo }

// UpdateConnectStrings rebuilds connect and STV strings using the supplied
// password.
func (si *ServerInfo) UpdateConnectStrings(password string) {
	si.updateConnectStrings(password)
}

func (si *ServerInfo) updateConnectStrings(password string) {
	info := si.lastInfo

	gameAddr := info.GameConnectAddress()
	if gameAddr == "" {
		si.connectLabel.SetText("")
		si.copyConnectButton.Disable()
	} else {
		si.connectLabel.SetText(connectCommand(gameAddr, password))
		si.copyConnectButton.Enable()
	}

	stvAddr := info.STVConnectAddress()
	if stvAddr == "" {
		si.stvLabel.SetText("")
		si.copySTVButton.Disable()
		return
	}

	si.stvLabel.SetText(connectCommand(stvAddr, password))
	si.copySTVButton.Enable()
}

// formatAddress returns a multi-line summary of the server's addresses,
// omitting empty or unknown placeholders.
func formatAddress(a rcon.Address, configured string) string {
	parts := []string{}
	if rcon.AddressIsValid(a.IP) {
		parts = append(parts, fmt.Sprintf("IP: %s", a.IP))
	}
	if rcon.AddressIsValid(configured) && configured != a.IP {
		parts = append(parts, fmt.Sprintf("Local: %s", configured))
	}
	if len(parts) == 0 {
		return "Local: 127.0.0.1:27015"
	}
	return strings.Join(parts, "\n")
}

// connectCommand builds a TF2 connect command, optionally including a password.
func connectCommand(address, password string) string {
	cmd := fmt.Sprintf("connect %s", address)
	if password != "" {
		cmd += fmt.Sprintf("; password %s", password)
	}
	return cmd
}
