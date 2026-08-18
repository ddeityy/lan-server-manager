package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"lan-server-manager/config"
	"lan-server-manager/logger"
	"lan-server-manager/ui"
)

const (
	defaultWindowWidth  = 1920
	defaultWindowHeight = 1080
)

func main() {
	logger.Infof("Starting LAN TF2 Server Manager")

	application := app.NewWithID("com.lan.server-manager")
	win := application.NewWindow("LAN TF2 Server Manager")
	win.Resize(fyne.Size{Width: defaultWindowWidth, Height: defaultWindowHeight})
	win.CenterOnScreen()

	cfg, err := config.Load("config/config.toml")
	if err != nil {
		logger.Errorf("Failed to load config.toml, using defaults: %v", err)
		cfg = config.Default()
	} else {
		logger.Infof("Loaded config from config/config.toml")
	}

	manager := ui.NewManager(win, cfg)
	win.SetContent(manager.Content())
	logger.Infof("Showing main window")
	win.ShowAndRun()
}
