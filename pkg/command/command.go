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
	Func  func() (string, error)
}

func (c FunctionCommand) Message() string {
	return c.Label
}

func (c FunctionCommand) Execute() (string, error) {
	out, err := c.Func()
	if err != nil {
		return "", fmt.Errorf(out)
	}
	return out, nil
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
		Func: func() (string, error) {
			time.Sleep(t)
			return "slept", nil
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

	t := template.Must(template.New("configuration-template.gotmpl").ParseFiles("configuration-template.gotmpl"))

	data := bytes.Buffer{}
	err = t.Execute(&data, replacement)

	if err != nil {
		return "ERROR parsing the template", err
	}

	return data.String(), nil
}

func GenerateDefaultNixosConfig(sel selection.Selection) (cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: "Generate default nixos configuration at /mnt",
		Cmd:   "nixos-generate-config --root /mnt",
	})

	cmds = append(cmds, FunctionCommand{
		Label: "Generate custom nixos configuration file",
		Func: func() (string, error) {
			config, err := GenerateCustomNixosConfig(sel)
			if err != nil {
				return "", err
			}
			err = os.WriteFile("/mnt/etc/nixos/configuration.nix", []byte(config), 0644)
			return config, err
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
