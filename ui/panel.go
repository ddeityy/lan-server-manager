package ui

import (
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/config"
	"lan-server-manager/server"
)

const defaultRefreshInterval = 1

func mapList() []string {
	if len(appConfig.Maps) > 0 {
		return appConfig.Maps
	}
	return config.Default().Maps
}

func configList() []string {
	if len(appConfig.Configs) > 0 {
		return appConfig.Configs
	}
	return config.Default().Configs
}

const actionRowGap = float32(8)

// actionRowLayout sizes two children in a 3:1 ratio (75% input / 25% button)
// with a small gap between them.
type actionRowLayout struct{}

func (l *actionRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 2 {
		return
	}
	available := size.Width - actionRowGap
	leftW := available * 0.75
	rightW := available * 0.25
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(leftW, size.Height))
	objects[1].Move(fyne.NewPos(leftW+actionRowGap, 0))
	objects[1].Resize(fyne.NewSize(rightW, size.Height))
}

func (l *actionRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var h, totalW float32
	for i, obj := range objects {
		if mh := obj.MinSize().Height; mh > h {
			h = mh
		}
		totalW += obj.MinSize().Width
		if i < len(objects)-1 {
			totalW += actionRowGap
		}
	}
	return fyne.NewSize(totalW, h)
}

func newActionRow(left, right fyne.CanvasObject) *fyne.Container {
	return container.New(&actionRowLayout{}, left, right)
}

// setMapSelection sets the map dropdown's current value and renders the
// remaining pool options so the currently selected map is not shown twice.
func setMapSelection(sel *widget.Select, value string) {
	maps := mapList()
	valid := slices.Contains(maps, value)
	if !valid {
		sel.Selected = ""
		sel.Options = append([]string(nil), maps...)
		sel.Refresh()
		return
	}

	opts := make([]string, 0, len(maps)-1)
	for _, m := range maps {
		if m != value {
			opts = append(opts, m)
		}
	}
	sel.Selected = value
	sel.Options = opts
	sel.Refresh()
}

// ServerPanel is the full UI for a single TF2 server connection.
type ServerPanel struct {
	window fyne.Window
	server *server.Server

	lastInfo server.ServerInfo

	addressEntry          *widget.Entry
	passwordEntry         *widget.Entry
	statusLabel           *widget.Label
	actionsStatusLabel    *widget.Label
	serverInfoStatusLabel *widget.Label
	serverNameLabel       *widget.Label

	autoRefreshCheck     *widget.Check
	refreshIntervalEntry *widget.Entry

	addressLabel     *widget.Label
	sourceTVLabel    *widget.Label
	mapLabel         *widget.Label
	playersLabel     *widget.Label
	connectLabel     *widget.Label
	stvLabel         *widget.Label
	playerList       *widget.List
	playersAccordion *widget.Accordion

	connectButton        *widget.Button
	disconnectButton     *widget.Button
	refreshButton        *widget.Button
	changeLevelButton    *widget.Button
	changePasswordButton *widget.Button
	execConfigButton     *widget.Button
	customCommandButton  *widget.Button
	kickAllButton        *widget.Button

	mapSelect           *widget.Select
	configSelect        *widget.Select
	customCommandEntry  *widget.Entry
	serverPasswordEntry *widget.Entry

	refreshMutex   sync.Mutex
	refreshTicker  *time.Ticker
	refreshStop    chan struct{}
	pendingMapSync bool

	tabItem        *container.TabItem
	onTitleChanged func()
	onChanged      func()
}

// NewServerPanel creates a new tab panel with default connection values.
func NewServerPanel(window fyne.Window, title string, onTitleChanged, onChanged func()) *ServerPanel {
	p := &ServerPanel{window: window, onTitleChanged: onTitleChanged, onChanged: onChanged}
	p.buildUI(title)
	return p
}

// TabItem returns the tab data for this panel.
func (p *ServerPanel) TabItem() *container.TabItem {
	return p.tabItem
}

