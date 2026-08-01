package ui

import (
	"fmt"
	"net"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/server"
)

func (p *ServerPanel) setConnected(connected bool) {
	if connected {
		p.connectButton.Disable()
		p.disconnectButton.Enable()
		p.refreshButton.Enable()
		p.changeLevelButton.Enable()
		if p.autoRefreshCheck.Checked {
			p.refreshButton.Disable()
		}
		p.changePasswordButton.Enable()
		p.execConfigButton.Enable()
		p.customCommandButton.Enable()
		p.kickAllButton.Enable()
		p.statusLabel.SetText("Connected")
		p.pendingMapSync = true
		if p.autoRefreshCheck.Checked {
			p.startAutoRefresh()
		}
		return
	}
	p.stopAutoRefresh()
	p.connectButton.Enable()
	p.disconnectButton.Disable()
	p.refreshButton.Disable()
	p.changeLevelButton.Disable()
	p.changePasswordButton.Disable()
	p.execConfigButton.Disable()
	p.customCommandButton.Disable()
	p.kickAllButton.Disable()
}

func (p *ServerPanel) resetInfo() {
	p.lastInfo = server.ServerInfo{}

	p.addressLabel.SetText("")
	p.mapLabel.SetText("")
	p.playersLabel.SetText("")
	p.sourceTVLabel.SetText("")
	p.connectLabel.SetText("")
	p.stvLabel.SetText("")
	p.serverInfoStatusLabel.SetText("")
	p.copyConnectButton.Disable()
	p.copySTVButton.Disable()

	p.playerList.Length = func() int { return 0 }
	p.playerList.UpdateItem = func(_ widget.ListItemID, _ fyne.CanvasObject) {}
	p.playerList.Refresh()

	p.playersAccordion.Items[0].Title = "Players"
	p.playersAccordion.Refresh()
}

func (p *ServerPanel) updateInfo(info server.ServerInfo, err error) {
	if err != nil {
		p.serverInfoStatusLabel.SetText("Refresh failed: " + formatError(err))
		return
	}

	p.lastInfo = info

	p.addressLabel.SetText(formatAddress(info.Address, info.ConfiguredAddress))
	if info.SourceTV.Address != "" {
		tvText := fmt.Sprintf("%s, delay %s", info.SourceTV.Address, info.SourceTV.Delay)
		if isAddressUsable(info.SourceTV.Local) {
			tvText += "\nlocal " + info.SourceTV.Local
		}
		p.sourceTVLabel.SetText(tvText)
	} else {
		p.sourceTVLabel.SetText("")
	}
	p.mapLabel.SetText(info.Map)
	p.playersLabel.SetText(fmt.Sprintf("%d / %d", info.HumanPlayers, info.MaxPlayers))

	p.updateConnectStrings()

	if p.pendingMapSync {
		p.pendingMapSync = false
		setMapSelection(p.mapSelect, info.Map)
		p.notifyChanged()
	}

	if info.Hostname != "" {
		p.updateTitle(info.Hostname)
	}

	updatePlayerList(p.playerList, info.Players, p.kick)

	p.playersAccordion.Items[0].Title = fmt.Sprintf("Players (%d)", len(info.Players))
	p.playersAccordion.Refresh()

	p.kickAllButton.Disable()
	if len(info.Players) > 0 {
		p.kickAllButton.Enable()
	}

	p.serverInfoStatusLabel.SetText("Connected")
}

// updateConnectStrings rebuilds the connect and stv copy strings from the
// current server info and password entry.
func (p *ServerPanel) updateConnectStrings() {
	password := p.serverPasswordEntry.Text

	gameAddr := p.lastInfo.GameConnectAddress()
	if gameAddr == "" {
		p.connectLabel.SetText("")
		p.copyConnectButton.Disable()
	} else {
		connect := fmt.Sprintf("connect %s", gameAddr)
		if password != "" {
			connect += fmt.Sprintf("; password %s", password)
		}
		p.connectLabel.SetText(connect)
		p.copyConnectButton.Enable()
	}

	stvAddr := p.lastInfo.STVConnectAddress()
	if stvAddr == "" {
		p.stvLabel.SetText("")
		p.copySTVButton.Disable()
		return
	}

	stv := fmt.Sprintf("connect %s", stvAddr)
	if password != "" {
		stv += fmt.Sprintf("; password %s", password)
	}
	p.stvLabel.SetText(stv)
	p.copySTVButton.Enable()
}

// formatAddress returns a multi-line summary of the server's addresses for the
// Address info tile, omitting empty or unknown placeholders.
func formatAddress(a server.Address, configured string) string {
	parts := []string{}
	if isAddressUsable(a.SDR) {
		parts = append(parts, fmt.Sprintf("SDR: %s", a.SDR))
	}
	if isAddressUsable(configured) && configured != a.SDR {
		parts = append(parts, fmt.Sprintf("Local: %s", configured))
	}
	if isAddressUsable(a.Public) {
		parts = append(parts, fmt.Sprintf("Public: %s", a.Public))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, "\n")
}

// isAddressUsable reports whether an address string is a real endpoint and not
// an empty or unknown placeholder like "?.?.?.?:?" or "0.0.0.0:27015".
func isAddressUsable(s string) bool {
	if s == "" {
		return false
	}
	host, _, err := net.SplitHostPort(s)
	if err != nil {
		host = s
	}
	return host != "0.0.0.0" && !strings.Contains(host, "?")
}

func (p *ServerPanel) copyConnectString() {
	fyne.CurrentApp().Clipboard().SetContent(p.connectLabel.Text)
	p.serverInfoStatusLabel.SetText("Connect string copied")
}

func (p *ServerPanel) copySTVString() {
	fyne.CurrentApp().Clipboard().SetContent(p.stvLabel.Text)
	p.serverInfoStatusLabel.SetText("STV string copied")
}
