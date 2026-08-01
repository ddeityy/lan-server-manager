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

	"lan-server-manager/server"
)

const defaultRefreshInterval = 10

var mapPool = []string{
	"cp_badlands",
	"cp_sunshine",
	"cp_process_f12",
	"cp_gullywash_f9",
	"cp_metalworks_f7",
	"koth_bagel_rc12",
	"koth_product_final",
	"cp_granary_pro_rc17a3",
	"mge_training_v8_beta4b",
}

func longestMapName() string {
	longest := ""
	for _, m := range mapPool {
		if len(m) > len(longest) {
			longest = m
		}
	}
	return longest
}

// setMapSelection sets the map dropdown's current value and renders the
// remaining pool options so the currently selected map is not shown twice.
func setMapSelection(sel *widget.Select, value string) {
	valid := slices.Contains(mapPool, value)
	if !valid {
		sel.Selected = ""
		sel.Options = append([]string(nil), mapPool...)
		sel.Refresh()
		return
	}

	opts := make([]string, 0, len(mapPool)-1)
	for _, m := range mapPool {
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

	addressEntry  *widget.Entry
	passwordEntry *widget.Entry
	statusLabel   *widget.Label

	autoRefreshCheck     *widget.Check
	refreshIntervalEntry *widget.Entry

	addressLabel     *widget.Label
	sourceTVLabel    *widget.Label
	mapLabel         *widget.Label
	playersLabel     *widget.Label
	playerList       *widget.List
	playersAccordion *widget.Accordion

	connectButton     *widget.Button
	disconnectButton  *widget.Button
	refreshButton     *widget.Button
	changeLevelButton *widget.Button
	kickAllButton     *widget.Button
	mapSelect         *widget.Select

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

	p.autoRefreshCheck = widget.NewCheck("Auto refresh", func(checked bool) {
		if checked && p.server != nil {
			p.startAutoRefresh()
		} else {
			p.stopAutoRefresh()
		}
	})

	p.refreshIntervalEntry = widget.NewEntry()
	p.refreshIntervalEntry.SetText(fmt.Sprintf("%d", defaultRefreshInterval))
	p.refreshIntervalEntry.OnSubmitted = func(string) {
		if p.autoRefreshCheck.Checked {
			p.startAutoRefresh()
		}
	}

	p.statusLabel = widget.NewLabel("")

	serverSectionLabel := widget.NewLabelWithStyle("Server", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.addressLabel = widget.NewLabel("Address: -")
	p.sourceTVLabel = widget.NewLabel("SourceTV: -")
	p.mapLabel = widget.NewLabel("Map: -")
	p.playersLabel = widget.NewLabel("Players: -")

	// Player list.
	p.playerList = widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("ID"),
				widget.NewLabel("Name"),
				widget.NewLabel("UniqueID"),
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

	p.mapSelect = widget.NewSelect(mapPool, p.handleMapSelected)
	setMapSelection(p.mapSelect, mapPool[0])
	// Set a placeholder wide enough so any map name fits without clipping.
	p.mapSelect.PlaceHolder = longestMapName()

	p.changeLevelButton = widget.NewButton("Change map", func() { p.changeLevel() })
	p.changeLevelButton.Disable()

	p.kickAllButton = widget.NewButton("Kick All Players", func() { p.confirmKickAll() })
	p.kickAllButton.Disable()

	connectionBar := container.NewHBox(p.connectButton)
	actionBar := container.NewHBox(p.mapSelect, p.changeLevelButton)

	p.refreshIntervalEntry.SetPlaceHolder("seconds")
	autoRefreshRow := container.NewHBox(
		p.autoRefreshCheck,
		widget.NewLabel("Interval (s)"),
		p.refreshIntervalEntry,
	)

	form := container.NewVBox(
		connectionBar,
		widget.NewForm(
			widget.NewFormItem("Address", p.addressEntry),
			widget.NewFormItem("Password", p.passwordEntry),
		),
		actionBar,
		p.statusLabel,
		autoRefreshRow,
	)

	serverHeader := container.NewHBox(serverSectionLabel, p.refreshButton)

	serverCard := container.NewVBox(
		serverHeader,
		p.addressLabel,
		p.sourceTVLabel,
		p.mapLabel,
		p.playersLabel,
		p.kickAllButton,
		p.playersAccordion,
	)

	content := container.NewBorder(form, nil, nil, nil, serverCard)
	p.tabItem = container.NewTabItem(title, content)
}

func (p *ServerPanel) updateTitle(title string) {
	p.tabItem.Text = title
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

func (p *ServerPanel) setConnected(connected bool) {
	if connected {
		p.connectButton.Disable()
		p.disconnectButton.Enable()
		p.refreshButton.Enable()
		p.changeLevelButton.Enable()
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
	p.kickAllButton.Disable()
	p.statusLabel.SetText("Disconnected")
}

func (p *ServerPanel) resetInfo() {
	p.lastInfo = server.ServerInfo{}

	p.addressLabel.SetText("Address: -")
	p.mapLabel.SetText("Map: -")
	p.playersLabel.SetText("Players: -")
	p.sourceTVLabel.SetText("SourceTV: -")

	p.playerList.Length = func() int { return 0 }
	p.playerList.UpdateItem = func(_ widget.ListItemID, _ fyne.CanvasObject) {}
	p.playerList.Refresh()

	p.playersAccordion.Items[0].Title = "Players"
	p.playersAccordion.Refresh()
}

func (p *ServerPanel) updateInfo(info server.ServerInfo, err error) {
	if err != nil {
		p.statusLabel.SetText(fmt.Sprintf("Refresh failed: %v", err))
		return
	}

	p.lastInfo = info

	p.addressLabel.SetText("Address: " + info.Address)
	if info.SourceTV.Address != "" {
		tvText := fmt.Sprintf("SourceTV: %s (%s)", info.SourceTV.Address, info.SourceTV.Delay)
		if info.SourceTV.Local != "" {
			tvText += " local " + info.SourceTV.Local
		}
		p.sourceTVLabel.SetText(tvText)
	} else {
		p.sourceTVLabel.SetText("SourceTV: -")
	}
	p.mapLabel.SetText("Map: " + info.Map)
	p.playersLabel.SetText(fmt.Sprintf("Players: %d / %d", info.HumanPlayers, info.MaxPlayers))

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
		btn := row.Objects[3].(*widget.Button)
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

	p.statusLabel.SetText("Connected")
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