func (p *ServerPanel) buildUI(title string) {
	p.addressEntry = widget.NewEntry()
	p.addressEntry.SetText("0.0.0.0:27015")
	p.addressEntry.OnChanged = func(string) { p.notifyChanged() }

	p.passwordEntry = widget.NewPasswordEntry()
	p.passwordEntry.SetText("test")
	p.passwordEntry.OnChanged = func(string) { p.notifyChanged() }

	p.serverPasswordEntry = widget.NewPasswordEntry()
	p.serverPasswordEntry.OnChanged = func(string) { p.notifyChanged() }

	p.autoRefreshCheck = widget.NewCheck("Auto refresh", p.handleAutoRefreshChanged)
	p.autoRefreshCheck.Checked = true
	p.autoRefreshCheck.Refresh()

	p.refreshIntervalEntry = widget.NewEntry()
	p.refreshIntervalEntry.SetText(fmt.Sprintf("%d", defaultRefreshInterval))
	restartAutoRefresh := func(string) {
		if p.autoRefreshCheck.Checked {
			p.startAutoRefresh()
		}
	}
	p.refreshIntervalEntry.OnSubmitted = restartAutoRefresh
	p.refreshIntervalEntry.OnChanged = restartAutoRefresh

	p.statusLabel = widget.NewLabel("")
	p.actionsStatusLabel = widget.NewLabel("")
	p.serverInfoStatusLabel = widget.NewLabel("")
	p.serverNameLabel = widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	p.addressLabel = widget.NewLabelWithStyle("-", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.sourceTVLabel = widget.NewLabelWithStyle("-", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.mapLabel = widget.NewLabelWithStyle("-", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.playersLabel = widget.NewLabelWithStyle("-", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.connectLabel = widget.NewLabelWithStyle("-", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.stvLabel = widget.NewLabelWithStyle("-", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Player list.
	p.playerList = newPlayerList()

	p.playersAccordion = widget.NewAccordion(
		widget.NewAccordionItem("Players", container.NewBorder(playerHeader(), nil, nil, nil, p.playerList)),
	)
	p.playersAccordion.Open(0)

	// Buttons.
	p.disconnectButton = widget.NewButton("Disconnect", func() { p.disconnect() })
	p.disconnectButton.Disable()

	p.connectButton = widget.NewButton("Connect", func() { p.connect() })

	p.refreshButton = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() { p.refresh() })
	p.refreshButton.Disable()

	p.mapSelect = widget.NewSelect(mapList(), p.handleMapSelected)
	setMapSelection(p.mapSelect, mapList()[0])

	p.configSelect = widget.NewSelect(configList(), func(string) { p.notifyChanged() })
	p.configSelect.SetSelected(configList()[0])

	p.changeLevelButton = widget.NewButton("Send", func() { p.changeLevel() })
	p.changeLevelButton.Disable()

	p.execConfigButton = widget.NewButton("Send", func() { p.execConfig() })
	p.execConfigButton.Disable()

	p.customCommandEntry = widget.NewEntry()

	p.customCommandButton = widget.NewButton("Send", func() { p.sendCustomCommand() })
	p.customCommandButton.Disable()

	p.changePasswordButton = widget.NewButton("Send", func() { p.changeServerPassword() })
	p.changePasswordButton.Disable()

	p.kickAllButton = widget.NewButton("Kick All Players", func() { p.confirmKickAll() })
	p.kickAllButton.Disable()

	p.refreshIntervalEntry.SetPlaceHolder("seconds")
	autoRefreshRow := container.NewHBox(
		p.autoRefreshCheck,
		widget.NewLabel("Interval (s)"),
		p.refreshIntervalEntry,
	)

	connectionBox := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Address   ", p.addressEntry),
			widget.NewFormItem("Password", p.passwordEntry),
		),
		container.NewGridWithColumns(2, p.connectButton, p.disconnectButton),
		p.statusLabel,
	)

	actionBox := container.NewVBox(
		widget.NewLabelWithStyle("Change server password", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(p.serverPasswordEntry, p.changePasswordButton),
		widget.NewLabelWithStyle("Change map", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(p.mapSelect, p.changeLevelButton),
		widget.NewLabelWithStyle("Exec config", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(p.configSelect, p.execConfigButton),
		widget.NewLabelWithStyle("Custom command", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newActionRow(p.customCommandEntry, p.customCommandButton),
		p.actionsStatusLabel,
	)

	sidebar := container.NewVBox(
		widget.NewCard("Server", "", p.serverNameLabel),
		widget.NewCard("Connection", "", connectionBox),
		widget.NewCard("Actions", "", actionBox),
	)

	copyConnect := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() { p.copyConnectString() })
	copySTV := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() { p.copySTVString() })

	addressTile := widget.NewCard("Address", "", p.addressLabel)
	mapTile := widget.NewCard("Map", "", p.mapLabel)
	playersTile := widget.NewCard("Players", "", p.playersLabel)
	sourceTVTile := widget.NewCard("SourceTV", "", p.sourceTVLabel)
	connectTile := widget.NewCard("Connect", "", container.NewBorder(nil, nil, nil, copyConnect, p.connectLabel))
	stvTile := widget.NewCard("STV", "", container.NewBorder(nil, nil, nil, copySTV, p.stvLabel))

	serverInfoHeader := container.NewHBox(
		widget.NewLabelWithStyle("Server Info", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.refreshButton,
		autoRefreshRow,
	)

	serverInfoCard := widget.NewCard("", "", container.NewVBox(
		serverInfoHeader,
		mapTile,
		playersTile,
		addressTile,
		sourceTVTile,
		stvTile,
		connectTile,
		p.serverInfoStatusLabel,
	))

	topRow := container.NewBorder(nil, nil, sidebar, nil, serverInfoCard)
	bottomRow := container.NewBorder(p.kickAllButton, nil, nil, nil, p.playersAccordion)

	content := container.NewBorder(topRow, nil, nil, nil, bottomRow)
	p.tabItem = container.NewTabItem(title, content)
}

func (p *ServerPanel) updateTitle(title string) {
	p.tabItem.Text = title
	p.serverNameLabel.SetText(title)
	if p.onTitleChanged != nil {
		p.onTitleChanged()
	}
}

func (p *ServerPanel) notifyChanged() {
	if p.onChanged != nil {
		p.onChanged()
	}
}

func (p *ServerPanel) handleMapSelected(value string) {
	setMapSelection(p.mapSelect, value)
	p.notifyChanged()
}

func (p *ServerPanel) handleAutoRefreshChanged(checked bool) {
	p.notifyChanged()

	if checked {
		p.refreshButton.Disable()
		if p.server != nil {
			p.startAutoRefresh()
		}
		return
	}

	p.stopAutoRefresh()
	if p.server != nil {
		p.refreshButton.Enable()
	}
}

func (p *ServerPanel) setConnected(connected bool) {
	if connected {
		p.connectButton.Disable()
		p.disconnectButton.Enable()
		p.refreshButton.Enable()
		p.changeLevelButton.Enable()
		if p.autoRefreshCheck.Checked {
			p.refreshButton.Disable()
		}
		p.changePasswordButton.Enable()
		p.execConfigButton.Enable()
		p.customCommandButton.Enable()
		p.kickAllButton.Enable()
		p.statusLabel.SetText("Connected")
		p.pendingMapSync = true
		if p.autoRefreshCheck.Checked {
			p.startAutoRefresh()
		}
		return
	}
	p.stopAutoRefresh()
	p.connectButton.Enable()
	p.disconnectButton.Disable()
	p.refreshButton.Disable()
	p.changeLevelButton.Disable()
	p.changePasswordButton.Disable()
	p.execConfigButton.Disable()
	p.customCommandButton.Disable()
	p.kickAllButton.Disable()
}

func (p *ServerPanel) resetInfo() {
	p.lastInfo = server.ServerInfo{}

	p.addressLabel.SetText("-")
	p.mapLabel.SetText("-")
	p.playersLabel.SetText("-")
	p.sourceTVLabel.SetText("-")
	p.connectLabel.SetText("-")
	p.stvLabel.SetText("-")
	p.serverInfoStatusLabel.SetText("")

	p.playerList.Length = func() int { return 0 }
	p.playerList.UpdateItem = func(_ widget.ListItemID, _ fyne.CanvasObject) {}
	p.playerList.Refresh()

	p.playersAccordion.Items[0].Title = "Players"
	p.playersAccordion.Refresh()
}

func (p *ServerPanel) updateInfo(info server.ServerInfo, err error) {
	if err != nil {
		p.serverInfoStatusLabel.SetText(fmt.Sprintf("Refresh failed: %v", err))
		return
	}

	p.lastInfo = info

	p.addressLabel.SetText(formatAddress(info.Address, info.ConfiguredAddress))
	if info.SourceTV.Address != "" {
		tvText := fmt.Sprintf("%s, delay %s", info.SourceTV.Address, info.SourceTV.Delay)
		if isAddressUsable(info.SourceTV.Local) {
			tvText += "\nlocal " + info.SourceTV.Local
		}
		p.sourceTVLabel.SetText(tvText)
	} else {
		p.sourceTVLabel.SetText("-")
	}
	p.mapLabel.SetText(info.Map)
	p.playersLabel.SetText(fmt.Sprintf("%d / %d", info.HumanPlayers, info.MaxPlayers))

	p.updateConnectStrings()

	if p.pendingMapSync {
		p.pendingMapSync = false
		setMapSelection(p.mapSelect, info.Map)
		p.notifyChanged()
	}

	if info.Hostname != "" {
		p.updateTitle(info.Hostname)
	}

	updatePlayerList(p.playerList, info.Players, p.kick)

	p.playersAccordion.Items[0].Title = fmt.Sprintf("Players (%d)", len(info.Players))
	p.playersAccordion.Refresh()

	p.kickAllButton.Disable()
	if len(info.Players) > 0 {
		p.kickAllButton.Enable()
	}

	p.serverInfoStatusLabel.SetText("Connected")
}

// updateConnectStrings rebuilds the connect and stv copy strings from the
// current server info and password entry.
func (p *ServerPanel) updateConnectStrings() {
	gameAddr := p.lastInfo.GameConnectAddress()
	if gameAddr == "" {
		p.connectLabel.SetText("-")
		p.stvLabel.SetText("-")
		return
	}

	password := p.serverPasswordEntry.Text
	connect := fmt.Sprintf("connect %s", gameAddr)
	if password != "" {
		connect += fmt.Sprintf("; password %s", password)
	}
	p.connectLabel.SetText(connect)

	stvAddr := p.lastInfo.STVConnectAddress()
	if stvAddr == "" {
		p.stvLabel.SetText("-")
		return
	}

	stv := fmt.Sprintf("connect %s", stvAddr)
	if password != "" {
		stv += fmt.Sprintf("; password %s", password)
	}
	p.stvLabel.SetText(stv)
}

// formatAddress returns a multi-line summary of the server's addresses for the
// Address info tile, omitting empty or unknown placeholders.
func formatAddress(a server.Address, configured string) string {
	parts := []string{}
	if isAddressUsable(a.SDR) {
		parts = append(parts, fmt.Sprintf("SDR: %s", a.SDR))
	}
	if isAddressUsable(a.Local) && a.Local != configured {
		parts = append(parts, fmt.Sprintf("Local: %s", a.Local))
	}
	if isAddressUsable(configured) && configured != a.SDR {
		parts = append(parts, fmt.Sprintf("Connect: %s", configured))
	}
	if isAddressUsable(a.Public) {
		parts = append(parts, fmt.Sprintf("Public: %s", a.Public))
	}
	if len(parts) == 0 {
		return "-"
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

func (p *ServerPanel) copyConnectString() {
	fyne.CurrentApp().Clipboard().SetContent(p.connectLabel.Text)
	p.serverInfoStatusLabel.SetText("Connect string copied")
}

func (p *ServerPanel) copySTVString() {
	fyne.CurrentApp().Clipboard().SetContent(p.stvLabel.Text)
	p.serverInfoStatusLabel.SetText("STV string copied")
}

// parseRefreshInterval reads the interval entry, returning seconds.
// Invalid or too-small values fall back to the default.
func (p *ServerPanel) parseRefreshInterval() int {
	s := p.refreshIntervalEntry.Text
	if s == "" {
		return defaultRefreshInterval
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return defaultRefreshInterval
	}
	return n
}

func (p *ServerPanel) startAutoRefresh() {
	if p.server == nil || !p.autoRefreshCheck.Checked {
		return
	}
	p.stopAutoRefresh()

	interval := time.Duration(p.parseRefreshInterval()) * time.Second
	p.refreshTicker = time.NewTicker(interval)
	p.refreshStop = make(chan struct{})

	go func() {
		for {
			select {
			case <-p.refreshStop:
				return
			case <-p.refreshTicker.C:
				info, err := p.doRefresh()
				fyne.Do(func() { p.updateInfo(info, err) })
			}
		}
	}()
}

func (p *ServerPanel) stopAutoRefresh() {
	if p.refreshTicker != nil {
		p.refreshTicker.Stop()
		p.refreshTicker = nil
	}
	if p.refreshStop != nil {
		close(p.refreshStop)
		p.refreshStop = nil
	}
}

// doRefresh performs a synchronous status refresh. It serializes access to the
// server so manual and automatic refreshes cannot overlap.
func (p *ServerPanel) doRefresh() (server.ServerInfo, error) {
	p.refreshMutex.Lock()
	defer p.refreshMutex.Unlock()

	if p.server == nil {
		return server.ServerInfo{}, fmt.Errorf("not connected")
	}
	if err := p.server.Refresh(); err != nil {
		return server.ServerInfo{}, err
	}
	return p.server.Info(), nil
}
