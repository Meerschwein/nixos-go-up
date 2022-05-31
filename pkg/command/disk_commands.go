package command

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"github.com/Meerschwein/nixos-go-up/pkg/configuration"
	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/pkg/util"
)

const (
	SALT_LENGTH = 16
	KEYLENGTH   = 512
	ITERATIONS  = 1000000
	CIPHER      = "aes-xts-plain64"
	HASH        = "sha512"
	SLOT        = 2
)

func PartitioningCommands(d disk.Disk, firmware configuration.Firmware) (cmds []Command) {
	for _, p := range d.Partitions {
		partType := "extended"
		if p.Primary {
			partType = "primary"
		}

		if p.Bootable && firmware == configuration.UEFI {
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
	return
}

func FormattingCommands(disk disk.Disk) (cmds []Command) {
	for _, p := range disk.Partitions {
		if disk.Encrypt && !p.Bootable {
			if disk.Yubikey {
				cmds = append(cmds, FormatAndEncryptPartitionWithYubikey(p, disk.EncryptionPasswd)...)
			} else {
				cmds = append(cmds, FormatAndEncryptPartition(p, disk.EncryptionPasswd)...)
			}
		} else {
			cmds = append(cmds, FormatPartition(p))
		}
	}
	return
}

func PartitioningTableCommand(d disk.Disk) (cmd Command) {
	switch d.PartitionTable {
	case disk.Mbr:
		cmd = ShellCommand{
			Label: fmt.Sprintf("Formatting %s to MBR", d.Name),
			Cmd:   fmt.Sprintf("parted -s /dev/%s -- mklabel msdos", d.Name),
		}
	case disk.Gpt:
		cmd = ShellCommand{
			Label: fmt.Sprintf("Formatting %s to GPT", d.Name),
			Cmd:   fmt.Sprintf("parted -s /dev/%s -- mklabel gpt", d.Name),
		}
	default:
		util.ExitIfErr(fmt.Errorf("unrecognized partitioning scheme %s! Aborting... ", d.PartitionTable))
	}

	return
}

func DiskCommands(conf configuration.Conf) (cmds []Command) {
	cmds = append(cmds, PartitioningTableCommand(conf.Disk))
	cmds = append(cmds, PartitioningCommands(conf.Disk, conf.Firmware)...)
	cmds = append(cmds, FormattingCommands(conf.Disk)...)
	return
}

func FormatPartition(p disk.Partition) Command {
	var labelArgs string
	switch p.Format {
	case disk.Ext4:
		labelArgs = "-L " + p.Label
	case disk.Fat32:
		labelArgs = "-n " + p.Label
	default:
		util.ExitIfErr(fmt.Errorf("unrecognized filesystem %s! Aborting... ", p.Format))
	}
	if p.Label == "" {
		labelArgs = ""
	}

	cmd := ShellCommand{
		Label: fmt.Sprintf("Formatting %s to %s", p.Path, p.Format),
	}
	switch p.Format {
	case disk.Ext4:
		cmd.Cmd = fmt.Sprintf("mkfs.ext4 %s %s", labelArgs, p.Path)
	case disk.Fat32:
		cmd.Cmd = fmt.Sprintf("mkfs.fat -F32 %s %s", labelArgs, p.Path)
	default:
		util.ExitIfErr(fmt.Errorf("unrecognized filesystem %s! Aborting... ", p.Format))
	}

	return cmd
}

func FormatPartitionMapped(p disk.Partition) Command {
	p.Path = "/dev/mapper/" + p.Label
	p.Label = ""

	return FormatPartition(p)
}

func FormatAndEncryptPartition(p disk.Partition, encryptionPasswd string) []Command {
	return []Command{
		ShellCommand{
			Label: "Encrypt " + p.Path,
			Cmd: fmt.Sprintf(
				"echo -n \"%s\" | cryptsetup luksFormat %s --key-file /dev/stdin -M luks2 --pbkdf argon2id -i 5000",
				util.EscapeBashDoubleQuotes(encryptionPasswd),
				p.Path,
			),
		},
		ShellCommand{
			Label: "Open LUKS partition",
			Cmd: fmt.Sprintf(
				"echo -n \"%s\" | cryptsetup luksOpen %s %s --key-file /dev/stdin",
				util.EscapeBashDoubleQuotes(encryptionPasswd),
				p.Path,
				p.Label,
			),
		},
		FormatPartitionMapped(p),
	}
}

func FormatAndEncryptPartitionWithYubikey(p disk.Partition, encryptionPasswd string) (cmds []Command) {
	salt_b := make([]byte, SALT_LENGTH)
	rand.Read(salt_b)
	salt := hex.EncodeToString(salt_b)

	challenge_b := sha512.Sum512([]byte(salt))
	challenge := hex.EncodeToString(challenge_b[:])

	// TODO Figure out slots
	cmds = append(cmds, ShellCommand{
		Label:    "Challenge the yubikey to a reponse",
		Cmd:      fmt.Sprintf("ykchalresp -%d -x %s 2>/dev/null", SLOT, challenge),
		OutLabel: "YUBI_RESPONSE",
	})

	cmd := ShellCommand{
		Label:    "Hash the yubikey response",
		OutLabel: "YUBI_LUKS_PASS",
	}
	if encryptionPasswd != "" {
		cmd.Cmd = fmt.Sprintf("echo -n '%s' | pbkdf2-sha512 %d %d $YUBI_RESPONSE", encryptionPasswd, KEYLENGTH/8, ITERATIONS)
	} else {
		cmd.Cmd = fmt.Sprintf("echo | pbkdf2-sha512 %d %d $YUBI_RESPONSE", KEYLENGTH/8, ITERATIONS)
	}
	cmds = append(cmds, cmd)

	// TODO Sometimes crashes
	cmds = append(cmds, ShellCommand{
		Label: "Format Cryptsetup",
		Cmd: fmt.Sprintf(
			`echo -n "$YUBI_LUKS_PASS" | cryptsetup luksFormat --cipher="%s" --key-size="%d" --hash="%s" --key-file=- "%s"`,
			CIPHER,
			KEYLENGTH,
			HASH,
			p.Path,
		),
	})

	cmds = append(cmds, ShellCommand{
		Label: "Open Luks",
		Cmd: fmt.Sprintf(
			`echo -n "$YUBI_LUKS_PASS" | cryptsetup luksOpen %s %s --key-file=-`,
			p.Path,
			p.Label,
		),
	})

	cmds = append(cmds, FormatPartitionMapped(p))

	cmds = append(cmds, MountByLabel(BOOTLABEL, "/root/boot")...)

	cmds = append(cmds,
		CreateDir("/root/boot/crypt-storage"),
		ShellCommand{
			Label: "Write into Cryptstore",
			Cmd:   fmt.Sprintf(`echo -ne "%s\n%d" > /root/boot/crypt-storage/default`, salt, ITERATIONS),
		},
		Unmount("/root/boot"),
	)

	return
}

func BIOSDiskSetup(conf configuration.Conf) (configuration.Conf, []Command) {
	conf.Disk.PartitionTable = disk.Mbr

	conf.Disk.Partitions = []disk.Partition{
		{
			Format:   disk.Ext4,
			Label:    ROOTLABEL,
			Path:     "/dev/" + conf.Disk.PartitionName(1),
			Number:   1,
			Primary:  true,
			From:     "1MiB",
			To:       "100%",
			Bootable: false,
		},
	}

	return conf, []Command{}
}

func UEFIDiskSetup(conf configuration.Conf) (configuration.Conf, []Command) {
	conf.Disk.PartitionTable = disk.Gpt

	conf.Disk.Partitions = []disk.Partition{
		{
			Format:   disk.Fat32,
			Label:    BOOTLABEL,
			Path:     "/dev/" + conf.Disk.PartitionName(1),
			Number:   1,
			Primary:  false,
			From:     "4MiB",
			To:       "512MiB",
			Bootable: true,
		},
		{
			Format:   disk.Ext4,
			Label:    ROOTLABEL,
			Path:     "/dev/" + conf.Disk.PartitionName(2),
			Number:   2,
			Primary:  true,
			From:     "512MiB",
			To:       "100%",
			Bootable: false,
		},
	}

	return conf, []Command{}
}
