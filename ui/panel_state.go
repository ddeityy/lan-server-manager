package ui

import (
	"strconv"

	"lan-server-manager/game/logparse"
	"lan-server-manager/game/scoreboard"
	"lan-server-manager/internal/logger"
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
	p.scoreboard.Reset()
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

	for _, rp := range info.Players {
		uid := strconv.Itoa(rp.UserID)
		key := scoreboard.PlayerKey(uid, rp.UniqueID)
		if _, ok := p.scoreboard.Player(key); !ok {
			p.scoreboard.Upsert(logparse.Player{
				Name:    rp.Name,
				UserID:  uid,
				SteamID: rp.UniqueID,
			})
		}
		p.scoreboard.SetPing(key, rp.Ping)
	}

	red, blu, _, unassigned := p.scoreboard.TeamsAndUnassigned()
	redScore, bluScore := p.scoreboard.Scores()
	p.scoreboardView.Update(red, blu, unassigned, p.scoreboard.TimeSinceStart(), redScore, bluScore)
}
