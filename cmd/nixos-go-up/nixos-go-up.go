package main

import (
	"flag"
	"fmt"

	"github.com/Meerschwein/nixos-go-up/pkg/command"
	"github.com/Meerschwein/nixos-go-up/pkg/selection"
	"github.com/Meerschwein/nixos-go-up/pkg/util"
	"github.com/Meerschwein/nixos-go-up/pkg/vars"
)

func init() {
	flag.BoolVar(&vars.DryRun, "dry-run", false, "dry-run")
}

func main() {
	flag.Parse()

	if !util.WasRunAsRoot() {
		fmt.Println("Please run as root!")
		return
	}

	if util.MountIsUsed() && !vars.DryRun {
		util.ExitIfErr(fmt.Errorf("something is was found at /mnt"))
	}

	selectionSteps := []selection.SelectionStep{
		selection.SelectDisk,
	}

	// BIOS encryption is not supported at the moment
	if util.IsUefiSystem() {
		selectionSteps = append(selectionSteps, selection.SelectDiskEncryption)
	}

	selectionSteps = append(selectionSteps,
		selection.SelectHostname,
		selection.SelectTimezone,
		selection.SelectDesktopEnviroment,
		selection.SelectKeyboardLayout,
		selection.SelectUsername,
		selection.SelectPassword,
	)

	sel, err := selection.GetSelections(selectionSteps)
	util.ExitIfErr(err)

	fmt.Printf("Your Selection so far:\n%v\n", sel)

	cont := selection.ConfirmationDialog("Are you sure you want to continue?")
	if !cont {
		fmt.Println("Aborting...")
		return
	}

	gens := command.MakeCommandGenerators(sel)

	cmds := command.GenerateCommands(sel, gens)

	if vars.DryRun {
		command.DryRun(cmds)
	} else {
		command.RunCmds(cmds)
	}
}
