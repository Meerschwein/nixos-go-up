package configuration

import "github.com/Meerschwein/nixos-go-up/pkg/util"

type Firmware string

var (
	UEFI Firmware = "uefi"
	BIOS Firmware = "bios"
)

func (c Conf) SetFirmware() Conf {
	if util.IsUefiSystem() {
		c.Firmware = UEFI
	} else {
		c.Firmware = BIOS
	}

	return c
}

func (c Conf) IsUEFI() bool {
	return c.Firmware == UEFI
}
