package command

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/pkg/selection"
	"github.com/Meerschwein/nixos-go-up/pkg/util"
)

type CommandGenerator func(selection.Selection) (selection.Selection, []Command)

func MakeCommandGenerators(sel selection.Selection) (gens []CommandGenerator) {
	if util.IsUefiSystem() {
		gens = append(gens, FormatDiskEfi)
	} else {
		gens = append(gens, FormatDiskLegacy)
	}

	if sel.Disk.Encrypt {
		gens = append(gens, Stable(MountDir("/dev/mapper/"+ROOTLABEL, "/mnt")...))
	} else {
		gens = append(gens, Stable(MountByLabel(ROOTLABEL, "/mnt")...))
	}

	if util.IsUefiSystem() {
		gens = append(gens, Stable(MountByLabel(BOOTLABEL, "/mnt/boot")...))
	}

	gens = append(gens,
		GenerateNixosConfig,
		Stable(ShellCommand{
			Label: "Running nixos-install",
			Cmd:   "nixos-install --no-root-passwd",
		}),
	)

	return
}

func Stable(cmd ...Command) CommandGenerator {
	return func(sel selection.Selection) (selection.Selection, []Command) {
		return sel, cmd
	}
}

func GenerateCommands(sel selection.Selection, generators []CommandGenerator) (cmds []Command) {
	loopSel := sel
	for _, gen := range generators {
		sel, c := gen(loopSel)
		loopSel = sel
		cmds = append(cmds, c...)
	}
	return
}

type Replacement struct {
	Bootloader           string
	GrubDevice           string
	Hostname             string
	Timezone             string
	NetworkingInterfaces string
	Desktopmanager       string
	KeyboardLayout       string
	Username             string
	PasswordHash         string
}

func GenerateCustomNixosConfig(sel selection.Selection) (string, error) {
	replacement := Replacement{
		Hostname:       sel.Hostname,
		Timezone:       sel.Timezone,
		Desktopmanager: selection.NixConfiguration(sel.DesktopEnviroment),
		KeyboardLayout: sel.KeyboardLayout,
		Username:       sel.Username,
		PasswordHash:   util.MkPasswd(sel.Password),
	}

	interfaces, err := util.GetInterfaces()
	if err != nil {
		return "", err
	}

	inters := ""
	for _, inter := range interfaces {
		inters += "networking.interfaces." + inter + ".useDHCP = true;\n  "
	}
	replacement.NetworkingInterfaces = inters

	if util.IsUefiSystem() {
		replacement.Bootloader = "boot.loader.systemd-boot.enable = true;"
		replacement.GrubDevice = "nodev"
	} else {
		replacement.Bootloader = "boot.loader.grub.enable = true;\n  boot.loader.grub.version = 2;"
		replacement.GrubDevice = "/dev/" + sel.Disk.Name
	}

	t := template.Must(template.New("NixOS configuration.nix").Parse(NixOSConfiguration()))

	var data bytes.Buffer
	err = t.Execute(&data, replacement)

	if err != nil {
		return "ERROR parsing the template", err
	}

	return data.String(), nil
}

func GenerateNixosConfig(sel selection.Selection) (s selection.Selection, cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: "Generate default nixos configuration at /mnt",
		Cmd:   "nixos-generate-config --root /mnt",
	})

	config, _ := GenerateCustomNixosConfig(sel)
	cmds = append(cmds, ShellCommand{
		Label: "Generate custom nixos configuration file",
		Cmd:   fmt.Sprintf(`echo "%s" > /mnt/etc/nixos/configuration.nix`, util.EscapeBashDoubleQuotes(config)),
	})

	if sel.Disk.Yubikey {
		// TODO
		// This is super hacky
		bootPart := disk.Partition{}
		storagePart := disk.Partition{}
		for _, p := range sel.Disk.Partitions {
			if p.Bootable {
				bootPart = p
			} else {
				storagePart = p
			}
		}
		cmds = append(cmds, ShellCommand{
			Label: "Modifying hardware-configuration.nix",
			Cmd: fmt.Sprintf(
				`echo "
// {
  boot.initrd.kernelModules = [ \"vfat\" \"nls_cp437\" \"nls_iso8859-1\" \"usbhid\" ];
  boot.initrd.luks.yubikeySupport = true;
  boot.initrd.luks.devices = {
    \"%s\" = {
      device = \"/dev/%s\";
      preLVM = true;
      yubikey = {
        slot = %d;
        twoFactor = %v;
        storage = {
          device = \"/dev/%s\";
        };
      };
    };
  };
}" >> /mnt/etc/nixos/hardware-configuration.nix`,
				storagePart.Label,
				storagePart.Path,
				SLOT,
				sel.Disk.EncryptionPasswd != "",
				bootPart.Path,
			),
		})

	}

	return sel, cmds
}
