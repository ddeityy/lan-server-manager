package ui

import (
	"fyne.io/fyne/v2"

	"lan-server-manager/server"
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

func (p *ServerPanel) updateInfo(info server.ServerInfo, err error) {
	if err != nil {
		p.serverInfo.SetStatus("Refresh failed: " + formatError(err))
		return
	}

	password := p.actions.ServerPassword()
	p.serverInfo.SetInfo(info, p.connection.Address(), password)

	if p.pendingMapSync {
		p.pendingMapSync = false
		p.actions.SetMap(info.Map)
		p.notifyChanged()
	}

	if info.Hostname != "" {
		p.updateTitle(info.Hostname)
	}

	p.players.Update(info.Players, p.kick)
	p.serverInfo.SetStatus("Connected")
}

func (p *ServerPanel) copyConnectString() {
	fyne.CurrentApp().Clipboard().SetContent(p.serverInfo.connectLabel.Text)
	p.serverInfo.SetStatus("Connect string copied")
}

func (p *ServerPanel) copySTVString() {
	fyne.CurrentApp().Clipboard().SetContent(p.serverInfo.stvLabel.Text)
	p.serverInfo.SetStatus("STV string copied")
}
