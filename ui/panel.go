package ui

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"lan-server-manager/game/logparse"
	"lan-server-manager/game/scoreboard"
	"lan-server-manager/rcon"
)

const defaultRefreshInterval = 1

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
	scoreboard     *scoreboard.Scoreboard

	rconMutex      sync.Mutex
	refreshTicker  *time.Ticker
	refreshStop    chan struct{}
	pendingMapSync bool

	tabItem        *container.TabItem
	onTitleChanged func()
}

// NewServerPanel creates a new tab panel with default connection values.
func NewServerPanel(window fyne.Window, title string, onTitleChanged func()) *ServerPanel {
	p := &ServerPanel{
		window:         window,
		onTitleChanged: onTitleChanged,
		title:          title,
	}

	p.serverInfo = newServerInfo(
		title,
		p.handleIntervalChanged,
	)

	p.connection = newConnection(p.connect, p.disconnect)
	p.actions = newActions(
		p.changeServerPassword,
		p.changeLevel,
		p.execConfig,
		p.sendCustomCommand,
	)
	p.sendMessage = newSendMessage(p.sendMessageAction)
	p.scoreboardView = newScoreboardView()
	p.scoreboardView.SetOnKick(p.kickPlayer)
	p.logs = newLogViewer()
	p.scoreboard = scoreboard.New()
	p.logs.SetOnEvent(func(evt logparse.Event) {
		p.scoreboard.Apply(evt)
		fyne.Do(func() {
			red, blu, _, unassigned := p.scoreboard.TeamsAndUnassigned()
			redScore, bluScore := p.scoreboard.Scores()
			p.scoreboardView.Update(red, blu, unassigned, p.scoreboard.TimeSinceStart(), redScore, bluScore)
		})
	})

	p.buildUI(title)
	return p
}

// TabItem returns the tab data for this panel.
func (p *ServerPanel) TabItem() *container.TabItem {
	return p.tabItem
}

func (p *ServerPanel) updateTitle(title string) {
	p.title = title
	p.tabItem.Text = title
	p.serverInfo.SetName(title)
	if p.onTitleChanged != nil {
		p.onTitleChanged()
	}
}

func (p *ServerPanel) handleIntervalChanged() {
	if p.client == nil {
		return
	}
	p.startAutoRefresh()
}
