package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"

	"lan-server-manager/internal/logger"
	"lan-server-manager/rcon"
)

func (p *ServerPanel) startAutoRefresh() {
	if p.client == nil {
		return
	}
	p.stopAutoRefresh()

	interval := time.Duration(p.serverInfo.RefreshInterval()) * time.Second
	p.refreshTicker = time.NewTicker(interval)
	p.refreshStop = make(chan struct{})
	logger.Infof("%s: auto refresh started (interval %s)", p.title, interval)

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
		logger.Infof("%s: auto refresh stopped", p.title)
	}
	if p.refreshStop != nil {
		close(p.refreshStop)
		p.refreshStop = nil
	}
}

// doRefresh performs a synchronous status refresh. It serializes access to the
// server so manual and automatic refreshes cannot overlap.
func (p *ServerPanel) doRefresh() (rcon.ServerInfo, error) {
	p.refreshMutex.Lock()
	defer p.refreshMutex.Unlock()

	if p.client == nil {
		return rcon.ServerInfo{}, fmt.Errorf("not connected")
	}
	if err := p.client.Refresh(); err != nil {
		logger.Errorf("%s: refresh failed: %v", p.title, err)
		return rcon.ServerInfo{}, err
	}
	return p.client.Info(), nil
}
