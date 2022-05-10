package command

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/pkg/selection"
)

const (
	SALT_LENGTH = 16
	KEYLENGTH   = 512
	ITERATIONS  = 1000000
	CIPHER      = "aes-xts-plain64"
	HASH        = "sha512"
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
	cmds = append(cmds, ShellCommand{
		Label: fmt.Sprintf("Encrypt %s", p.Path),
		Cmd:   fmt.Sprintf("echo -n '%s' | cryptsetup luksFormat /dev/%s --key-file /dev/stdin -M luks2 --pbkdf argon2id -i 5000", encryptionPasswd, p.Path),
	}, ShellCommand{
		Label: "Open LUKS partition",
		Cmd:   fmt.Sprintf("echo -n '%s' | cryptsetup luksOpen /dev/%s %s --key-file /dev/stdin", encryptionPasswd, p.Path, p.Label),
	},
	)

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

func MakeEncryptedFilesystemYubikeyCommand(p disk.Partition, encryptionPasswd string) (cmds []Command) {
	// TODO
	// These instuctions are executed when they shouldnt but it cant bechanged
	// since the following commands need their output
	salt_b := make([]byte, SALT_LENGTH)
	rand.Read(salt_b)
	salt := hex.EncodeToString(salt_b)

	challenge_b := (sha512.Sum512([]byte(salt)))
	challenge := hex.EncodeToString(challenge_b[:])

	// TODO Figure out slots
	response, err := Run2(fmt.Sprintf("ykchalresp -2 -x %s 2>/dev/null", challenge))
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	luksPass := ""
	if encryptionPasswd != "" {
		luksPass, err = Run2(fmt.Sprintf("echo -n '%s' | pbkdf2-sha512 %d %d '%s'", encryptionPasswd, KEYLENGTH/8, ITERATIONS, response))
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	} else {
		luksPass, err = Run2(fmt.Sprintf("echo '' | pbkdf2-sha512 %d %d '%s'", KEYLENGTH/8, ITERATIONS, response))
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}

	// TODO Sometimes crashes
	cmds = append(cmds, ShellCommand{
		Label: "Format Cryptsetup",
		Cmd: fmt.Sprintf(
			`echo -n "%s" | cryptsetup luksFormat --cipher="%s" --key-size="%d" --hash="%s" --key-file=- "/dev/%s"`,
			luksPass,
			CIPHER,
			KEYLENGTH,
			HASH,
			p.Path,
		),
	})

	cmds = append(cmds, ShellCommand{
		Label: "Open Luks",
		Cmd: fmt.Sprintf(
			`echo -n "%s" | cryptsetup luksOpen /dev/%s %s --key-file=-`,
			luksPass,
			p.Path,
			p.Label,
		),
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

	cmds = append(cmds, Sleep(2))

	cmds = append(cmds, ShellCommand{
		Label: "Create /root/boot",
		Cmd:   "mkdir -p /root/boot",
	})

	cmds = append(cmds, ShellCommand{
		Label: fmt.Sprintf("Mounting %s to /root/boot", BOOTLABEL),
		Cmd:   fmt.Sprintf("mount /dev/disk/by-label/%s /root/boot", BOOTLABEL),
	})

	cmds = append(cmds, Sleep(2))

	cmds = append(cmds, ShellCommand{
		Label: "Generate cryptstore Dir",
		Cmd:   "mkdir -p /root/boot/crypt-storage",
	})

	cmds = append(cmds, ShellCommand{
		Label: "Write into Cryptstore",
		Cmd:   fmt.Sprintf(`echo -ne "%s\n%d" > /root/boot/crypt-storage/default`, salt, ITERATIONS),
	})

	// TODO doesnt work
	cmds = append(cmds, ShellCommand{
		Label: "Unmounting /root/boot",
		Cmd:   "umount /root/boot",
	})

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

func FormatDiskLegacy(sel selection.Selection) (s selection.Selection, cmds []Command) {

	sel.Disk.Table = disk.Mbr

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
	sel.Disk.Table = disk.Gpt

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
