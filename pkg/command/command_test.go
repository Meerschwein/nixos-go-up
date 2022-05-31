package command_test

import (
	"testing"

	"github.com/Meerschwein/nixos-go-up/pkg/command"
	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/test/generators"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestDisk_TableCommands_Properties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d1 := generators.DiskGen().Draw(t, "Disk").(disk.Disk)
		d2 := generators.DiskGen().Filter(func(d disk.Disk) bool {
			return d.PartitionTable != d1.PartitionTable
		}).Draw(t, "Disk").(disk.Disk)

		cmds := command.PartitioningTableCommand(d1)
		cmds2 := command.PartitioningTableCommand(d2)

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

		cmds := command.FormatPartition(part)
		cmds2 := command.FormatPartition(part2)

		require.NotEmpty(t, cmds, "Didn't get any commands")
		require.NotEqual(t, cmds, cmds2, "Different Formats got the same commands")
	})
}
