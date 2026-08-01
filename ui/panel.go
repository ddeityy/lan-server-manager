package ui

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/server"
)

const defaultRefreshInterval = 1

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

	addressLabel      *widget.Label
	sourceTVLabel     *widget.Label
	mapLabel          *widget.Label
	playersLabel      *widget.Label
	connectLabel      *widget.Label
	stvLabel          *widget.Label
	copyConnectButton *widget.Button
	copySTVButton     *widget.Button
	playerList        *widget.List

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

	playersAccordion *widget.Accordion

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
