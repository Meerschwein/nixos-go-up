package command

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Meerschwein/nixos-go-up/pkg/configuration"
	"github.com/Meerschwein/nixos-go-up/pkg/util"
)

type CommandGenerator func(configuration.Conf) (configuration.Conf, []Command)

func MakeCommandGenerators(conf configuration.Conf) (gens []CommandGenerator) {
	if conf.IsUEFI() {
		gens = append(gens, UEFIDiskSetup)
	} else {
		gens = append(gens, BIOSDiskSetup)
	}

	gens = append(gens, Stable(DiskCommands))

	if conf.Disk.Encrypt {
		gens = append(gens, CmdsToGen(MountDir("/dev/mapper/"+ROOTLABEL, "/mnt")...))
	} else {
		gens = append(gens, CmdsToGen(MountByLabel(ROOTLABEL, "/mnt")...))
	}

	if conf.IsUEFI() {
		gens = append(gens, CmdsToGen(MountByLabel(BOOTLABEL, "/mnt/boot")...))
	}

	gens = append(gens,
		WriteNixosConfig,
		CmdsToGen(ShellCommand{
			Label: "Running nixos-install",
			Cmd:   "nixos-install --no-root-passwd",
		}),
	)

	return
}

func CmdsToGen(cmds ...Command) CommandGenerator {
	return func(conf configuration.Conf) (configuration.Conf, []Command) {
		return conf, cmds
	}
}

func Stable(f func(configuration.Conf) []Command) CommandGenerator {
	return func(conf configuration.Conf) (configuration.Conf, []Command) {
		return conf, f(conf)
	}
}

func GenerateCommands(conf configuration.Conf, generators []CommandGenerator) (cmds []Command) {
	loopConf := conf
	for _, gen := range generators {
		conf, c := gen(loopConf)
		loopConf = conf
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

	inters := ""
	for _, inter := range conf.NetInterfaces {
		inters += "networking.interfaces." + inter + ".useDHCP = true;\n  "
	}
	replacement.NetworkingInterfaces = inters

	if conf.IsUEFI() {
		replacement.Bootloader = "boot.loader.systemd-boot.enable = true;"
		replacement.GrubDevice = "nodev"
	} else {
		replacement.Bootloader = "boot.loader.grub.enable = true;\n  boot.loader.grub.version = 2;"
		replacement.GrubDevice = "/dev/" + conf.Disk.Name
	}

	t := template.Must(template.New("NixOS configuration.nix").Parse(NixOSConfiguration()))
	var data bytes.Buffer

	err := t.Execute(&data, replacement)
	util.ExitIfErr(err)

	return data.String()
}

func WriteNixosConfig(conf configuration.Conf) (_ configuration.Conf, cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: "Generate default nixos configuration at /mnt",
		Cmd:   "nixos-generate-config --root /mnt",
	})

	config := GenerateCustomNixosConfig(conf)
	cmds = append(cmds, WriteToFile("Generate custom nixos configuration file", config, "/mnt/etc/nixos/configuration.nix"))

	if conf.Yubikey {
		cmds = append(cmds, AppendToFile(
			"Modifying hardware-configuration.nix",
			fmt.Sprintf(
				`
// {
  boot.initrd.kernelModules = [ "nvme" "vfat" "nls_cp437" "nls_iso8859-1" "usbhid" ];
  boot.initrd.luks.yubikeySupport = true;
  boot.initrd.luks.devices = {
    "%s" = {
      device = "%s";
      preLVM = true;
      yubikey = {
        slot = %d;
        twoFactor = %v;
        storage = {
          device = "%s";
        };
      };
    };
  };
}`,
				conf.Disk.GetRootPartition().Label,
				conf.Disk.GetRootPartition().Path,
				conf.YubikeySlot,
				conf.Disk.EncryptionPasswd != "",
				conf.Disk.GetBootPartition().Path,
			),
			"/mnt/etc/nixos/hardware-configuration.nix",
		))
	}

	return conf, cmds
}
