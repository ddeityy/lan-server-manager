package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/internal/logger"
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
	logger.Infof("%s: %s", p.title, pending)

	go func() {
		p.rconMutex.Lock()
		err := action()
		p.rconMutex.Unlock()

		fyne.Do(func() {
			button.Enable()
			if err != nil {
				logger.Errorf("%s: %s: %v", p.title, failure, err)
				p.actions.SetStatus(failure + ": " + p.formatError(err))
				return
			}
			if onSuccess != nil {
				onSuccess()
			}
			logger.Infof("%s: %s", p.title, success)
			p.actions.SetStatus(success)
		})
	}()
}

func (p *ServerPanel) closeClient() {
	if p.client == nil {
		return
	}
	if err := p.client.Close(); err != nil {
		logger.Warnf("%s: close connection failed: %v", p.title, err)
	}
	p.client = nil
}

// Disconnect closes the RCON connection and resets the panel UI.
func (p *ServerPanel) Disconnect() {
	logger.Infof("%s: disconnecting", p.title)
	p.closeClient()
	p.logs.Stop()
	p.setConnected(false)
	p.connection.SetStatus("Disconnected")
	p.resetInfo()
}

func (p *ServerPanel) connect() {
	address := p.connection.Address()
	password := p.connection.Password()
	if address == "" {
		p.connection.SetStatus("Address is required")
		logger.Warnf("%s: connect attempted with empty address", p.title)
		return
	}

	p.connection.SetConnecting()
	logger.Infof("%s: connecting to %s", p.title, address)

	go func() {
		p.closeClient()
		p.client = rcon.NewClient(address, password)
		err := p.client.Connect()

		fyne.Do(func() {
			if err != nil {
				logger.Errorf("%s: connection to %s failed: %v", p.title, address, err)
				p.connection.SetStatus("Connection failed: " + p.formatError(err))
				p.setConnected(false)
				return
			}

			logger.Infof("%s: connected to %s", p.title, address)
			p.updateTitle(address)
			p.setConnected(true)
			p.refresh()
			p.startLogTail()

			if p.actions.serverPasswordEntry.Text != "" {
				p.changeServerPassword()
			}
		})
	}()
}

func (p *ServerPanel) disconnect() {
	logger.Infof("%s: disconnecting", p.title)
	p.closeClient()
	p.logs.Stop()

	fyne.Do(func() {
		p.setConnected(false)
		p.connection.SetStatus("Disconnected")
		p.resetInfo()
		p.updateTitle("Server")
	})
}

func (p *ServerPanel) startLogTail() {
	logger.Infof("%s: auto-starting log tail", p.title)
	p.logs.Stop()
	target := p.logs.Target()
	target.ContainerName = strings.TrimSpace(p.connection.ContainerName())
	target.SSHHost = strings.TrimSpace(p.connection.SSHHost())
	target.SSHPassword = p.connection.SSHPassword()
	if target.SSHUser == "" {
		target.SSHUser = "root"
	}
	p.logs.SetTarget(target)
	if err := p.logs.start(); err != nil {
		logger.Errorf("%s: failed to start log tail: %v", p.title, err)
	}
}

func (p *ServerPanel) refresh() {
	if p.client == nil {
		return
	}

	logger.Infof("%s: refreshing status", p.title)
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
	logger.Infof("%s: kicking player %d", p.title, userid)

	go func() {
		p.rconMutex.Lock()
		err := p.client.Kick(userid)
		p.rconMutex.Unlock()

		fyne.Do(func() {
			if err != nil {
				logger.Errorf("%s: kick %d failed: %v", p.title, userid, err)
				p.players.SetStatus("Kick failed: " + p.formatError(err))
				return
			}
			logger.Infof("%s: kicked player %d", p.title, userid)
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
	logger.Infof("%s: kicking all players", p.title)

	go func() {
		players := p.serverInfo.LastInfo().Players

		if len(players) == 0 {
			fyne.Do(func() {
				p.players.SetStatus("No players to kick")
				p.players.kickAllButton.Enable()
			})
			return
		}

		p.rconMutex.Lock()
		var lastErr error
		for _, player := range players {
			if err := p.client.Kick(player.UserID); err != nil {
				lastErr = err
			}
		}
		p.rconMutex.Unlock()

		fyne.Do(func() {
			if lastErr != nil {
				logger.Errorf("%s: kick all finished with errors: %v", p.title, lastErr)
				p.players.SetStatus("Kick all finished with errors: " + p.formatError(lastErr))
			} else {
				logger.Infof("%s: kicked all players", p.title)
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
	logger.Infof("%s: changing level to %s", p.title, mapName)

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
	password := p.actions.serverPasswordEntry.Text
	if strings.TrimSpace(password) == "" {
		p.actions.SetStatus("Enter a password first")
		return
	}
	if p.client == nil {
		p.actions.SetStatus("Not connected")
		return
	}
	logger.Infof("%s: setting server password", p.title)

	p.runAction(
		p.actions.changePasswordButton,
		"Setting server password...",
		"Server password updated",
		"Set password failed",
		func() error { return p.client.SetPassword(password) },
		nil,
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
	logger.Infof("%s: executing config %s", p.title, configName)

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
	msg := strings.TrimSpace(p.actions.messageEntry.Text)
	if msg == "" {
		p.actions.SetStatus("Enter a message first")
		return
	}
	if p.client == nil {
		p.actions.SetStatus("Not connected")
		return
	}
	logger.Infof("%s: sending message", p.title)

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
	cmd := strings.TrimSpace(p.actions.customCommandEntry.Text)
	if cmd == "" {
		p.actions.SetStatus("Enter a command first")
		return
	}
	if p.client == nil {
		p.actions.SetStatus("Not connected")
		return
	}
	logger.Infof("%s: sending custom command: %s", p.title, cmd)

	p.runAction(
		p.actions.customCommandButton,
		"Sending: "+cmd,
		"Sent: "+cmd,
		"Command failed",
		func() error { return p.client.Execute(cmd) },
		nil,
	)
}
