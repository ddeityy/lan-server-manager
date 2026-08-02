package ui

import (
	"lan-server-manager/rcon"
)

func (p *ServerPanel) setConnected(connected bool) {
	if connected {
		p.connection.SetConnected(true)
		p.actions.SetEnabled(true)
		p.players.SetEnabled(true)
		p.pendingMapSync = true
		p.startAutoRefresh()
		return
	}
	p.stopAutoRefresh()
	p.connection.SetConnected(false)
	p.actions.SetEnabled(false)
	p.players.SetEnabled(false)
}

func (p *ServerPanel) resetInfo() {
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

	if info.Hostname != "" {
		p.updateTitle(info.Hostname)
	}

	p.players.Update(info.Players, p.kick)
}
