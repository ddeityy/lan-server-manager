package ui

import (
	"strconv"

	"lan-server-manager/game/logparse"
	"lan-server-manager/logger"
	"lan-server-manager/rcon"
)

func (p *ServerPanel) setConnected(connected bool) {
	if connected {
		logger.Infof("%s: state -> connected", p.title)
		p.connection.SetConnected(true)
		p.actions.SetEnabled(true)
		p.sendMessage.SetEnabled(true)
		p.pendingMapSync = true
		p.startAutoRefresh()
		return
	}
	logger.Infof("%s: state -> disconnected", p.title)
	p.stopAutoRefresh()
	p.connection.SetConnected(false)
	p.actions.SetEnabled(false)
	p.sendMessage.SetEnabled(false)
}

func (p *ServerPanel) resetInfo() {
	logger.Infof("%s: resetting info", p.title)
	p.serverInfo.Reset()
	p.scoreboardView.Reset()
	p.cvars.Reset()
	p.actor.Events() <- logparse.ResetEvent()
}

func (p *ServerPanel) updateInfo(info rcon.ServerInfo, err error) {
	if err != nil {
		return
	}

	p.serverInfo.SetInfo(info)

	if p.pendingMapSync {
		p.pendingMapSync = false
		p.actions.setMapSelection(info.Map)
	}

	if info.Hostname != "" && info.Hostname != p.title {
		logger.Infof("%s: hostname updated to %q", p.title, info.Hostname)
		p.updateTitle(info.Hostname)
	}

	for _, rconPlayer := range info.Players {
		uid := strconv.Itoa(rconPlayer.UserID)
		p.actor.Events() <- logparse.StatusSeedEvent(logparse.Player{
			Name:    rconPlayer.Name,
			UserID:  uid,
			SteamID: rconPlayer.UniqueID,
			Ping:    rconPlayer.Ping,
		})
	}
}
