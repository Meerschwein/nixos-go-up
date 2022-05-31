package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Meerschwein/nixos-go-up/pkg/command"
	"github.com/Meerschwein/nixos-go-up/pkg/selection"
	"github.com/Meerschwein/nixos-go-up/pkg/util"
)

var (
	dryRun     bool
	toScript   bool
	scriptname string
)

func init() {
	flag.BoolVar(&dryRun, "dry-run", false, "dry-run")
	flag.BoolVar(&toScript, "to-script", false, "to-script")
	flag.StringVar(&scriptname, "script-name", "nixos-install.sh", "script name")
}

func main() {
	flag.Parse()

	if !util.WasRunAsRoot() {
		util.ExitIfErr(fmt.Errorf("run as root"))
	}

	if util.MountIsUsed() && !dryRun {
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

	if dryRun {
		command.DryRun(cmds)
	} else if toScript {
		script := command.ShellScript(cmds)
		err := os.WriteFile(scriptname, []byte(script), 0o644)
		util.ExitIfErr(err)
	} else {
		command.RunCmds(cmds)
	}
}
