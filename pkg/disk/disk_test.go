package disk_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

var (
	partitionGen = rapid.Custom(func(t *rapid.T) disk.Partition {
		return disk.Partition{
			Format:   rapid.SampledFrom([]disk.DiskFormat{disk.Ext4, disk.Fat32}).Draw(t, "Partition_Format").(disk.DiskFormat),
			Label:    rapid.String().Draw(t, "Partition_Label").(string),
			Primary:  rapid.Bool().Draw(t, "Partition_Primary").(bool),
			Path:     rapid.String().Draw(t, "Partition_Path").(string),
			Number:   rapid.Int().Draw(t, "Partition_Number").(int),
			From:     rapid.String().Draw(t, "Partition_From").(string),
			To:       rapid.String().Draw(t, "Partition_To").(string),
			Bootable: rapid.Bool().Draw(t, "Partition_Bootable").(bool),
		}
	})

	diskGen = rapid.Custom(func(t *rapid.T) disk.Disk {
		return disk.Disk{
			Name:       rapid.String().Draw(t, "Disk_Name").(string),
			Vendor:     rapid.String().Draw(t, "Disk_Vendor").(string),
			Model:      rapid.String().Draw(t, "Disk_Model").(string),
			SizeGB:     rapid.Int().Draw(t, "Disk_SizeGB").(int),
			Table:      rapid.SampledFrom([]disk.Table{disk.Gpt, disk.Mbr}).Draw(t, "Disk_Table").(disk.Table),
			Partitions: rapid.SliceOfN(partitionGen, 0, 5).Draw(t, "Disk_Partitions").([]disk.Partition),
		}
	})

	bootFormGen = rapid.Custom(func(t *rapid.T) disk.BootForm {
		return rapid.SampledFrom([]disk.BootForm{disk.UEFI, disk.BIOS}).Draw(t, "BootForm").(disk.BootForm)
	})
)

func TestDisk_PartitionName(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d := diskGen.Draw(t, "Disk").(disk.Disk)
		part := rapid.Int().Draw(t, "Partition number").(int)

		res := d.PartitionName(part)

		require.True(t, strings.HasSuffix(res, strconv.Itoa(part)), "Partition number at the end")
		require.True(t, strings.HasPrefix(res, d.Name), "Diskname at the start")
	})
}

func TestDisk_TableCommands(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d1 := diskGen.Draw(t, "Disk").(disk.Disk)
		d2 := diskGen.Filter(func(d disk.Disk) bool {
			return d.Table != d1.Table
		}).Draw(t, "Disk").(disk.Disk)

		cmds := d1.TableCommands()
		cmds2 := d2.TableCommands()

		require.NotEmpty(t, cmds, "Didn't get any commands")
		require.NotEqual(t, cmds, cmds2, "Different Tables got the same commands")
	})
}

func TestDisk_MakeFilesystemCommand(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		part := partitionGen.Draw(t, "Partition").(disk.Partition)
		part2 := partitionGen.Filter(func(p disk.Partition) bool {
			return p.Format != part.Format
		}).Draw(t, "Partition 2").(disk.Partition)

		cmds := disk.MakeFilesystemCommand(part)
		cmds2 := disk.MakeFilesystemCommand(part2)

		require.NotEmpty(t, cmds, "Didn't get any commands")
		require.NotEqual(t, cmds, cmds2, "Different Formats got the same commands")
	})
}

func TestDisk_Commands(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d := diskGen.Draw(t, "Disk").(disk.Disk)

		bf1 := bootFormGen.Draw(t, "BootForm 1").(disk.BootForm)

		cmds := d.Commands(bf1)

		require.NotEmpty(t, cmds, "Didn't get any commands")
	})
}
