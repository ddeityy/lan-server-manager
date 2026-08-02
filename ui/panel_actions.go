package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/rcon"
)

// runAction disables button, sets a pending status, executes the RCON action in a
// goroutine, and then re-enables the button and reports the result on the UI thread.
func (p *ServerPanel) runAction(
	button *widget.Button,
	pending, success, failure string,
	action func() error,
	onSuccess func(),
) {
	button.Disable()
	p.actions.SetStatus(pending)

	go func() {
		err := action()

		fyne.Do(func() {
			button.Enable()
			if err != nil {
				p.actions.SetStatus(failure + ": " + formatError(err))
				return
			}
			if onSuccess != nil {
				onSuccess()
			}
			p.actions.SetStatus(success)
		})
	}()
}

// Disconnect closes the RCON connection and resets the panel UI.
func (p *ServerPanel) Disconnect() {
	if p.client != nil {
		p.client.Close()
		p.client = nil
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
		if p.client != nil {
			p.client.Close()
		}
		p.client = rcon.NewClient(address, password)
		err := p.client.Connect()

		fyne.Do(func() {
			if err != nil {
				p.connection.SetStatus("Connection failed: " + formatError(err))
				p.setConnected(false)
				return
			}

			p.updateTitle(address)
			p.setConnected(true)
			p.refresh()

			if p.actions.ServerPassword() != "" {
				p.changeServerPassword()
			}
		})
	}()
}

func (p *ServerPanel) disconnect() {
	if p.client != nil {
		p.client.Close()
		p.client = nil
	}

	fyne.Do(func() {
		p.setConnected(false)
		p.connection.SetStatus("Disconnected")
		p.resetInfo()
		p.updateTitle("Server")
	})
}

func (p *ServerPanel) refresh() {
	if p.client == nil {
		return
	}

	p.serverInfo.SetRefreshing()

	go func() {
		info, err := p.doRefresh()
		fyne.Do(func() { p.updateInfo(info, err) })
	}()
}

func (p *ServerPanel) kick(userid int) {
	if p.client == nil {
		p.players.SetStatus("Not connected")
		return
	}

	p.players.SetStatus(fmt.Sprintf("Kicking player %d...", userid))

	go func() {
		err := p.client.Kick(userid)

		fyne.Do(func() {
			if err != nil {
				p.players.SetStatus("Kick failed: " + formatError(err))
				return
			}
			p.players.SetStatus(fmt.Sprintf("Kicked player %d", userid))
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
	if p.client == nil {
		p.players.SetStatus("Not connected")
		return
	}

	p.players.kickAllButton.Disable()
	p.players.SetStatus("Kicking all players...")

	go func() {
		players := p.serverInfo.LastInfo().Players

		if len(players) == 0 {
			fyne.Do(func() {
				p.players.SetStatus("No players to kick")
				p.players.kickAllButton.Enable()
			})
			return
		}

		var lastErr error
		for _, player := range players {
			if err := p.client.Kick(player.UserID); err != nil {
				lastErr = err
			}
		}

		fyne.Do(func() {
			if lastErr != nil {
				p.players.SetStatus("Kick all finished with errors: " + formatError(lastErr))
			} else {
				p.players.SetStatus("Kicked all players")
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
	if p.client == nil {
		p.actions.SetStatus("Not connected")
		return
	}

	p.runAction(
		p.actions.changeLevelButton,
		"Changing level to "+mapName+"...",
		"Changed level to "+mapName,
		"Changelevel failed",
		func() error { return p.client.ChangeLevel(mapName) },
		nil,
	)
}

func (p *ServerPanel) changeServerPassword() {
	password := p.actions.ServerPassword()
	if strings.TrimSpace(password) == "" {
		p.actions.SetStatus("Enter a password first")
		return
	}
	if p.client == nil {
		p.actions.SetStatus("Not connected")
		return
	}

	p.runAction(
		p.actions.changePasswordButton,
		"Setting server password...",
		"Server password updated",
		"Set password failed",
		func() error { return p.client.SetPassword(password) },
		func() { p.serverInfo.UpdateConnectStrings(password) },
	)
}

func (p *ServerPanel) execConfig() {
	configName := p.actions.SelectedConfig()
	if configName == "" {
		p.actions.SetStatus("Select a config first")
		return
	}
	if p.client == nil {
		p.actions.SetStatus("Not connected")
		return
	}

	p.runAction(
		p.actions.execConfigButton,
		"Executing config "+configName+"...",
		"Executed config "+configName,
		"Exec config failed",
		func() error { return p.client.ExecConfig(configName) },
		nil,
	)
}

func (p *ServerPanel) sendMessage() {
	msg := strings.TrimSpace(p.actions.Message())
	if msg == "" {
		p.actions.SetStatus("Enter a message first")
		return
	}
	if p.client == nil {
		p.actions.SetStatus("Not connected")
		return
	}

	p.runAction(
		p.actions.sendMessageButton,
		"Sending message...",
		"Sent message",
		"Send message failed",
		func() error { return p.client.Execute("say " + msg) },
		nil,
	)
}

func (p *ServerPanel) sendCustomCommand() {
	cmd := strings.TrimSpace(p.actions.CustomCommand())
	if cmd == "" {
		p.actions.SetStatus("Enter a command first")
		return
	}
	if p.client == nil {
		p.actions.SetStatus("Not connected")
		return
	}

	p.runAction(
		p.actions.customCommandButton,
		"Sending: "+cmd,
		"Sent: "+cmd,
		"Command failed",
		func() error { return p.client.Execute(cmd) },
		nil,
	)
}
