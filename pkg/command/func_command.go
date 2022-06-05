package command

import "fmt"

type FuncCommand struct {
	Label    string
	Cmd      func(state map[string]string) (val string, err error)
	OutLabel string
}

func (f FuncCommand) Message() string {
	return f.Label
}

func (f FuncCommand) Execute(state map[string]string) (key string, val string, err error) {
	key = f.OutLabel
	val, err = f.Cmd(state)
	return
}

func (f FuncCommand) ToShellCommand() string {
	_, res, _ := f.Execute(map[string]string{})
	return fmt.Sprintf("Result %s = %s", f.OutLabel, res)
}