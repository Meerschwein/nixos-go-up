package disk

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/Meerschwein/nixos-go-up/pkg/util"
)

type Filesystem string

var (
	Ext4  Filesystem = "ext4"
	Fat32 Filesystem = "fat32"
)

type PartitionTable string

var (
	Gpt PartitionTable = "gpt"
	Mbr PartitionTable = "mbr"
)

type Disk struct {
	Name             string
	SizeGB           int
	Encrypt          bool
	Yubikey          bool
	EncryptionPasswd string

	PartitionTable PartitionTable
	Partitions     []Partition
}

type Partition struct {
	Format   Filesystem
	Label    string
	Primary  bool
	Path     string
	Number   int
	From     string
	To       string
	Bootable bool
}

func (d Disk) WithSize() Disk {
	_size := util.GetFirstLineOfFile("/sys/block/" + d.Name + "/size")
	if _size == "" {
		return d
	}

	size, _ := strconv.ParseFloat(_size, 32)

	d.SizeGB = int(size / 1024 / 1024 / 2)

	return d
}

func GetDisks() (disks []Disk) {
	devices, err := ioutil.ReadDir("/sys/block")
	util.ExitIfErr(err)

	for _, device := range devices {
		if strings.HasPrefix(device.Name(), "loop") {
			continue
		}

		d := Disk{Name: device.Name()}

		disks = append(disks, d.WithSize())
	}

	return
}

func DisplayDisks(disks []Disk) (display []string) {
	longestNameLength := 0
	for _, d := range disks {
		if len(d.Name) > longestNameLength {
			longestNameLength = len(d.Name)
		}
	}

	for _, d := range disks {
		display = append(display,
			fmt.Sprintf("%s%s %dGb",
				d.Name,
				strings.Repeat(" ", longestNameLength-len(d.Name)),
				d.SizeGB,
			),
		)
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
		return d.Name + strconv.Itoa(partition)
	}
}
