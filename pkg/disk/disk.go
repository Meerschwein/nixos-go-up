package disk

import (
	"fmt"
	"io/ioutil"
	"log"
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

type Firmware string

var (
	UEFI Firmware = "uefi"
	BIOS Firmware = "bios"
)

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

type Disk struct {
	Name             string
	Vendor           string
	Model            string
	SizeGB           int
	Encrypt          bool
	Yubikey          bool
	EncryptionPasswd string

	PartitionTable PartitionTable
	Partitions     []Partition
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
		return d.Name + strconv.Itoa(partition)
	}
}
