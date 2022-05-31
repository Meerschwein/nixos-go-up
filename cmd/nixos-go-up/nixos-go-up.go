package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Meerschwein/nixos-go-up/pkg/command"
	"github.com/Meerschwein/nixos-go-up/pkg/configuration"
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

	flag.Parse()

	if !util.WasRunAsRoot() {
		util.ExitIfErr(fmt.Errorf("run as root"))
	}

	if util.MountIsUsed() && !dryRun {
		util.ExitIfErr(fmt.Errorf("something is was found at /mnt"))
	}
}

func main() {
	conf := configuration.Conf{}
	conf = conf.SetFirmware()
	conf.NetInterfaces = util.GetInterfaces()

	selectionSteps := []selection.SelectionStep{
		selection.Disk,
	}

	// BIOS encryption is not supported at the moment
	if conf.IsUEFI() {
		selectionSteps = append(selectionSteps, selection.DiskEncryption)
	}

	selectionSteps = append(selectionSteps,
		selection.Hostname,
		selection.Timezone,
		selection.DesktopEnviroment,
		selection.Keyboardlayout,
		selection.Username,
		selection.Password,
	)

	conf, err := selection.GetSelections(conf, selectionSteps)
	util.ExitIfErr(err)

	fmt.Printf("Your Selection so far:\n%v\n", conf)

	cont := selection.ConfirmationDialog("Are you sure you want to continue?")
	if !cont {
		fmt.Println("Aborting...")
		return
	}

	gens := command.MakeCommandGenerators(conf)

	cmds := command.GenerateCommands(conf, gens)

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
