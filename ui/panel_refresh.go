package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"

	"lan-server-manager/server"
)

func (p *ServerPanel) startAutoRefresh() {
	if p.server == nil || !p.serverInfo.AutoRefreshEnabled() {
		return
	}
	p.stopAutoRefresh()

	interval := time.Duration(p.serverInfo.RefreshInterval()) * time.Second
	p.refreshTicker = time.NewTicker(interval)
	p.refreshStop = make(chan struct{})

	go func() {
		for {
			select {
			case <-p.refreshStop:
				return
			case <-p.refreshTicker.C:
				info, err := p.doRefresh()
				fyne.Do(func() { p.updateInfo(info, err) })
			}
		}
	}()
}

func (p *ServerPanel) stopAutoRefresh() {
	if p.refreshTicker != nil {
		p.refreshTicker.Stop()
		p.refreshTicker = nil
	}
	if p.refreshStop != nil {
		close(p.refreshStop)
		p.refreshStop = nil
	}
}

// doRefresh performs a synchronous status refresh. It serializes access to the
// server so manual and automatic refreshes cannot overlap.
func (p *ServerPanel) doRefresh() (server.ServerInfo, error) {
	p.refreshMutex.Lock()
	defer p.refreshMutex.Unlock()

	if p.server == nil {
		return server.ServerInfo{}, fmt.Errorf("not connected")
	}
	if err := p.server.Refresh(); err != nil {
		return server.ServerInfo{}, err
	}
	return p.server.Info(), nil
}
