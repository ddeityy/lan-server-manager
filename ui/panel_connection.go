package ui

import (
	"fmt"

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
				p.statusLabel.SetText(fmt.Sprintf("Connection failed: %v", err))
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
		p.resetInfo()
		p.updateTitle("Server")
	})
}

func (p *ServerPanel) refresh() {
	if p.server == nil {
		return
	}

	fyne.Do(func() { p.statusLabel.SetText("Refreshing...") })

	go func() {
		info, err := p.doRefresh()
		fyne.Do(func() { p.updateInfo(info, err) })
	}()
}

func (p *ServerPanel) kick(userid int) {
	if p.server == nil {
		fyne.Do(func() { p.statusLabel.SetText("Not connected") })
		return
	}

	fyne.Do(func() { p.statusLabel.SetText(fmt.Sprintf("Kicking player %d...", userid)) })

	go func() {
		err := p.server.Kick(userid)

		fyne.Do(func() {
			if err != nil {
				p.statusLabel.SetText(fmt.Sprintf("Kick failed: %v", err))
				return
			}
			p.statusLabel.SetText(fmt.Sprintf("Kicked player %d", userid))
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
		fyne.Do(func() { p.statusLabel.SetText("Not connected") })
		return
	}

	fyne.Do(func() {
		p.kickAllButton.Disable()
		p.statusLabel.SetText("Kicking all players...")
	})

	go func() {
		var players []server.Player
		fyne.DoAndWait(func() {
			players = p.lastInfo.Players
		})

		if len(players) == 0 {
			fyne.Do(func() {
				p.statusLabel.SetText("No players to kick")
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
				p.statusLabel.SetText(fmt.Sprintf("Kick all finished with errors: %v", lastErr))
			} else {
				p.statusLabel.SetText("Kicked all players")
			}
			p.refresh()
		})
	}()
}

func (p *ServerPanel) changeLevel() {
	selected := p.mapSelect.Selected
	if selected == "" {
		fyne.Do(func() { p.statusLabel.SetText("Select a map first") })
		return
	}
	if p.server == nil {
		fyne.Do(func() { p.statusLabel.SetText("Not connected") })
		return
	}

	fyne.Do(func() {
		p.changeLevelButton.Disable()
		p.statusLabel.SetText("Changing level to " + selected + "...")
	})

	go func() {
		err := p.server.ChangeLevel(selected)

		fyne.Do(func() {
			p.changeLevelButton.Enable()
			if err != nil {
				p.statusLabel.SetText(fmt.Sprintf("Changelevel failed: %v", err))
				return
			}
			p.statusLabel.SetText("Changed level to " + selected)
		})
	}()
}
