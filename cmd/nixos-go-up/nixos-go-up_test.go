package main

import (
	"testing"

	"github.com/Meerschwein/nixos-go-up/pkg/selection"
	"github.com/Meerschwein/nixos-go-up/test/generators"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func Test_Functions_Generate_Same_Output(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		sel1 := generators.SelectionGen().Draw(t, "Selection 1").(selection.Selection)
		sel2 := generators.SelectionGen().Filter(func(sel selection.Selection) bool {
			return sel.Hostname != sel1.Hostname
		}).Draw(t, "Selection 2").(selection.Selection)

		require.Equal(t, UefiMountBootDir(sel1), UefiMountBootDir(sel2), "UefiMountBootDir produces different commands")
		require.Equal(t, MountRootToMnt(sel1), MountRootToMnt(sel2), "MountRootToMnt produces different commands")
		require.Equal(t, NixosInstall(sel1), NixosInstall(sel2), "NixosInstall produces different commands")
	})
}
