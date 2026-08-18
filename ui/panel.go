package ui

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"lan-server-manager/game/logparse"
	"lan-server-manager/game/scoreboard"
	"lan-server-manager/rcon"
)

const defaultRefreshInterval = 5

// ServerPanel is the full UI for a single TF2 server connection.
type ServerPanel struct {
	window fyne.Window
	client *rcon.Client
	title  string

	connection     *Connection
	actions        *Actions
	serverInfo     *ServerInfo
	scoreboardView *ScoreboardView
	logs           *LogViewer
	sendMessage    *SendMessage
	cvars          *CVarPanel
	actor          *scoreboard.Actor
	refreshTicker  *time.Ticker
	refreshStop    chan struct{}
	pendingMapSync bool

	tabItem        *container.TabItem
	onTitleChanged func()
}

// NewServerPanel creates a new tab panel with default connection values.
func NewServerPanel(window fyne.Window, title string, onTitleChanged func()) *ServerPanel {
	panel := &ServerPanel{
		window:         window,
		onTitleChanged: onTitleChanged,
		title:          title,
	}

	panel.serverInfo = newServerInfo(
		title,
		panel.handleIntervalChanged,
	)

	panel.connection = newConnection(panel.connect, panel.disconnect)
	panel.actions = newActions(
		panel.changeServerPassword,
		panel.changeLevel,
		panel.execConfig,
		panel.sendCustomCommand,
	)
	panel.sendMessage = newSendMessage(panel.sendMessageAction)
	panel.scoreboardView = newScoreboardView()
	panel.scoreboardView.SetOnKick(panel.kickPlayer)
	panel.logs = newLogViewer()
	panel.cvars = newCVarPanel()
	panel.actor = scoreboard.NewActor(8)
	panel.logs.SetOnEvent(func(evt logparse.Event) {
		panel.actor.Events() <- evt
	})
	go panel.consumeSnapshots()

	panel.buildUI(title)
	return panel
}

// TabItem returns the tab data for this panel.
func (panel *ServerPanel) TabItem() *container.TabItem {
	return panel.tabItem
}

func (panel *ServerPanel) updateTitle(title string) {
	panel.title = title
	panel.tabItem.Text = title
	panel.serverInfo.SetName(title)
	if panel.onTitleChanged != nil {
		panel.onTitleChanged()
	}
}

func (panel *ServerPanel) handleIntervalChanged() {
	if panel.client == nil {
		return
	}
	panel.startAutoRefresh()
}

func (panel *ServerPanel) consumeSnapshots() {
	for snap := range panel.actor.Snapshots() {
		fyne.Do(func(snap scoreboard.Snapshot) func() {
			return func() { panel.applySnapshot(snap) }
		}(snap))
	}
}

func (panel *ServerPanel) applySnapshot(snap scoreboard.Snapshot) {
	panel.scoreboardView.Update(snap)
	for name, value := range snap.CVars {
		if value == "" {
			continue
		}
		if value == panel.cvars.Value(name) {
			panel.cvars.ClearPending(name)
			continue
		}
		panel.cvars.Set(name, value)
	}
}
