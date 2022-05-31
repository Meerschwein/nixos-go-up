package configuration

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Meerschwein/nixos-go-up/pkg/disk"
)

type Conf struct {
	Disk              disk.Disk
	Hostname          string
	Timezone          string
	Username          string
	Password          string
	DesktopEnviroment DesktopEnviroment
	KeyboardLayout    string
}

func (c Conf) String() (res string) {
	encrypt := strconv.FormatBool(c.Disk.Encrypt)
	if c.Disk.Yubikey {
		encrypt += " with Yubikey"
	}

	return fmt.Sprintf(`
Disk:         %s
Encyrpt disk: %v
Hostname:     %s
Timezone:     %s
Desktop:      %s
Keyboard:     %s
Username:     %s
Password:     %s`,
		c.Disk.Name,
		encrypt,
		c.Hostname,
		c.Timezone,
		c.DesktopEnviroment,
		c.KeyboardLayout,
		c.Username,
		strings.Repeat("*", len(c.Password)),
	)
}
