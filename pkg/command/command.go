package command

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"
	"time"

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
		if string(out) != "" {
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

type RepeatedFunctionCommand struct {
	Label string
	Func  func() (success bool)
	Limit int
	Wait  time.Duration
}

func (c RepeatedFunctionCommand) Message() string {
	return c.Label
}

func (c RepeatedFunctionCommand) Execute() (msg string, err error) {
	i := 0
	for ; i < c.Limit; i++ {
		if c.Func() {
			msg = "Ran " + strconv.Itoa(i) + " times"
			return
		}
		time.Sleep(c.Wait)
	}
	if i >= c.Limit {
		msg = "Exceeded Limit of " + strconv.Itoa(c.Limit)
	}
	return

}

func (c RepeatedFunctionCommand) DryRun() string {
	return fmt.Sprintf("Ran a function at worst %d times", c.Limit)
}

func Sleep(secs int) Command {
	return ShellCommand{
		Label: fmt.Sprintf("Sleep for %d seconds", secs),
		Cmd:   fmt.Sprintf("sleep %ds", secs),
	}
}

type OutputShellcommand struct {
	Label string
	Cmd   string
	Func  func(string) (string, error)
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

func WaitUntilFormattingSuccess(sel selection.Selection) (s selection.Selection, cmds []Command) {
	cmds = append(cmds, RepeatedFunctionCommand{
		Label: "Wait until all partitions have appeared",
		Func: func() bool {
			partitionPath := "/dev/" + sel.Disk.PartitionName(1)
			_, err := os.Stat(partitionPath)
			return err == nil
		},
		Limit: 10,
		Wait:  1 * time.Second,
	})

	return sel, cmds
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
	replacement := Replacement{}

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

	replacement.Hostname = sel.Hostname
	replacement.Timezone = sel.Timezone
	replacement.KeyboardLayout = sel.KeyboardLayout
	replacement.Username = sel.Username
	replacement.Desktopmanager = selection.NixConfiguration(sel.DesktopEnviroment)

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

func RefreshBlockIndices(sel selection.Selection) (s selection.Selection, cmds []Command) {
	cmds = append(cmds, RepeatedFunctionCommand{
		Label: "Refresh blockindices to prevent mountung errors",
		Func: func() bool {
			err := Run("blockdev", "--rereadpt", "/dev/"+sel.Disk.Name)
			return err == nil
		},
		Limit: 10,
		Wait:  1 * time.Second,
	})

	return sel, cmds
}
