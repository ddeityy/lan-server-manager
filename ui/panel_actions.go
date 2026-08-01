package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"lan-server-manager/server"
)

// Disconnect closes the RCON connection and resets the panel UI.
func (p *ServerPanel) Disconnect() {
	if p.server != nil {
		p.server.Close()
		p.server = nil
	}
	p.setConnected(false)
	p.connection.SetStatus("Disconnected")
	p.resetInfo()
}

func (p *ServerPanel) connect() {
	address := p.connection.Address()
	password := p.connection.Password()
	if address == "" {
		p.connection.SetStatus("Address is required")
		return
	}

	p.connection.SetConnecting()

	go func() {
		if p.server != nil {
			p.server.Close()
		}
		p.server = server.NewServer(address, password)
		err := p.server.Connect()

		fyne.Do(func() {
			if err != nil {
				p.connection.SetStatus("Connection failed: " + formatError(err))
				p.setConnected(false)
				return
			}

			p.updateTitle(address)
			p.setConnected(true)
			p.refresh()
		})
	}()
}

func (p *ServerPanel) disconnect() {
	if p.server != nil {
		p.server.Close()
		p.server = nil
	}

	fyne.Do(func() {
		p.setConnected(false)
		p.connection.SetStatus("Disconnected")
		p.resetInfo()
		p.updateTitle("Server")
	})
}

func (p *ServerPanel) refresh() {
	if p.server == nil {
		return
	}

	p.serverInfo.SetRefreshing()

	go func() {
		info, err := p.doRefresh()
		fyne.Do(func() { p.updateInfo(info, err) })
	}()
}

func (p *ServerPanel) kick(userid int) {
	if p.server == nil {
		p.actions.SetStatus("Not connected")
		return
	}

	p.actions.SetStatus(fmt.Sprintf("Kicking player %d...", userid))

	go func() {
		err := p.server.Kick(userid)

		fyne.Do(func() {
			if err != nil {
				p.actions.SetStatus("Kick failed: " + formatError(err))
				return
			}
			p.actions.SetStatus(fmt.Sprintf("Kicked player %d", userid))
			p.refresh()
		})
	}()
}

func (p *ServerPanel) confirmKickAll() {
	dialog.NewConfirm(
		"Kick all players",
		"Are you sure you want to kick every connected player?",
		func(confirmed bool) {
			if !confirmed {
				return
			}
			p.kickAll()
		},
		p.window,
	).Show()
}

func (p *ServerPanel) kickAll() {
	if p.server == nil {
		p.actions.SetStatus("Not connected")
		return
	}

	p.players.kickAllButton.Disable()
	p.actions.SetStatus("Kicking all players...")

	go func() {
		players := p.serverInfo.LastInfo().Players

		if len(players) == 0 {
			fyne.Do(func() {
				p.actions.SetStatus("No players to kick")
				p.players.kickAllButton.Enable()
			})
			return
		}

		var lastErr error
		for _, player := range players {
			if err := p.server.Kick(player.UserID); err != nil {
				lastErr = err
			}
		}

		fyne.Do(func() {
			if lastErr != nil {
				p.actions.SetStatus("Kick all finished with errors: " + formatError(lastErr))
			} else {
				p.actions.SetStatus("Kicked all players")
			}
			p.refresh()
		})
	}()
}

func (p *ServerPanel) changeLevel() {
	mapName := p.actions.SelectedMap()
	if mapName == "" {
		p.actions.SetStatus("Select a map first")
		return
	}
	if p.server == nil {
		p.actions.SetStatus("Not connected")
		return
	}

	p.actions.changeLevelButton.Disable()
	p.actions.SetStatus("Changing level to " + mapName + "...")

	go func() {
		err := p.server.ChangeLevel(mapName)

		fyne.Do(func() {
			p.actions.changeLevelButton.Enable()
			if err != nil {
				p.actions.SetStatus("Changelevel failed: " + formatError(err))
				return
			}
			p.actions.SetStatus("Changed level to " + mapName)
		})
	}()
}

func (p *ServerPanel) changeServerPassword() {
	password := p.actions.ServerPassword()
	if strings.TrimSpace(password) == "" {
		p.actions.SetStatus("Enter a password first")
		return
	}
	if p.server == nil {
		p.actions.SetStatus("Not connected")
		return
	}

	p.actions.changePasswordButton.Disable()
	p.actions.SetStatus("Setting server password...")

	go func() {
		err := p.server.SetPassword(password)

		fyne.Do(func() {
			p.actions.changePasswordButton.Enable()
			if err != nil {
				p.actions.SetStatus("Set password failed: " + formatError(err))
				return
			}
			p.serverInfo.UpdateConnectStrings(password)
			p.actions.SetStatus("Server password updated")
		})
	}()
}

func (p *ServerPanel) execConfig() {
	configName := p.actions.SelectedConfig()
	if configName == "" {
		p.actions.SetStatus("Select a config first")
		return
	}
	if p.server == nil {
		p.actions.SetStatus("Not connected")
		return
	}

	p.actions.execConfigButton.Disable()
	p.actions.SetStatus("Executing config " + configName + "...")

	go func() {
		err := p.server.ExecConfig(configName)

		fyne.Do(func() {
			p.actions.execConfigButton.Enable()
			if err != nil {
				p.actions.SetStatus("Exec config failed: " + formatError(err))
				return
			}
			p.actions.SetStatus("Executed config " + configName)
		})
	}()
}

func (p *ServerPanel) sendCustomCommand() {
	cmd := strings.TrimSpace(p.actions.CustomCommand())
	if cmd == "" {
		p.actions.SetStatus("Enter a command first")
		return
	}
	if p.server == nil {
		p.actions.SetStatus("Not connected")
		return
	}

	p.actions.customCommandButton.Disable()
	p.actions.SetStatus("Sending: " + cmd)

	go func() {
		err := p.server.Execute(cmd)

		fyne.Do(func() {
			p.actions.customCommandButton.Enable()
			if err != nil {
				p.actions.SetStatus("Command failed: " + formatError(err))
				return
			}
			p.actions.SetStatus("Sent: " + cmd)
		})
	}()
}
