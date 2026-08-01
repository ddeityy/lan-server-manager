package ui

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/server"
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

	lastInfo server.ServerInfo
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
func (si *ServerInfo) View() fyne.CanvasObject {
	addressTile := widget.NewCard("Address", "", si.addressLabel)
	mapTile := widget.NewCard("Map", "", si.mapLabel)

	sourceTVTile := widget.NewCard("SourceTV", "", si.sourceTVLabel)
	connectTile := widget.NewCard("Connect", "", container.NewBorder(nil, nil, nil, si.copyConnectButton, si.connectLabel))

	nameTile := widget.NewCard("Name", "", si.serverNameLabel)

	header := container.NewHBox(
		widget.NewLabelWithStyle("Server Info", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Refresh interval (s):"),
		si.refreshIntervalEntry,
	)

	stvConnectTile := widget.NewCard("STV connect", "", container.NewBorder(nil, nil, nil, si.copySTVButton, si.stvLabel))

	return widget.NewCard("", "", container.NewVBox(
		header,
		nameTile,
		mapTile,
		addressTile,
		connectTile,
		sourceTVTile,
		stvConnectTile,
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
	si.lastInfo = server.ServerInfo{}
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
func (si *ServerInfo) SetInfo(info server.ServerInfo, configuredAddress, password string) {
	si.lastInfo = info

	si.addressLabel.SetText(formatAddress(info.Address, configuredAddress))

	if info.SourceTV.Address != "" {
		tvText := fmt.Sprintf("%s, delay %s", info.SourceTV.Address, info.SourceTV.Delay)
		if isAddressUsable(info.SourceTV.Local) {
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
func (si *ServerInfo) LastInfo() server.ServerInfo { return si.lastInfo }

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
		connect := fmt.Sprintf("connect %s", gameAddr)
		if password != "" {
			connect += fmt.Sprintf("; password %s", password)
		}
		si.connectLabel.SetText(connect)
		si.copyConnectButton.Enable()
	}

	stvAddr := info.STVConnectAddress()
	if stvAddr == "" {
		si.stvLabel.SetText("")
		si.copySTVButton.Disable()
		return
	}

	stv := fmt.Sprintf("connect %s", stvAddr)
	if password != "" {
		stv += fmt.Sprintf("; password %s", password)
	}
	si.stvLabel.SetText(stv)
	si.copySTVButton.Enable()
}

// formatAddress returns a multi-line summary of the server's addresses,
// omitting empty or unknown placeholders.
func formatAddress(a server.Address, configured string) string {
	parts := []string{}
	if isAddressUsable(a.SDR) {
		parts = append(parts, fmt.Sprintf("SDR: %s", a.SDR))
	}
	if isAddressUsable(configured) && configured != a.SDR {
		parts = append(parts, fmt.Sprintf("Local: %s", configured))
	}
	if isAddressUsable(a.Public) {
		parts = append(parts, fmt.Sprintf("Public: %s", a.Public))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}

// isAddressUsable reports whether an address string is a real endpoint and not
// an empty or unknown placeholder like "?.?.?.?:?" or "0.0.0.0:27015".
func isAddressUsable(s string) bool {
	if s == "" {
		return false
	}
	host, _, err := net.SplitHostPort(s)
	if err != nil {
		host = s
	}
	return host != "0.0.0.0" && !strings.Contains(host, "?")
}
