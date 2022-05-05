package command

import (
	"fmt"
	"log"
	"os"

	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/pkg/selection"
)

func Commands(d disk.Disk, boot disk.Firmware) (cmds []Command) {
	cmds = append(cmds, TableCommands(d)...)

	for _, p := range d.Partitions {
		partType := "extended"
		if p.Primary {
			partType = "primary"
		}

		if p.Bootable && boot == disk.UEFI {
			partType = "ESP"
		}

		fsType := ""
		if p.Format == disk.Fat32 {
			fsType = "fat32"
		}

		cmds = append(cmds, ShellCommand{
			Label: fmt.Sprintf("Create partition %d on %s from %s to %s", p.Number, d.Name, p.From, p.To),
			Cmd: fmt.Sprintf("parted -s /dev/%s -- mkpart %s %s %s %s",
				d.Name,
				partType,
				fsType,
				p.From,
				p.To,
			),
		})

		if p.Bootable {
			cmds = append(cmds, ShellCommand{
				Label: fmt.Sprintf("Set partition %d bootable", p.Number),
				Cmd:   fmt.Sprintf("parted -s /dev/%s -- set %d esp on", d.Name, p.Number),
			})
		}
	}

	for _, p := range d.Partitions {
		if d.Encrypt && !p.Bootable {
			cmds = append(cmds, MakeEncryptedFilesystemCommand(p, d.EncryptionPasswd)...)
		} else {
			cmds = append(cmds, MakeFilesystemCommand(p)...)
		}
	}

	return
}

func TableCommands(d disk.Disk) (cmds []Command) {
	var cmd ShellCommand
	switch d.Table {
	case disk.Mbr:
		cmd = ShellCommand{
			Label: fmt.Sprintf("Formatting %s to MBR", d.Name),
			Cmd:   "parted -s /dev/" + d.Name + " -- mklabel msdos",
		}
	case disk.Gpt:
		cmd = ShellCommand{
			Label: fmt.Sprintf("Formatting %s to GPT", d.Name),
			Cmd:   "parted -s /dev/" + d.Name + " -- mklabel gpt",
		}
	default:
		log.Panicf("unrecognized partitioning scheme %s! Aborting... ", d.Table)
		return
	}
	cmds = append(cmds, cmd)

	return
}

func MakeEncryptedFilesystemCommand(p disk.Partition, encryptionPasswd string) (cmds []Command) {

	luksFile := ".luks-key"
	cmds = append(cmds, FunctionCommand{
		Label: "Generate LUKS key file",
		Func: func() (string, error) {
			return "", os.WriteFile("./"+luksFile, []byte(encryptionPasswd), 0644)
		},
	}, ShellCommand{
		Label: fmt.Sprintf("Encrypt %s", p.Path),
		Cmd:   fmt.Sprintf("cryptsetup luksFormat /dev/%s --key-file ./%s -M luks2 --pbkdf argon2id -i 5000", p.Path, luksFile),
	}, ShellCommand{
		Label: "Open LUKS partition",
		Cmd:   fmt.Sprintf("cryptsetup luksOpen /dev/%s %s --key-file ./%s", p.Path, p.Label, luksFile),
	}, ShellCommand{
		Label: "Delete LUKS key file",
		Cmd:   fmt.Sprintf("shred ./%s", luksFile),
	})

	cmd := ShellCommand{
		Label: fmt.Sprintf("Partition /dev/%s to %s", p.Path, p.Format),
	}

	switch p.Format {
	case disk.Ext4:
		cmd.Cmd = fmt.Sprintf("mkfs.ext4 /dev/mapper/%s", p.Label)
	case disk.Fat32:
		cmd.Cmd = fmt.Sprintf("mkfs.fat -F32 /dev/mapper/%s", p.Label)
	default:
		log.Panicf("unrecognized filesystem %s! Aborting... ", p.Format)
	}

	cmds = append(cmds, cmd)

	return
}

func MakeFilesystemCommand(p disk.Partition) (cmds []Command) {
	cmd := ShellCommand{
		Label: fmt.Sprintf("Partition /dev/%s to %s", p.Path, p.Format),
	}

	switch p.Format {
	case disk.Ext4:
		cmd.Cmd = fmt.Sprintf("mkfs.ext4 -L %s /dev/%s", p.Label, p.Path)
	case disk.Fat32:
		cmd.Cmd = fmt.Sprintf("mkfs.fat -F32 -n %s /dev/%s", p.Label, p.Path)
	default:
		log.Panicf("unrecognized filesystem %s! Aborting... ", p.Format)
	}

	cmds = append(cmds, cmd)

	return
}

func FormatDiskLegacy(sel selection.Selection) (cmds []Command) {
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

	cmds = Commands(d, disk.BIOS)

	return
}

func FormatDiskEfi(sel selection.Selection) (cmds []Command) {
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

	cmds = Commands(d, disk.UEFI)

	return
}
