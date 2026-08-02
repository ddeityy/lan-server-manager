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
	serverNameLabel      *widget.Label
	mapLabel             *widget.Label
	refreshIntervalEntry *widget.Entry

	lastInfo rcon.ServerInfo
}

func newServerInfo(
	title string,
	onIntervalChanged func(),
) *ServerInfo {
	si := &ServerInfo{}

	si.serverNameLabel = widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	si.mapLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	si.refreshIntervalEntry = widget.NewEntry()
	si.refreshIntervalEntry.SetText(fmt.Sprintf("%d", defaultRefreshInterval))
	si.refreshIntervalEntry.SetPlaceHolder("seconds")
	si.refreshIntervalEntry.OnSubmitted = func(string) { onIntervalChanged() }
	si.refreshIntervalEntry.OnChanged = func(string) { onIntervalChanged() }

	return si
}

// View returns the server info card as a single canvas object.
func (si *ServerInfo) View(connection fyne.CanvasObject) fyne.CanvasObject {
	connectionTile := widget.NewCard("Connection", "", connection)
	nameTile := widget.NewCard("Name", "", si.serverNameLabel)
	mapTile := widget.NewCard("Map", "", si.mapLabel)

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
	))
}

// SetName updates the name displayed at the top of the server info card.
func (si *ServerInfo) SetName(name string) { si.serverNameLabel.SetText(name) }

// Reset clears all displayed server info.
func (si *ServerInfo) Reset() {
	si.lastInfo = rcon.ServerInfo{}
	si.serverNameLabel.SetText("")
	si.mapLabel.SetText("")
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
	si.mapLabel.SetText(info.Map)
}

// LastInfo returns the most recently displayed server information.
func (si *ServerInfo) LastInfo() rcon.ServerInfo { return si.lastInfo }
