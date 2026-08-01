package ui

import (
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/config"
	"lan-server-manager/server"
)

const defaultRefreshInterval = 10

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

func longestMapName() string {
	return appConfig.LongestName(mapList())
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
	kickAllButton        *widget.Button

	mapSelect           *widget.Select
	configSelect        *widget.Select
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

	p.refreshIntervalEntry = widget.NewEntry()
	p.refreshIntervalEntry.SetText(fmt.Sprintf("%d", defaultRefreshInterval))
	p.refreshIntervalEntry.OnSubmitted = func(string) {
		if p.autoRefreshCheck.Checked {
			p.startAutoRefresh()
		}
	}

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
	p.playerList = widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("ID"),
				widget.NewLabel("Name"),
				widget.NewLabel("UniqueID"),
				widget.NewLabel("Conn"),
				widget.NewLabel("Ping"),
				widget.NewLabel("Loss"),
				widget.NewLabel("State"),
				widget.NewButton("Kick", nil),
			)
		},
		func(_ widget.ListItemID, _ fyne.CanvasObject) {},
	)

	p.playersAccordion = widget.NewAccordion(
		widget.NewAccordionItem("Players", p.playerList),
	)

	// Buttons.
	p.disconnectButton = widget.NewButton("Disconnect", func() { p.disconnect() })
	p.disconnectButton.Disable()

	p.connectButton = widget.NewButton("Connect", func() { p.connect() })

	p.refreshButton = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() { p.refresh() })
	p.refreshButton.Disable()

	p.mapSelect = widget.NewSelect(mapList(), p.handleMapSelected)
	setMapSelection(p.mapSelect, mapList()[0])
	p.mapSelect.PlaceHolder = longestMapName()

	p.changeLevelButton = widget.NewButton("Change map", func() { p.changeLevel() })
	p.changeLevelButton.Disable()

	p.configSelect = widget.NewSelect(configList(), func(string) { p.notifyChanged() })
	p.configSelect.SetSelected(configList()[0])
	// Size the config dropdown to match the map dropdown so the action rows
	// line up. The placeholder only affects min-width; the selected config is
	// still displayed.
	p.configSelect.PlaceHolder = longestMapName()

	p.execConfigButton = widget.NewButton("Exec config", func() { p.execConfig() })
	p.execConfigButton.Disable()

	p.changePasswordButton = widget.NewButton("Set password", func() { p.changeServerPassword() })
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
		container.NewGridWithColumns(2, p.serverPasswordEntry, p.changePasswordButton),
		container.NewGridWithColumns(2, p.mapSelect, p.changeLevelButton),
		container.NewGridWithColumns(2, p.configSelect, p.execConfigButton),
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
		container.NewGridWithColumns(2, mapTile, playersTile),
		container.NewGridWithColumns(2, addressTile, sourceTVTile),
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
	p.kickAllButton.Disable()
	p.statusLabel.SetText("Disconnected")
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

	p.addressLabel.SetText(info.Address)
	if info.SourceTV.Address != "" {
		tvText := fmt.Sprintf("%s (%s)", info.SourceTV.Address, info.SourceTV.Delay)
		if info.SourceTV.Local != "" {
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

	p.playerList.Length = func() int { return len(info.Players) }
	p.playerList.UpdateItem = func(i widget.ListItemID, o fyne.CanvasObject) {
		player := info.Players[i]
		row := o.(*fyne.Container)
		row.Objects[0].(*widget.Label).SetText(fmt.Sprintf("%d", player.UserID))
		row.Objects[1].(*widget.Label).SetText(player.Name)
		row.Objects[2].(*widget.Label).SetText(player.UniqueID)
		row.Objects[3].(*widget.Label).SetText(player.Connected)
		row.Objects[4].(*widget.Label).SetText(fmt.Sprintf("%d", player.Ping))
		row.Objects[5].(*widget.Label).SetText(fmt.Sprintf("%d", player.Loss))
		row.Objects[6].(*widget.Label).SetText(player.State)
		btn := row.Objects[7].(*widget.Button)
		btn.SetText("Kick")
		btn.OnTapped = func() { p.kick(player.UserID) }
		btn.Enable()
	}
	p.playerList.Refresh()

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
	if p.lastInfo.Address == "" {
		p.connectLabel.SetText("-")
		p.stvLabel.SetText("-")
		return
	}

	password := p.serverPasswordEntry.Text
	connect := fmt.Sprintf("connect %s", p.lastInfo.Address)
	if password != "" {
		connect += fmt.Sprintf("; password %s", password)
	}
	p.connectLabel.SetText(connect)

	stv := fmt.Sprintf("connect %s", p.lastInfo.SourceTV.Address)
	if password != "" {
		stv += fmt.Sprintf("; password %s", password)
	}
	if p.lastInfo.SourceTV.Address == "" {
		stv = "-"
	}
	p.stvLabel.SetText(stv)
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
