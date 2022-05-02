package command

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Meerschwein/nixos-go-up/pkg/util"
)

type Command interface {
	Message() string
	Execute() (string, error)
	DryRun() string
}

func DryRun(cmds []Command) {
	for _, cmd := range cmds {
		fmt.Printf("--\n%s\n%s\n", cmd.Message(), cmd.DryRun())
	}
}

func Run(cmds []Command) {
	for _, cmd := range cmds {
		fmt.Printf("--\n%s\n", cmd.Message())
		out, err := cmd.Execute()
		if string(out) != "" {
			fmt.Println(out)
		}
		util.ExitIfErr(err)
	}
}

type ShellCommand struct {
	Label string
	Cmd   string
}

func (c ShellCommand) Message() string {
	return c.Label
}

func (c ShellCommand) Execute() (string, error) {
	s := strings.Split(c.Cmd, " ")
	name := s[0]
	args := s[1:]
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

func (c ShellCommand) DryRun() string {
	return c.Cmd
}

type FunctionCommand struct {
	Label string
	Func  func() (success bool)
}

func (c FunctionCommand) Message() string {
	return c.Label
}

func (c FunctionCommand) Execute() (string, error) {
	success := c.Func()
	if !success {
		return "", fmt.Errorf("unsuccessfull")
	}
	return "", nil
}

func (c FunctionCommand) DryRun() string {
	return "Ran a function"
}

type RepeatedFunctionCommand struct {
	Label string
	Func  func() (success bool)
	Limit int
	Wait  time.Duration
}

func (c RepeatedFunctionCommand) Message() string {
	return c.Label
}

func (c RepeatedFunctionCommand) Execute() (msg string, err error) {
	i := 0
	for ; i < c.Limit; i++ {
		if c.Func() {
			msg = "Ran " + strconv.Itoa(i) + " times"
			return
		}
		time.Sleep(c.Wait)
	}
	if i >= c.Limit {
		msg = "Exceeded Limit of " + strconv.Itoa(c.Limit)
	}
	return

}

func (c RepeatedFunctionCommand) DryRun() string {
	return fmt.Sprintf("Ran a function at worst %d times", c.Limit)
}