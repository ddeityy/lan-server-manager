package main

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"lan-server-manager/config"
	"lan-server-manager/ui"
)

func main() {
	a := app.NewWithID("com.lan.server-manager")
	w := a.NewWindow("LAN TF2 Server Manager")
	w.Resize(fyne.Size{Width: 1200, Height: 1200})

	cfg, err := config.Load("config/config.toml")
	if err != nil {
		log.Printf("Failed to load config.toml, using defaults: %v", err)
		cfg = config.Default()
	}

	manager := ui.NewManager(w, a.Preferences(), cfg)
	w.SetContent(manager.Content())
	w.ShowAndRun()
}
