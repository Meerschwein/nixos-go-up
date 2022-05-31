package command

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/pkg/selection"
)

const (
	SALT_LENGTH = 16
	KEYLENGTH   = 512
	ITERATIONS  = 1000000
	CIPHER      = "aes-xts-plain64"
	HASH        = "sha512"
	SLOT        = 2
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
			if d.Yubikey {
				cmds = append(cmds, MakeEncryptedFilesystemYubikeyCommand(p, d.EncryptionPasswd)...)
			} else {
				cmds = append(cmds, MakeEncryptedFilesystemCommand(p, d.EncryptionPasswd)...)
			}
		} else {
			cmds = append(cmds, MakeDiskFormattingCommand(p.Format, "/dev/"+p.Path, p.Label))
		}
	}

	return
}

func TableCommands(d disk.Disk) (cmds []Command) {
	var cmd ShellCommand
	switch d.PartitionTable {
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
		log.Panicf("unrecognized partitioning scheme %s! Aborting... ", d.PartitionTable)
	}
	cmds = append(cmds, cmd)

	return
}

func MakeDiskFormattingCommand(format disk.Filesystem, path string, label string) Command {
	labelArgs := ""
	switch format {
	case disk.Ext4:
		labelArgs = "-L " + label
	case disk.Fat32:
		labelArgs = "-n " + label
	default:
		log.Panicf("unrecognized filesystem %s! Aborting... ", format)
	}
	if label == "" {
		labelArgs = ""
	}

	cmd := ShellCommand{
		Label: fmt.Sprintf("Formatting %s to %s", path, format),
	}
	switch format {
	case disk.Ext4:
		cmd.Cmd = fmt.Sprintf("mkfs.ext4 %s %s", labelArgs, path)
	case disk.Fat32:
		cmd.Cmd = fmt.Sprintf("mkfs.fat -F32 %s %s", labelArgs, path)
	default:
		log.Panicf("unrecognized filesystem %s! Aborting... ", format)
	}

	return cmd
}

func MakeEncryptedFilesystemCommand(p disk.Partition, encryptionPasswd string) (cmds []Command) {
	cmds = append(cmds, ShellCommand{
		Label: fmt.Sprintf("Encrypt %s", p.Path),
		Cmd:   fmt.Sprintf("echo -n '%s' | cryptsetup luksFormat /dev/%s --key-file /dev/stdin -M luks2 --pbkdf argon2id -i 5000", encryptionPasswd, p.Path),
	}, ShellCommand{
		Label: "Open LUKS partition",
		Cmd:   fmt.Sprintf("echo -n '%s' | cryptsetup luksOpen /dev/%s %s --key-file /dev/stdin", encryptionPasswd, p.Path, p.Label),
	},
	)

	cmds = append(cmds, MakeDiskFormattingCommand(p.Format, "/dev/mapper/"+p.Label, ""))

	return
}

func MakeEncryptedFilesystemYubikeyCommand(p disk.Partition, encryptionPasswd string) (cmds []Command) {
	salt_b := make([]byte, SALT_LENGTH)
	rand.Read(salt_b)
	salt := hex.EncodeToString(salt_b)

	challenge_b := (sha512.Sum512([]byte(salt)))
	challenge := hex.EncodeToString(challenge_b[:])

	// TODO Figure out slots
	cmds = append(cmds, ShellCommand{
		Label:    "Challenge the yubikey to a reponse",
		Cmd:      fmt.Sprintf("ykchalresp -%d -x %s 2>/dev/null", SLOT, challenge),
		OutLabel: "YUBI_RESPONSE",
	})

	if encryptionPasswd != "" {
		cmds = append(cmds, ShellCommand{
			Label:    "Hash the yubikey response",
			Cmd:      fmt.Sprintf("echo -n '%s' | pbkdf2-sha512 %d %d $YUBI_RESPONSE", encryptionPasswd, KEYLENGTH/8, ITERATIONS),
			OutLabel: "YUBI_LUKS_PASS",
		})
	} else {
		cmds = append(cmds, ShellCommand{
			Label:    "Hash the yubikey response",
			Cmd:      fmt.Sprintf("echo | pbkdf2-sha512 %d %d $YUBI_RESPONSE", KEYLENGTH/8, ITERATIONS),
			OutLabel: "YUBI_LUKS_PASS",
		})
	}

	// TODO Sometimes crashes
	cmds = append(cmds, ShellCommand{
		Label: "Format Cryptsetup",
		Cmd: fmt.Sprintf(
			`echo -n "$YUBI_LUKS_PASS" | cryptsetup luksFormat --cipher="%s" --key-size="%d" --hash="%s" --key-file=- "/dev/%s"`,
			CIPHER,
			KEYLENGTH,
			HASH,
			p.Path,
		),
	})

	cmds = append(cmds, ShellCommand{
		Label: "Open Luks",
		Cmd: fmt.Sprintf(
			`echo -n "$YUBI_LUKS_PASS" | cryptsetup luksOpen /dev/%s %s --key-file=-`,
			p.Path,
			p.Label,
		),
	})

	cmds = append(cmds, MakeDiskFormattingCommand(p.Format, "/dev/mapper/"+p.Label, ""))

	cmds = append(cmds, MountByLabel(BOOTLABEL, "/root/boot")...)

	cmds = append(cmds, ShellCommand{
		Label: "Generate cryptstore Dir",
		Cmd:   "mkdir -p /root/boot/crypt-storage",
	})

	cmds = append(cmds, ShellCommand{
		Label: "Write into Cryptstore",
		Cmd:   fmt.Sprintf(`echo -ne "%s\n%d" > /root/boot/crypt-storage/default`, salt, ITERATIONS),
	})

	cmds = append(cmds, Unmount("/root/boot"))

	return
}

func FormatDiskLegacy(sel selection.Selection) (s selection.Selection, cmds []Command) {
	sel.Disk.PartitionTable = disk.Mbr

	sel.Disk.Partitions = []disk.Partition{
		{
			Format:   disk.Ext4,
			Label:    ROOTLABEL,
			Path:     sel.Disk.PartitionName(1),
			Number:   1,
			Primary:  true,
			From:     "1MiB",
			To:       "100%",
			Bootable: false,
		},
	}

	cmds = Commands(sel.Disk, disk.BIOS)

	return sel, cmds
}

func FormatDiskEfi(sel selection.Selection) (s selection.Selection, cmds []Command) {
	sel.Disk.PartitionTable = disk.Gpt

	sel.Disk.Partitions = []disk.Partition{
		{
			Format:   disk.Fat32,
			Label:    BOOTLABEL,
			Path:     sel.Disk.PartitionName(1),
			Number:   1,
			Primary:  false,
			From:     "4MiB",
			To:       "512MiB",
			Bootable: true,
		},
		{
			Format:   disk.Ext4,
			Label:    ROOTLABEL,
			Path:     sel.Disk.PartitionName(2),
			Number:   2,
			Primary:  true,
			From:     "512MiB",
			To:       "100%",
			Bootable: false,
		},
	}

	cmds = Commands(sel.Disk, disk.UEFI)

	return sel, cmds
}
