package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"lan-server-manager/ui"
)

func main() {
	a := app.NewWithID("com.lan.server-manager")
	w := a.NewWindow("LAN TF2 Server Manager")
	w.Resize(fyne.Size{Width: 900, Height: 700})

	manager := ui.NewManager(w, a.Preferences())
	w.SetContent(manager.Content())
	w.ShowAndRun()
}
