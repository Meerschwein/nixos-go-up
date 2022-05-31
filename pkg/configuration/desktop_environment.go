package configuration

import (
	"fmt"

	"github.com/Meerschwein/nixos-go-up/pkg/util"
)

type DesktopEnviroment string

const (
	XFCE  DesktopEnviroment = "xfce"
	GNOME DesktopEnviroment = "gnome"
	NONE  DesktopEnviroment = "none"
)

func DesktopEnviroments() []DesktopEnviroment {
	return []DesktopEnviroment{XFCE, GNOME, NONE}
}

func NixExpression(dm DesktopEnviroment) (config string) {
	switch dm {
	case XFCE:
		config = "services.xserver.desktopManager.xfce.enable = true;\n  " +
			"services.xserver.displayManager.defaultSession = \"xfce\";"
	case GNOME:
		config = "services.xserver.desktopManager.gnome.enable = true;\n  " +
			"services.xserver.displayManager.gdm.enable = true;"
	case NONE:
		config = ""
	default:
		util.ExitIfErr(fmt.Errorf("unknown Desktop Enviroment: %s", string(dm)))
	}
	return
}
