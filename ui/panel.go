package ui

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"lan-server-manager/server"
)

const defaultRefreshInterval = 1

// ServerPanel is the full UI for a single TF2 server connection.
type ServerPanel struct {
	window fyne.Window
	server *server.Server

	connection *Connection
	actions    *Actions
	serverInfo *ServerInfo
	players    *PlayerSection

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
	p := &ServerPanel{
		window:         window,
		onTitleChanged: onTitleChanged,
		onChanged:      onChanged,
	}

	p.connection = newConnection(p.connect, p.disconnect, p.notifyChanged)
	p.actions = newActions(
		p.changeServerPassword,
		p.changeLevel,
		p.execConfig,
		p.sendMessage,
		p.sendCustomCommand,
		p.handleMapSelected,
		p.handleConfigSelected,
		p.notifyChanged,
	)
	p.serverInfo = newServerInfo(
		title,
		p.copyConnectString,
		p.copySTVString,
		p.handleIntervalChanged,
	)
	p.players = newPlayerSection(p.confirmKickAll)

	p.buildUI(title)
	return p
}

// TabItem returns the tab data for this panel.
func (p *ServerPanel) TabItem() *container.TabItem {
	return p.tabItem
}

func (p *ServerPanel) updateTitle(title string) {
	p.tabItem.Text = title
	p.serverInfo.SetName(title)
	if p.onTitleChanged != nil {
		p.onTitleChanged()
	}
}

func (p *ServerPanel) notifyChanged() {
	if p.onChanged != nil {
		p.onChanged()
	}
}

func (p *ServerPanel) handleMapSelected() {
	p.notifyChanged()
}

func (p *ServerPanel) handleConfigSelected() {
	p.notifyChanged()
}

func (p *ServerPanel) handleIntervalChanged() {
	if p.server == nil {
		return
	}
	p.startAutoRefresh()
}
