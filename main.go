package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"lan-server-manager/config"
	"lan-server-manager/logger"
	"lan-server-manager/ui"
)

func main() {
	logger.Infof("Starting LAN TF2 Server Manager")

	a := app.NewWithID("com.lan.server-manager")
	w := a.NewWindow("LAN TF2 Server Manager")
	w.Resize(fyne.Size{Width: 1920, Height: 1080})
	w.CenterOnScreen()

	cfg, err := config.Load("config/config.toml")
	if err != nil {
		logger.Errorf("Failed to load config.toml, using defaults: %v", err)
		cfg = config.Default()
	} else {
		logger.Infof("Loaded config from config/config.toml")
	}

	manager := ui.NewManager(w, cfg)
	w.SetContent(manager.Content())
	logger.Infof("Showing main window")
	w.ShowAndRun()
}
