package command

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Meerschwein/nixos-go-up/pkg/configuration"
	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/pkg/util"
)

type CommandGenerator func(configuration.Conf) (configuration.Conf, []Command)

func MakeCommandGenerators(conf configuration.Conf) (gens []CommandGenerator) {
	if util.IsUefiSystem() {
		gens = append(gens, FormatDiskEfi)
	} else {
		gens = append(gens, FormatDiskLegacy)
	}

	if conf.Disk.Encrypt {
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
	return func(conf configuration.Conf) (configuration.Conf, []Command) {
		return conf, cmd
	}
}

func GenerateCommands(conf configuration.Conf, generators []CommandGenerator) (cmds []Command) {
	loopSel := conf
	for _, gen := range generators {
		sel, c := gen(loopSel)
		loopSel = sel
		cmds = append(cmds, c...)
	}
	return
}

func GenerateCustomNixosConfig(conf configuration.Conf) string {
	replacement := Replacement{
		Hostname:       conf.Hostname,
		Timezone:       conf.Timezone,
		Desktopmanager: configuration.NixExpression(conf.DesktopEnviroment),
		KeyboardLayout: conf.KeyboardLayout,
		Username:       conf.Username,
		PasswordHash:   util.MkPasswd(conf.Password),
	}

	interfaces, err := util.GetInterfaces()
	util.ExitIfErr(err)

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
		replacement.GrubDevice = "/dev/" + conf.Disk.Name
	}

	t := template.Must(template.New("NixOS configuration.nix").Parse(NixOSConfiguration()))
	var data bytes.Buffer

	err = t.Execute(&data, replacement)
	util.ExitIfErr(err)

	return data.String()
}

func GenerateNixosConfig(conf configuration.Conf) (s configuration.Conf, cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: "Generate default nixos configuration at /mnt",
		Cmd:   "nixos-generate-config --root /mnt",
	})

	config := GenerateCustomNixosConfig(conf)
	cmds = append(cmds, WriteToFile("Generate custom nixos configuration file", config, "/mnt/etc/nixos/configuration.nix"))

	if conf.Disk.Yubikey {
		// TODO
		// This is super hacky
		bootPart := disk.Partition{}
		storagePart := disk.Partition{}
		for _, p := range conf.Disk.Partitions {
			if p.Bootable {
				bootPart = p
			} else {
				storagePart = p
			}
		}

		cmds = append(cmds, AppendToFile(
			"Modifying hardware-configuration.nix",
			fmt.Sprintf(
				`
// {
  boot.initrd.kernelModules = [ "vfat" "nls_cp437" "nls_iso8859-1" "usbhid" ];
  boot.initrd.luks.yubikeySupport = true;
  boot.initrd.luks.devices = {
    "%s" = {
      device = "/dev/%s";
      preLVM = true;
      yubikey = {
        slot = %d;
        twoFactor = %v;
        storage = {
          device = "/dev/%s";
        };
      };
    };
  };
}`,
				storagePart.Label,
				storagePart.Path,
				SLOT,
				conf.Disk.EncryptionPasswd != "",
				bootPart.Path,
			),
			"/mnt/etc/nixos/hardware-configuration.nix",
		))

	}

	return conf, cmds
}
