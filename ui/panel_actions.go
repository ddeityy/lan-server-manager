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
	p.statusLabel.SetText("Disconnected")
	p.resetInfo()
}

func (p *ServerPanel) connect() {
	address := p.addressEntry.Text
	password := p.passwordEntry.Text
	if address == "" {
		p.statusLabel.SetText("Address is required")
		return
	}

	fyne.Do(func() {
		p.connectButton.Disable()
		p.statusLabel.SetText("Connecting...")
	})

	go func() {
		if p.server != nil {
			p.server.Close()
		}
		p.server = server.NewServer(address, password)
		err := p.server.Connect()

		fyne.Do(func() {
			if err != nil {
				p.statusLabel.SetText("Connection failed: " + formatError(err))
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
		p.statusLabel.SetText("Disconnected")
		p.resetInfo()
		p.updateTitle("Server")
	})
}

func (p *ServerPanel) refresh() {
	if p.server == nil {
		return
	}

	fyne.Do(func() { p.serverInfoStatusLabel.SetText("Refreshing...") })

	go func() {
		info, err := p.doRefresh()
		fyne.Do(func() { p.updateInfo(info, err) })
	}()
}

func (p *ServerPanel) kick(userid int) {
	if p.server == nil {
		fyne.Do(func() { p.actionsStatusLabel.SetText("Not connected") })
		return
	}

	fyne.Do(func() { p.actionsStatusLabel.SetText(fmt.Sprintf("Kicking player %d...", userid)) })

	go func() {
		err := p.server.Kick(userid)

		fyne.Do(func() {
			if err != nil {
				p.actionsStatusLabel.SetText("Kick failed: " + formatError(err))
				return
			}
			p.actionsStatusLabel.SetText(fmt.Sprintf("Kicked player %d", userid))
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
		fyne.Do(func() { p.actionsStatusLabel.SetText("Not connected") })
		return
	}

	fyne.Do(func() {
		p.kickAllButton.Disable()
		p.actionsStatusLabel.SetText("Kicking all players...")
	})

	go func() {
		var players []server.Player
		fyne.DoAndWait(func() {
			players = p.lastInfo.Players
		})

		if len(players) == 0 {
			fyne.Do(func() {
				p.actionsStatusLabel.SetText("No players to kick")
				p.kickAllButton.Enable()
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
				p.actionsStatusLabel.SetText("Kick all finished with errors: " + formatError(lastErr))
			} else {
				p.actionsStatusLabel.SetText("Kicked all players")
			}
			p.refresh()
		})
	}()
}

func (p *ServerPanel) changeLevel() {
	mapName := p.mapSelect.Selected
	if mapName == "" {
		fyne.Do(func() { p.actionsStatusLabel.SetText("Select a map first") })
		return
	}
	if p.server == nil {
		fyne.Do(func() { p.actionsStatusLabel.SetText("Not connected") })
		return
	}

	fyne.Do(func() {
		p.changeLevelButton.Disable()
		p.actionsStatusLabel.SetText("Changing level to " + mapName + "...")
	})

	go func() {
		err := p.server.ChangeLevel(mapName)

		fyne.Do(func() {
			p.changeLevelButton.Enable()
			if err != nil {
				p.actionsStatusLabel.SetText("Changelevel failed: " + formatError(err))
				return
			}
			p.actionsStatusLabel.SetText("Changed level to " + mapName)
		})
	}()
}

func (p *ServerPanel) changeServerPassword() {
	password := p.serverPasswordEntry.Text
	if strings.TrimSpace(password) == "" {
		fyne.Do(func() { p.actionsStatusLabel.SetText("Enter a password first") })
		return
	}
	if p.server == nil {
		fyne.Do(func() { p.actionsStatusLabel.SetText("Not connected") })
		return
	}

	fyne.Do(func() {
		p.changePasswordButton.Disable()
		p.actionsStatusLabel.SetText("Setting server password...")
	})

	go func() {
		err := p.server.SetPassword(password)

		fyne.Do(func() {
			p.changePasswordButton.Enable()
			if err != nil {
				p.actionsStatusLabel.SetText("Set password failed: " + formatError(err))
				return
			}
			p.updateConnectStrings()
			p.actionsStatusLabel.SetText("Server password updated")
		})
	}()
}

func (p *ServerPanel) execConfig() {
	configName := p.configSelect.Selected
	if configName == "" {
		fyne.Do(func() { p.actionsStatusLabel.SetText("Select a config first") })
		return
	}
	if p.server == nil {
		fyne.Do(func() { p.actionsStatusLabel.SetText("Not connected") })
		return
	}

	fyne.Do(func() {
		p.execConfigButton.Disable()
		p.actionsStatusLabel.SetText("Executing config " + configName + "...")
	})

	go func() {
		err := p.server.ExecConfig(configName)

		fyne.Do(func() {
			p.execConfigButton.Enable()
			if err != nil {
				p.actionsStatusLabel.SetText("Exec config failed: " + formatError(err))
				return
			}
			p.actionsStatusLabel.SetText("Executed config " + configName)
		})
	}()
}

func (p *ServerPanel) sendCustomCommand() {
	cmd := strings.TrimSpace(p.customCommandEntry.Text)
	if cmd == "" {
		fyne.Do(func() { p.actionsStatusLabel.SetText("Enter a command first") })
		return
	}
	if p.server == nil {
		fyne.Do(func() { p.actionsStatusLabel.SetText("Not connected") })
		return
	}

	fyne.Do(func() {
		p.customCommandButton.Disable()
		p.actionsStatusLabel.SetText("Sending: " + cmd)
	})

	go func() {
		err := p.server.Execute(cmd)

		fyne.Do(func() {
			p.customCommandButton.Enable()
			if err != nil {
				p.actionsStatusLabel.SetText("Command failed: " + formatError(err))
				return
			}
			p.actionsStatusLabel.SetText("Sent: " + cmd)
		})
	}()
}
