package assets

import (
	"embed"
	"fmt"

	"fyne.io/fyne/v2"

	"lan-server-manager/game/logparse"
)

//go:embed "static/class_icons/*.png"
var iconsFS embed.FS

// ClassIcon returns the embedded PNG resource for a player class.
func ClassIcon(class logparse.PlayerClass) fyne.Resource {
	name := classFileName(class)
	if name == "" {
		return nil
	}

	path := "static/class_icons/Leaderboard_class_" + name + ".png"
	data, err := iconsFS.ReadFile(path)
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource(name+".png", data)
}

func classFileName(class logparse.PlayerClass) string {
	switch class {
	case logparse.ClassScout:
		return "scout"
	case logparse.ClassSoldier:
		return "soldier"
	case logparse.ClassPyro:
		return "pyro"
	case logparse.ClassDemoman:
		return "demoman"
	case logparse.ClassHeavy:
		return "heavy"
	case logparse.ClassEngineer:
		return "engineer"
	case logparse.ClassMedic:
		return "medic"
	case logparse.ClassSniper:
		return "sniper"
	case logparse.ClassSpy:
		return "spy"
	default:
		return "allclass"
	}
}

// ClassName returns a display name for a player class.
func ClassName(class logparse.PlayerClass) string {
	switch class {
	case logparse.ClassScout:
		return "Scout"
	case logparse.ClassSoldier:
		return "Soldier"
	case logparse.ClassPyro:
		return "Pyro"
	case logparse.ClassDemoman:
		return "Demoman"
	case logparse.ClassHeavy:
		return "Heavy"
	case logparse.ClassEngineer:
		return "Engineer"
	case logparse.ClassMedic:
		return "Medic"
	case logparse.ClassSniper:
		return "Sniper"
	case logparse.ClassSpy:
		return "Spy"
	default:
		return fmt.Sprintf("Class(%d)", class)
	}
}
