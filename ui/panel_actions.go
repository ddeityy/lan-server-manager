package ui

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"lan-server-manager/game/logparse"
	"lan-server-manager/game/scoreboard"
	"lan-server-manager/logger"
	"lan-server-manager/rcon"
)

// runAction disables button, sets a pending status, executes the RCON action in a
// goroutine, and then re-enables the button and reports the result on the UI thread.
func (p *ServerPanel) runAction(
	button *widget.Button,
	pending, success, failure string,
	action func() error,
	onSuccess, onFailure func(),
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
				if onFailure != nil {
					onFailure()
				}
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
			go func() {
				p.rconMutex.Lock()
				defer p.rconMutex.Unlock()
				p.queryTrackedCVars()
			}()

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

func (p *ServerPanel) queryTrackedCVars() {
	if p.client == nil {
		return
	}
	for _, name := range TrackedCVarNames() {
		resp, err := p.client.ExecuteWithResponse(name)
		if err != nil {
			logger.Warnf("%s: query %s failed: %v", p.title, name, err)
			continue
		}
		for _, line := range strings.Split(resp, "\n") {
			line = strings.TrimSpace(line)
			cvarName, value, ok := logparse.ParseCVar(line)
			if !ok {
				continue
			}
			logger.Infof("%s: queried %s = %s", p.title, cvarName, value)
			p.applyCVar(cvarName, value)
		}
	}
}

func (p *ServerPanel) applyCVar(name, value string) {
	p.scoreboard.Apply(logparse.Event{
		Type: logparse.EventCVar,
		Data: map[string]string{"cvar": name, "value": value},
	})
	fyne.Do(func() { p.scoreboardView.SetCVar(name, value) })
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
		p.actions.ClearServerPassword,
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

	pendingCVars := TrackedCVarNames()
	p.scoreboardView.MarkCVarsPending(pendingCVars...)

	p.runAction(
		p.actions.execConfigButton,
		"Executing config "+configName+"...",
		"Executed config "+configName,
		"Exec config failed",
		func() error { return p.client.ExecConfig(configName) },
		func() {
			go func() {
				p.rconMutex.Lock()
				defer p.rconMutex.Unlock()
				p.queryTrackedCVars()
			}()
		},
		func() { p.scoreboardView.ClearCVarsPending(pendingCVars...) },
	)
}

func (p *ServerPanel) sendMessageAction() {
	msg := strings.TrimSpace(p.sendMessage.Text())
	if msg == "" {
		p.sendMessage.SetEnabled(true)
		return
	}
	if p.client == nil {
		p.actions.SetStatus("Not connected")
		return
	}
	logger.Infof("%s: sending message", p.title)

	p.runAction(
		p.sendMessage.button,
		"Sending message...",
		"Sent message",
		"Send message failed",
		func() error { return p.client.Execute("say " + msg) },
		p.sendMessage.Clear,
		nil,
	)
}

func (p *ServerPanel) kickPlayer(player scoreboard.PlayerStats) {
	if p.client == nil {
		p.actions.SetStatus("Not connected")
		return
	}
	uid, err := strconv.Atoi(player.UserID)
	if err != nil || uid <= 0 {
		p.actions.SetStatus("Cannot kick: missing user ID")
		return
	}

	logger.Infof("%s: kicking player %d (%s)", p.title, uid, player.Name)
	go func() {
		p.rconMutex.Lock()
		err := p.client.Kick(uid)
		p.rconMutex.Unlock()
		fyne.Do(func() {
			if err != nil {
				logger.Errorf("%s: kick failed: %v", p.title, err)
				p.actions.SetStatus("Kick failed: " + p.formatError(err))
				return
			}
			logger.Infof("%s: kicked player %d", p.title, uid)
			p.actions.SetStatus("Kicked " + player.Name)
		})
	}()
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

	pendingCVars := trackedCVarNamesFromCommand(cmd)
	if len(pendingCVars) > 0 {
		p.scoreboardView.MarkCVarsPending(pendingCVars...)
	}

	p.runAction(
		p.actions.customCommandButton,
		"Sending: "+cmd,
		"Sent: "+cmd,
		"Command failed",
		func() error { return p.client.Execute(cmd) },
		p.actions.ClearCustomCommand,
		func() { p.scoreboardView.ClearCVarsPending(pendingCVars...) },
	)
}
