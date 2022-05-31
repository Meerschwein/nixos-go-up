package command

import (
	"fmt"

	"github.com/Meerschwein/nixos-go-up/pkg/util"
)

const (
	BOOTLABEL = "NIXBOOT"
	ROOTLABEL = "NIXROOT"
)

type Command interface {
	Message() string
	Execute(map[string]string) (key string, val string, err error)
	DryRun() string
}

func DryRun(cmds []Command) {
	for _, cmd := range cmds {
		fmt.Printf("--\n%s\n%s\n", cmd.Message(), cmd.DryRun())
	}
}

func RunCmds(cmds []Command) {
	state := make(map[string]string)
	for _, cmd := range cmds {
		fmt.Printf("-----\n%s\n", cmd.Message())
		key, val, err := cmd.Execute(state)
		if val != "" {
			fmt.Println(val)
		}
		if key != "" {
			state[key] = val
		}
		util.ExitIfErr(err)
	}
}

func ShellScript(cmds []Command) (script string) {
	script += "#!/usr/bin/env bash\n\n"

	for _, c := range cmds {
		script += fmt.Sprintf(
			"# %s\n%s\n\n",
			c.Message(),
			c.DryRun(),
		)
	}

	return
}
