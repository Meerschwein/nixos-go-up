package main

import (
	"flag"
	"fmt"

	"github.com/Meerschwein/nixos-go-up/pkg/command"
	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/pkg/selection"
	"github.com/Meerschwein/nixos-go-up/pkg/util"

	"os"
	"time"

	"os/exec"
	"strings"
)

var (
	dryRun bool
)

const (
	BOOTLABEL = "NIXBOOT"
	ROOTLABEL = "NIXROOT"
)

func init() {
	flag.BoolVar(&dryRun, "dry-run", false, "dry-run")

}

type CommandGenerator func(sel selection.Selection) (cmds []command.Command)

func main() {
	flag.Parse()
	
	if !util.WasRunAsRoot() {
		fmt.Println("Please run as root!")
		return
	}

	if util.MountIsUsed() && !dryRun {
		util.ExitIfErr(fmt.Errorf("something is was found at /mnt"))
	}

	selectionSteps := []selection.SelectionStep{
		selection.SelectDisk,
		selection.SelectHostname,
		selection.SelectTimezone,
		selection.SelectDesktopEnviroment,
		selection.SelectKeyboardLayout,
		selection.SelectUsername,
		selection.SelectPassword,
	}

	input, err := GetSelections(selectionSteps)
	util.ExitIfErr(err)

	fmt.Printf("Your Selection so far:\n%s\n", input)

	cont := selection.ConfirmationDialog("Are you sure you want to continue?")
	if !cont {
		fmt.Println("Aborting...")
		return
	}

	uefiGenerators := []CommandGenerator{
		FormatDiskEfi,
		WaitUntilFormattingSuccess,
		RefreshBlockIndices,
		MountRootToMnt, UefiMountBootDir,
		GenerateDefaultNixosConfig,
		NixosInstall,
	}

	biosGenerators := []CommandGenerator{
		FormatDiskLegacy,
		WaitUntilFormattingSuccess,
		RefreshBlockIndices,
		MountRootToMnt,
		GenerateDefaultNixosConfig,
		NixosInstall,
	}

	var cmds []command.Command
	if util.IsUefiSystem() {
		cmds = GenerateCommands(input, uefiGenerators)
	} else {
		cmds = GenerateCommands(input, biosGenerators)
	}

	if dryRun {
		command.DryRun(cmds)
	} else {
		command.Run(cmds)
	}
}

func GenerateCommands(sel selection.Selection, generators []CommandGenerator) (cmds []command.Command) {
	for _, gen := range generators {
		cmds = append(cmds, gen(sel)...)
	}
	return
}

func GetSelections(steps []selection.SelectionStep) (sel selection.Selection, err error) {
	for _, step := range steps {
		sel, err = step(sel)
		if err != nil {
			return
		}
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
	if dryRun {
		return name + " " + strings.Join(args, " "), nil
	} else {
		out, err := exec.Command(name, args...).Output()
		return string(out), err
	}
}

func FormatDiskLegacy(sel selection.Selection) (cmds []command.Command) {
	d := sel.Disk

	d.Table = disk.Mbr

	d.Partitions = []disk.Partition{
		{
			Format:   disk.Ext4,
			Label:    ROOTLABEL,
			Path:     d.PartitionName(1),
			Number:   1,
			Primary:  true,
			From:     "1MiB",
			To:       "100%",
			Bootable: false,
		},
	}

	cmds = d.Commands(disk.BIOS)

	return
}

func FormatDiskEfi(sel selection.Selection) (cmds []command.Command) {
	d := sel.Disk

	d.Table = disk.Gpt

	d.Partitions = []disk.Partition{
		{
			Format:   disk.Fat32,
			Label:    BOOTLABEL,
			Path:     d.PartitionName(1),
			Number:   1,
			Primary:  false,
			From:     "4MiB",
			To:       "512MiB",
			Bootable: true,
		},
		{
			Format:   disk.Ext4,
			Label:    ROOTLABEL,
			Path:     d.PartitionName(2),
			Number:   2,
			Primary:  true,
			From:     "512MiB",
			To:       "100%",
			Bootable: false,
		},
	}

	cmds = d.Commands(disk.UEFI)

	return
}

func PasswordHash(password string) (string, error) {
	pass, err := RunWithOutput("mkpasswd", "--method=sha-512", password)
	return strings.TrimSpace(pass), err
}

func WaitUntilFormattingSuccess(sel selection.Selection) (cmds []command.Command) {
	cmds = append(cmds, command.RepeatedFunctionCommand{
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

func MountRootToMnt(_ selection.Selection) (cmds []command.Command) {
	cmds = append(cmds, command.ShellCommand{
		Label: fmt.Sprintf("Mounting %s at /mnt", ROOTLABEL),
		Cmd:   fmt.Sprintf("mount /dev/disk/by-label/%s /mnt", ROOTLABEL),
	})

	return
}

func NixosInstall(_ selection.Selection) (cmds []command.Command) {
	cmds = append(cmds, command.ShellCommand{
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
		replacements = append(replacements, [2]string{"$GRUB_DEVICE_C$", "# "})
		replacements = append(replacements, [2]string{"$GRUB_DEVICE$", "nodev"})
	} else {
		replacements = append(replacements, [2]string{"$GRUB_DEVICE_C$", ""})
		replacements = append(replacements, [2]string{"$GRUB_DEVICE$", "/dev/" + sel.Disk.Name})
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

func GenerateDefaultNixosConfig(sel selection.Selection) (cmds []command.Command) {
	cmds = append(cmds, command.ShellCommand{
		Label: "Generate default nixos configuration at /mnt",
		Cmd:   "nixos-generate-config --root /mnt",
	})

	cmds = append(cmds, command.FunctionCommand{
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

func UefiMountBootDir(_ selection.Selection) (cmds []command.Command) {
	cmds = append(cmds, command.ShellCommand{
		Label: "Create /mnt/boot",
		Cmd:   "mkdir -p/mnt/boot",
	})

	cmds = append(cmds, command.ShellCommand{
		Label: fmt.Sprintf("Mounting %s to /mnt/boot", BOOTLABEL),
		Cmd:   fmt.Sprintf("mount /dev/disk/by-label/%s /mnt/boot", BOOTLABEL),
	})

	return
}

func RefreshBlockIndices(sel selection.Selection) (cmds []command.Command) {
	cmds = append(cmds, command.RepeatedFunctionCommand{
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
