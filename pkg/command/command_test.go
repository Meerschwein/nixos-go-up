package command_test

import (
	"testing"

	"github.com/Meerschwein/nixos-go-up/pkg/command"
	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/Meerschwein/nixos-go-up/pkg/selection"
	"github.com/Meerschwein/nixos-go-up/test/generators"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestDisk_TableCommands_Properties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d1 := generators.DiskGen().Draw(t, "Disk").(disk.Disk)
		d2 := generators.DiskGen().Filter(func(d disk.Disk) bool {
			return d.Table != d1.Table
		}).Draw(t, "Disk").(disk.Disk)

		cmds := command.TableCommands(d1)
		cmds2 := command.TableCommands(d2)

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

		cmds := command.MakeFilesystemCommand(part)
		cmds2 := command.MakeFilesystemCommand(part2)

		require.NotEmpty(t, cmds, "Didn't get any commands")
		require.NotEqual(t, cmds, cmds2, "Different Formats got the same commands")
	})
}

func TestDisk_Commands_Properties(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d := generators.DiskGen().Draw(t, "Disk").(disk.Disk)

		bf1 := generators.FirmwareGen().Draw(t, "Firmware").(disk.Firmware)

		cmds := command.Commands(d, bf1)

		require.NotEmpty(t, cmds, "Didn't get any commands")
	})
}

func Test_Functions_Generate_Same_Output(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		sel1 := generators.SelectionGen().Draw(t, "Selection 1").(selection.Selection)
		sel2 := generators.SelectionGen().Filter(func(sel selection.Selection) bool {
			return sel.Hostname != sel1.Hostname
		}).Draw(t, "Selection 2").(selection.Selection)

		funcs := []func(selection.Selection) (selection.Selection, []command.Command){
			command.UefiMountBootDir,
			command.MountRootToMnt,
			command.NixosInstall,
		}

		for _, f := range funcs {
			outSel1, outCmds1 := f(sel1)
			outSel2, outCmds2 := f(sel2)

			require.Equal(t, sel1, outSel1, "Same Selections 1")
			require.Equal(t, sel2, outSel2, "Same Selections 2")
			require.Equal(t, outCmds1, outCmds2, "Same Commands")
		}
	})
}
