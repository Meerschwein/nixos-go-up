package command

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

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
	s := strings.Split(c.Cmd, " ")
	name := s[0]
	args := s[1:]
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

func (c ShellCommand) DryRun() string {
	return c.Cmd
}

type FunctionCommand struct {
	Label string
	Func  func() (success bool)
}

func (c FunctionCommand) Message() string {
	return c.Label
}

func (c FunctionCommand) Execute() (string, error) {
	success := c.Func()
	if !success {
		return "", fmt.Errorf("unsuccessfull")
	}
	return "", nil
}

func (c FunctionCommand) DryRun() string {
	return "Ran a function"
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

func Sleep(t time.Duration) Command {
	return FunctionCommand{
		Label: fmt.Sprintf("Sleep for %f s", t.Seconds()),
		Func: func() bool {
			time.Sleep(t)
			return true
		},
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
		WaitUntilFormattingSuccess,
		RefreshBlockIndices,
		func(_ selection.Selection) []Command {
			return []Command{Sleep(2 * time.Second)}
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

	generators = append(generators,
		GenerateDefaultNixosConfig,
		NixosInstall,
	)

	return
}

type CommandGenerator func(sel selection.Selection) (cmds []Command)

func GenerateCommands(sel selection.Selection, generators []CommandGenerator) (cmds []Command) {
	for _, gen := range generators {
		cmds = append(cmds, gen(sel)...)
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

func WaitUntilFormattingSuccess(sel selection.Selection) (cmds []Command) {
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

	return
}

func MountEncryptedRootToMnt(_ selection.Selection) (cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: fmt.Sprintf("Mounting /dev/mapper/%s at /mnt", ROOTLABEL),
		Cmd:   fmt.Sprintf("mount /dev/mapper/%s /mnt", ROOTLABEL),
	})

	return
}

func MountRootToMnt(sel selection.Selection) (cmds []Command) {

	cmds = append(cmds, ShellCommand{
		Label: fmt.Sprintf("Mounting /dev/disk/by-label/%s at /mnt", ROOTLABEL),
		Cmd:   fmt.Sprintf("mount /dev/disk/by-label/%s /mnt", ROOTLABEL),
	})

	return
}

func NixosInstall(_ selection.Selection) (cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: "Running nixos-install",
		Cmd:   "nixos-install --no-root-passwd",
	})

	return
}

func GenerateCustomNixosConfig(sel selection.Selection) (string, error) {
	pasHash, err := PasswordHash(sel.Password)
	if err != nil {
		return "", err
	}

	interfaces, err := util.GetInterfaces()
	if err != nil {
		return "", err
	}

	inters := ""
	for _, inter := range interfaces {
		inters += "networking.interfaces." + inter + ".useDHCP = true;\n  "
	}

	replacements := [][2]string{
		{"$HOSTNAME$", sel.Hostname},
		{"$TIMEZONE$", sel.Timezone},
		{"$KEYBOARD_LAYOUT$", sel.KeyboardLayout},
		{"$USERNAME$", sel.Username},
		{"$PASSWORD$", pasHash},
		{"$NETWORKING_INTERFACES$", inters},
		{"$DESKTOP_MANAGER$", selection.NixConfiguration(sel.DesktopEnviroment)},
	}

	if util.IsUefiSystem() {
		replacements = append(replacements, [2]string{"$BOOTLOADER$", "boot.loader.systemd-boot.enable = true;"})
		replacements = append(replacements, [2]string{"$GRUB_DEVICE$", "nodev"})
	} else {
		replacements = append(replacements, [2]string{"$BOOTLOADER$", "boot.loader.grub.enable = true;\n  boot.loader.grub.version = 2;"})
		replacements = append(replacements, [2]string{"$GRUB_DEVICE$", "/dev/" + sel.Disk.Name})
	}

	if sel.Disk.Encrypt {
		replacements = append(replacements, [2]string{"$GRUB_ENCRYTION$", ""})
	} else {
		replacements = append(replacements, [2]string{"$GRUB_ENCRYTION$", "# "})
	}

	dataB, err := os.ReadFile("configuration-template.nix")
	if err != nil {
		return "", err
	}

	data := string(dataB)
	for _, rep := range replacements {
		data = strings.Replace(data, rep[0], rep[1], 1)
	}

	return data, nil
}

func GenerateDefaultNixosConfig(sel selection.Selection) (cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: "Generate default nixos configuration at /mnt",
		Cmd:   "nixos-generate-config --root /mnt",
	})

	cmds = append(cmds, FunctionCommand{
		Label: "Generate custom nixos configuration file",
		Func: func() (success bool) {
			config, err := GenerateCustomNixosConfig(sel)
			if err != nil {
				return false
			}
			err = os.WriteFile("/mnt/etc/nixos/configuration.nix", []byte(config), 0644)
			return err == nil
		},
	})

	return
}

func UefiMountBootDir(sel selection.Selection) (cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: "Create /mnt/boot",
		Cmd:   "mkdir -p /mnt/boot",
	})

	cmds = append(cmds, ShellCommand{
		Label: fmt.Sprintf("Mounting %s to /mnt/boot", BOOTLABEL),
		Cmd:   fmt.Sprintf("mount /dev/disk/by-label/%s /mnt/boot", BOOTLABEL),
	})

	return
}

func RefreshBlockIndices(sel selection.Selection) (cmds []Command) {
	cmds = append(cmds, RepeatedFunctionCommand{
		Label: "Refresh blockindices to prevent mountung errors",
		Func: func() bool {
			err := Run("blockdev", "--rereadpt", "/dev/"+sel.Disk.Name)
			return err == nil
		},
		Limit: 10,
		Wait:  1 * time.Second,
	})

	return
}
