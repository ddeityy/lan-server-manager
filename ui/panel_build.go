package ui

import (
	"fmt"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/config"
)

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

const sidebarMinWidth = float32(320)

// minWidthLayout wraps a single child and enforces a minimum width.
type minWidthLayout struct {
	width float32
}

func (l *minWidthLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(size)
}

func (l *minWidthLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(l.width, 0)
	}
	s := objects[0].MinSize()
	if s.Width < l.width {
		s.Width = l.width
	}
	return s
}

func withMinWidth(obj fyne.CanvasObject, width float32) fyne.CanvasObject {
	return container.New(&minWidthLayout{width: width}, obj)
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

func (p *ServerPanel) buildUI(title string) {
	p.initWidgets(title)

	connectionBox := p.buildConnectionBox()
	actionBox := p.buildActionsBox()
	serverInfoCard := p.buildServerInfoCard()

	sidebar := container.NewVBox(
		widget.NewCard("Server", "", p.serverNameLabel),
		widget.NewCard("Connection", "", connectionBox),
		widget.NewCard("Actions", "", actionBox),
	)

	topRow := container.NewBorder(nil, nil, withMinWidth(sidebar, sidebarMinWidth), nil, serverInfoCard)
	bottomRow := container.NewBorder(p.kickAllButton, nil, nil, nil, p.playersAccordion)

	content := container.NewBorder(topRow, nil, nil, nil, bottomRow)
	p.tabItem = container.NewTabItem(title, content)
}

func (p *ServerPanel) initWidgets(title string) {
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

	p.addressLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.sourceTVLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.mapLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.playersLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.connectLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	p.stvLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

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

	p.playerList = newPlayerList()
	p.playersAccordion = widget.NewAccordion(
		widget.NewAccordionItem("Players", container.NewBorder(playerHeader(), nil, nil, nil, p.playerList)),
	)
	p.playersAccordion.Open(0)
}

func (p *ServerPanel) buildConnectionBox() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Address   ", p.addressEntry),
			widget.NewFormItem("Password", p.passwordEntry),
		),
		container.NewGridWithColumns(2, p.connectButton, p.disconnectButton),
		p.statusLabel,
	)
}

func (p *ServerPanel) buildActionsBox() fyne.CanvasObject {
	return container.NewVBox(
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
}

func (p *ServerPanel) buildAutoRefreshRow() fyne.CanvasObject {
	p.refreshIntervalEntry.SetPlaceHolder("seconds")
	return container.NewHBox(
		p.autoRefreshCheck,
		widget.NewLabel("Interval (s)"),
		p.refreshIntervalEntry,
	)
}

func (p *ServerPanel) buildServerInfoCard() fyne.CanvasObject {
	p.copyConnectButton = widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() { p.copyConnectString() })
	p.copyConnectButton.Disable()
	p.copySTVButton = widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() { p.copySTVString() })
	p.copySTVButton.Disable()

	addressTile := widget.NewCard("Address", "", p.addressLabel)
	mapTile := widget.NewCard("Map", "", p.mapLabel)
	playersTile := widget.NewCard("Players", "", p.playersLabel)
	sourceTVTile := widget.NewCard("SourceTV", "", p.sourceTVLabel)
	connectTile := widget.NewCard("Connect", "", container.NewBorder(nil, nil, nil, p.copyConnectButton, p.connectLabel))
	stvTile := widget.NewCard("STV", "", container.NewBorder(nil, nil, nil, p.copySTVButton, p.stvLabel))

	serverInfoHeader := container.NewHBox(
		widget.NewLabelWithStyle("Server Info", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.refreshButton,
		p.buildAutoRefreshRow(),
	)

	return widget.NewCard("", "", container.NewVBox(
		serverInfoHeader,
		mapTile,
		playersTile,
		addressTile,
		sourceTVTile,
		stvTile,
		connectTile,
		p.serverInfoStatusLabel,
	))
}
