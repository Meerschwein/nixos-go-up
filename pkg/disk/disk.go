package disk

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	"github.com/Meerschwein/nixos-go-up/pkg/command"
	"github.com/Meerschwein/nixos-go-up/pkg/util"
)

type DiskFormat string

var (
	Ext4  DiskFormat = "ext4"
	Fat32 DiskFormat = "fat32"
)

type Table string

var (
	Gpt Table = "gpt"
	Mbr Table = "mbr"
)

type Partition struct {
	Format   DiskFormat
	Label    string
	Primary  bool
	Path     string
	Number   int
	From     string
	To       string
	Bootable bool
}

type Disk struct {
	Name   string
	Vendor string
	Model  string
	SizeGB int

	Table      Table
	Partitions []Partition
}

func (d Disk) WithSize() Disk {
	sizeKB := util.GetFirstLineOfFile("/sys/block/" + d.Name + "/size")
	if sizeKB == "" {
		return d
	}

	_sizeKB, _ := strconv.ParseFloat(sizeKB, 32)

	d.SizeGB = int(_sizeKB / 1024 / 1024)

	return d
}

func GetDisks() (disks []Disk) {
	files, err := ioutil.ReadDir("/sys/block")
	if err != nil {
		log.Fatal(err)
	}

	base := "/sys/block/%s/device/%s"
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "loop") {
			continue
		}

		d := Disk{
			Name:   file.Name(),
			Vendor: util.GetFirstLineOfFile(fmt.Sprintf(base, file.Name(), "vendor")),
			Model:  util.GetFirstLineOfFile(fmt.Sprintf(base, file.Name(), "model")),
		}

		d = d.WithSize()

		disks = append(disks, d)
	}

	return
}

func DisplayDisks(disks []Disk) (display []string) {
	for _, disk := range disks {
		display = append(display, fmt.Sprintf(
			"%s\t%d Gb total",
			disk.Name,
			disk.SizeGB,
		))
	}

	return
}

func (d Disk) PartitionName(partition int) string {
	switch {
	case strings.HasPrefix(d.Name, "sd"):
		return d.Name + strconv.Itoa(partition)
	case strings.HasPrefix(d.Name, "nvme"):
		return d.Name + "p" + strconv.Itoa(partition)
	default:
		fmt.Printf("Warning unrecognised disk type: %s guessing %s\n",
			d.Name,
			d.Name+strconv.Itoa(partition),
		)
		return d.Name + strconv.Itoa(partition)
	}
}

func (d Disk) TableCommands() (cmds []command.Command, err error) {
	var cmd command.ShellCommand
	switch d.Table {
	case Mbr:
		cmd = command.ShellCommand{
			Label: fmt.Sprintf("Formatting %s to MBR", d.Name),
			Cmd:   "parted -s /dev/" + d.Name + " -- mklabel msdos",
		}
	case Gpt:
		cmd = command.ShellCommand{
			Label: fmt.Sprintf("Formatting %s to GPT", d.Name),
			Cmd:   "parted -s /dev/" + d.Name + " -- mklabel gpt",
		}
	default:
		err = fmt.Errorf("unrecognized partitioning scheme %s! Aborting... ", d.Table)
		return
	}
	cmds = append(cmds, cmd)

	return
}

func MakeFilesystemCommand(p Partition) (cmds []command.Command, err error) {
	cmd := command.ShellCommand{
		Label: fmt.Sprintf("Partition /dev/%s to %s", p.Path, p.Format),
	}

	switch p.Format {
	case Ext4:
		cmd.Cmd = fmt.Sprintf("mkfs.ext4 -L %s /dev/%s", p.Label, p.Path)
	case Fat32:
		cmd.Cmd = fmt.Sprintf("mkfs.fat -F32 -n %s /dev/%s", p.Label, p.Path)
	default:
		err = fmt.Errorf("unrecognized filesystem %s! Aborting... ", p.Format)
	}

	cmds = append(cmds, cmd)

	return
}

type BootForm string

var (
	UEFI BootForm = "uefi"
	BIOS BootForm = "bios"
)

func (d Disk) Commands(boot BootForm) (cmds []command.Command, err error) {
	newCmds, err := d.TableCommands()
	if err != nil {
		return
	}
	cmds = append(cmds, newCmds...)

	for _, p := range d.Partitions {
		partType := "extended"
		if p.Primary {
			partType = "primary"
		}

		if p.Bootable && boot == UEFI {
			partType = "ESP"
		}

		fsType := ""
		if p.Format == Fat32 {
			fsType = "fat32"
		}

		cmds = append(cmds, command.ShellCommand{
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
			cmds = append(cmds, command.ShellCommand{
				Label: fmt.Sprintf("Set partition %d bootable", p.Number),
				Cmd:   fmt.Sprintf("parted -s /dev/%s -- set %d esp on", d.Name, p.Number),
			})
		}
	}

	for _, p := range d.Partitions {
		newCmds, err = MakeFilesystemCommand(p)
		if err != nil {
			return
		}
		cmds = append(cmds, newCmds...)
	}

	return
}
