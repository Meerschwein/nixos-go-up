package disk_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/test/generators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

var (
	bootFormGen = rapid.Custom(func(t *rapid.T) disk.BootForm {
		return rapid.SampledFrom([]disk.BootForm{disk.UEFI, disk.BIOS}).Draw(t, "BootForm").(disk.BootForm)
	})
)

func TestDisk_PartitionName_Properties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d := generators.DiskGen().Draw(t, "Disk").(disk.Disk)
		part := rapid.Int().Draw(t, "Partition number").(int)

		res := d.PartitionName(part)

		require.True(t, strings.HasSuffix(res, strconv.Itoa(part)), "Partition number at the end")
		require.True(t, strings.HasPrefix(res, d.Name), "Diskname at the start")
	})
}

func TestDisk_PartitionName_Unit(t *testing.T) {
	testcases := []struct {
		Name      string
		Partition int
		Expected  string
	}{
		{
			Name:      "sda",
			Partition: 1,
			Expected:  "sda1",
		},
		{
			Name:      "sd",
			Partition: 2,
			Expected:  "sd2",
		},
		{
			Name:      "sdb",
			Partition: 3,
			Expected:  "sdb3",
		},
		{
			Name:      "nvme0n1",
			Partition: 1,
			Expected:  "nvme0n1p1",
		},
		{
			Name:      "nvme1n4",
			Partition: 2,
			Expected:  "nvme1n4p2",
		},
		{
			Name:      "nvme",
			Partition: 5,
			Expected:  "nvmep5",
		},
		{
			Name:      "test",
			Partition: 1,
			Expected:  "test1",
		},
		{
			Name:      "blahblah",
			Partition: 5,
			Expected:  "blahblah5",
		},
	}

	for _, test := range testcases {
		d := disk.Disk{Name: test.Name}

		actual := d.PartitionName(test.Partition)

		assert.Equal(t, test.Expected, actual)
	}
}

func TestDisk_TableCommands_Properties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d1 := generators.DiskGen().Draw(t, "Disk").(disk.Disk)
		d2 := generators.DiskGen().Filter(func(d disk.Disk) bool {
			return d.Table != d1.Table
		}).Draw(t, "Disk").(disk.Disk)

		cmds := d1.TableCommands()
		cmds2 := d2.TableCommands()

		require.NotEmpty(t, cmds, "Didn't get any commands")
		require.NotEqual(t, cmds, cmds2, "Different Tables got the same commands")
	})
}

func TestDisk_MakeFilesystemCommand_Properties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		part := generators.PartitionGen().Draw(t, "Partition").(disk.Partition)
		part2 := generators.PartitionGen().Filter(func(p disk.Partition) bool {
			return p.Format != part.Format
		}).Draw(t, "Partition 2").(disk.Partition)

		cmds := disk.MakeFilesystemCommand(part)
		cmds2 := disk.MakeFilesystemCommand(part2)

		require.NotEmpty(t, cmds, "Didn't get any commands")
		require.NotEqual(t, cmds, cmds2, "Different Formats got the same commands")
	})
}

func TestDisk_Commands_Properties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d := generators.DiskGen().Draw(t, "Disk").(disk.Disk)

		bf1 := bootFormGen.Draw(t, "BootForm").(disk.BootForm)

		cmds := d.Commands(bf1)

		require.NotEmpty(t, cmds, "Didn't get any commands")
	})
}
