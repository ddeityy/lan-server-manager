package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/server"
)

var mapPool = []string{
	"cp_sunshine",
	"cp_process_f12",
	"cp_gullywash_f9",
	"cp_metalworks_f7",
	"koth_bagel_rc12",
	"koth_product_final",
	"cp_granary_pro_rc17a3",
	"cp_badlands",
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

// ServerPanel is the full UI for a single TF2 server connection.
type ServerPanel struct {
	window fyne.Window
	server *server.Server

	lastInfo server.ServerInfo

	addressEntry  *widget.Entry
	passwordEntry *widget.Entry
	statusLabel   *widget.Label

	ipLabel          *widget.Label
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

	tabItem     *container.TabItem
	onTitleChan func()
	onChanged   func()
}

// NewServerPanel creates a new tab panel with default connection values.
func NewServerPanel(window fyne.Window, title string, onTitleChanged, onChanged func()) *ServerPanel {
	p := &ServerPanel{window: window, onTitleChan: onTitleChanged, onChanged: onChanged}
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

	p.statusLabel = widget.NewLabel("")

	regionLabel := widget.NewLabelWithStyle("Server", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.ipLabel = widget.NewLabel("Address: -")
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

	p.mapSelect = widget.NewSelect(mapPool, func(string) { p.notifyChanged() })
	p.mapSelect.SetSelected("cp_badlands")
	// Set a placeholder wide enough so any map name fits without clipping.
	p.mapSelect.PlaceHolder = longestMapName()

	p.changeLevelButton = widget.NewButton("Change Level", func() { p.changeLevel() })
	p.changeLevelButton.Disable()

	p.kickAllButton = widget.NewButton("Kick All Players", func() { p.confirmKickAll() })
	p.kickAllButton.Disable()

	connectionBar := container.NewHBox(p.connectButton, p.disconnectButton)
	actionBar := container.NewHBox(p.mapSelect, p.changeLevelButton)

	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Address", p.addressEntry),
			widget.NewFormItem("Password", p.passwordEntry),
		),
		connectionBar,
		actionBar,
		p.statusLabel,
	)

	serverHeader := container.NewHBox(regionLabel, p.refreshButton)

	serverCard := container.NewVBox(
		serverHeader,
		p.ipLabel,
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
	if p.onTitleChan != nil {
		p.onTitleChan()
	}
}

func (p *ServerPanel) notifyChanged() {
	if p.onChanged != nil {
		p.onChanged()
	}
}

func (p *ServerPanel) setConnected(connected bool) {
	if connected {
		p.connectButton.Disable()
		p.disconnectButton.Enable()
		p.refreshButton.Enable()
		p.changeLevelButton.Enable()
		p.kickAllButton.Enable()
		p.statusLabel.SetText("Connected")
		return
	}
	p.connectButton.Enable()
	p.disconnectButton.Disable()
	p.refreshButton.Disable()
	p.changeLevelButton.Disable()
	p.kickAllButton.Disable()
	p.statusLabel.SetText("Disconnected")
}

func (p *ServerPanel) resetInfo() {
	p.lastInfo = server.ServerInfo{}

	p.ipLabel.SetText("Address: -")
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

	p.ipLabel.SetText("Address: " + info.Address)
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
