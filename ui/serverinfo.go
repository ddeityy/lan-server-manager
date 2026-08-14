package ui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/rcon"
)

// ServerInfo holds the widgets that display parsed status output and the
// controls that refresh it.
type ServerInfo struct {
	serverName           string
	connectionCard       *widget.Card
	refreshIntervalEntry *widget.Entry

	lastInfo rcon.ServerInfo
}

func newServerInfo(
	title string,
	onIntervalChanged func(),
) *ServerInfo {
	return &ServerInfo{
		serverName:           title,
		refreshIntervalEntry: newRefreshIntervalEntry(onIntervalChanged),
	}
}

func newRefreshIntervalEntry(onIntervalChanged func()) *widget.Entry {
	entry := widget.NewEntry()
	entry.SetText(fmt.Sprintf("%d", defaultRefreshInterval))
	entry.SetPlaceHolder("seconds")
	entry.OnSubmitted = func(string) { onIntervalChanged() }
	entry.OnChanged = func(string) { onIntervalChanged() }
	return entry
}

// View returns the server info card as a single canvas object.
func (si *ServerInfo) View(connection fyne.CanvasObject) fyne.CanvasObject {
	si.connectionCard = widget.NewCard("Connection", "", connection)

	header := container.NewHBox(
		widget.NewLabelWithStyle("Server Info", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Refresh interval (s):"),
		si.refreshIntervalEntry,
	)

	return widget.NewCard("", "", container.NewVBox(
		header,
		si.connectionCard,
	))
}

// SetName updates the server name used in the connection card title.
func (si *ServerInfo) SetName(name string) {
	si.serverName = name
	si.refreshTitle()
}

// Reset clears all displayed server info.
func (si *ServerInfo) Reset() {
	si.lastInfo = rcon.ServerInfo{}
	si.serverName = ""
	if si.connectionCard != nil {
		si.connectionCard.SetTitle("Connection")
	}
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

// SetInfo updates the displayed server info.
func (si *ServerInfo) SetInfo(info rcon.ServerInfo) {
	si.lastInfo = info
	if info.Hostname != "" {
		si.serverName = info.Hostname
	}
	si.refreshTitle()
}

func (si *ServerInfo) refreshTitle() {
	if si.connectionCard == nil {
		return
	}
	if si.serverName == "" && si.lastInfo.Map == "" {
		si.connectionCard.SetTitle("Connection")
		return
	}
	if si.lastInfo.Map == "" {
		si.connectionCard.SetTitle(si.serverName)
		return
	}
	si.connectionCard.SetTitle(si.serverName + " | " + si.lastInfo.Map)
}

// LastInfo returns the most recently displayed server information.
func (si *ServerInfo) LastInfo() rcon.ServerInfo { return si.lastInfo }
