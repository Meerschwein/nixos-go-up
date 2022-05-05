package disk

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

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
	Name             string
	Vendor           string
	Model            string
	SizeGB           int
	Encrypt          bool
	EncryptionPasswd string

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

type Firmware string

var (
	UEFI Firmware = "uefi"
	BIOS Firmware = "bios"
)
