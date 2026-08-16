package ui

import (
	"lan-server-manager/internal/logger"
	"lan-server-manager/rcon"
)

func (p *ServerPanel) setConnected(connected bool) {
	if connected {
		logger.Infof("%s: state -> connected", p.title)
		p.connection.SetConnected(true)
		p.actions.SetEnabled(true)
		p.players.SetEnabled(true)
		p.pendingMapSync = true
		p.startAutoRefresh()
		return
	}
	logger.Infof("%s: state -> disconnected", p.title)
	p.stopAutoRefresh()
	p.connection.SetConnected(false)
	p.actions.SetEnabled(false)
	p.players.SetEnabled(false)
}

func (p *ServerPanel) resetInfo() {
	logger.Infof("%s: resetting info", p.title)
	p.serverInfo.Reset()
	p.players.Reset()
}

func (p *ServerPanel) updateInfo(info rcon.ServerInfo, err error) {
	if err != nil {
		return
	}

	p.serverInfo.SetInfo(info)

	if p.pendingMapSync {
		p.pendingMapSync = false
		p.actions.SetMap(info.Map)
	}

	if info.Hostname != "" && info.Hostname != p.title {
		logger.Infof("%s: hostname updated to %q", p.title, info.Hostname)
		p.updateTitle(info.Hostname)
	}

	// logger.Infof("%s: refreshed %d players on %s", p.title, len(info.Players), info.Map)
	p.players.Update(info.Players, p.kick)
}
