package command

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"text/template"

	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/pkg/selection"
	"github.com/Meerschwein/nixos-go-up/pkg/util"
	"github.com/Meerschwein/nixos-go-up/pkg/vars"
)

const (
	BOOTLABEL = "NIXBOOT"
	ROOTLABEL = "NIXROOT"
)

type Command interface {
	Message() string
	Execute() (string, error)
	DryRun() string
}

func DryRun(cmds []Command) {
	for _, cmd := range cmds {
		fmt.Printf("--\n%s\n%s\n", cmd.Message(), cmd.DryRun())
	}
}

func RunCmds(cmds []Command) {
	for _, cmd := range cmds {
		fmt.Printf("--\n%s\n", cmd.Message())
		out, err := cmd.Execute()
		if out != "" {
			fmt.Println(out)
		}
		util.ExitIfErr(err)
	}
}

type ShellCommand struct {
	Label string
	Cmd   string
}

func (c ShellCommand) Message() string {
	return c.Label
}

func (c ShellCommand) Execute() (string, error) {
	out, err := exec.Command("bash", "-c", c.Cmd).CombinedOutput()
	return string(out), err
}

func (c ShellCommand) DryRun() string {
	return c.Cmd
}

func Sleep(secs int) Command {
	return ShellCommand{
		Label: fmt.Sprintf("Sleep for %d seconds", secs),
		Cmd:   fmt.Sprintf("sleep %ds", secs),
	}
}

func MakeCommandGenerators(sel selection.Selection) (generators []CommandGenerator) {
	if util.IsUefiSystem() {
		generators = append(generators, FormatDiskEfi)
	} else {
		generators = append(generators, FormatDiskLegacy)
	}

	generators = append(generators,
		func(sel selection.Selection) (selection.Selection, []Command) {
			return sel, []Command{Sleep(5)}
		},
	)

	if sel.Disk.Encrypt {
		generators = append(generators, MountEncryptedRootToMnt)
	} else {
		generators = append(generators, MountRootToMnt)
	}

	if util.IsUefiSystem() {
		generators = append(generators, UefiMountBootDir)
	}

	generators = append(generators, GenerateNixosConfig)

	generators = append(generators, NixosInstall)

	return
}

type CommandGenerator func(sel selection.Selection) (s selection.Selection, cmds []Command)

func GenerateCommands(sel selection.Selection, generators []CommandGenerator) (cmds []Command) {
	loopSel := sel
	for _, gen := range generators {
		sel, c := gen(loopSel)
		loopSel = sel
		cmds = append(cmds, c...)
	}
	return
}

func Run(name string, args ...string) error {
	out, err := RunWithOutput(name, args...)
	if string(out) != "" && err == nil {
		fmt.Println(string(out))
	}
	return err
}

func Run2(cmd string) (string, error) {
	if vars.DryRun {
		return cmd, nil
	} else {
		out, err := exec.Command("bash", "-c", cmd).Output()
		return string(out), err
	}

}

func RunWithOutput(name string, args ...string) (string, error) {
	if vars.DryRun {
		return name + " " + strings.Join(args, " "), nil
	} else {
		out, err := exec.Command(name, args...).Output()
		return string(out), err
	}
}

func PasswordHash(password string) (string, error) {
	pass, err := RunWithOutput("mkpasswd", "--method=sha-512", password)
	return strings.TrimSpace(pass), err
}

func MountEncryptedRootToMnt(sel selection.Selection) (s selection.Selection, cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: fmt.Sprintf("Mounting /dev/mapper/%s at /mnt", ROOTLABEL),
		Cmd:   fmt.Sprintf("mount /dev/mapper/%s /mnt", ROOTLABEL),
	})

	return sel, cmds
}

func MountRootToMnt(sel selection.Selection) (s selection.Selection, cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: fmt.Sprintf("Mounting /dev/disk/by-label/%s at /mnt", ROOTLABEL),
		Cmd:   fmt.Sprintf("mount /dev/disk/by-label/%s /mnt", ROOTLABEL),
	})

	return sel, cmds
}

func NixosInstall(sel selection.Selection) (s selection.Selection, cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: "Running nixos-install",
		Cmd:   "nixos-install --no-root-passwd",
	})

	return sel, cmds
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
	}

	pasHash, err := PasswordHash(sel.Password)
	if err != nil {
		return "", err
	}
	replacement.PasswordHash = pasHash

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

	t := template.Must(template.New("nixosconfiguration").Parse(NixOSConfiguration()))

	data := bytes.Buffer{}
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
		Cmd:   fmt.Sprintf("echo '%s' > /mnt/etc/nixos/configuration.nix", config),
	})

	if sel.Disk.Yubikey {
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
				`echo '// {
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
				}' >> /mnt/etc/nixos/hardware-configuration.nix`,
				storagePart.Label,
				storagePart.Path,
				2,
				sel.Disk.EncryptionPasswd != "",
				bootPart.Path,
			),
		})

	}

	return sel, cmds
}

func UefiMountBootDir(sel selection.Selection) (s selection.Selection, cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: "Create /mnt/boot",
		Cmd:   "mkdir -p /mnt/boot",
	})

	cmds = append(cmds, ShellCommand{
		Label: fmt.Sprintf("Mounting %s to /mnt/boot", BOOTLABEL),
		Cmd:   fmt.Sprintf("mount /dev/disk/by-label/%s /mnt/boot", BOOTLABEL),
	})

	return sel, cmds
}
