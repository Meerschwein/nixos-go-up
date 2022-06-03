package command

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/Meerschwein/nixos-go-up/pkg/util"
)

type ShellCommand struct {
	Label             string
	Cmd               string
	OutLabel          string
	InputPreprocessor func(string) string
}

func (c ShellCommand) Message() string {
	return c.Label
}

func (c ShellCommand) Execute(state map[string]string) (key string, val string, err error) {
	for k, v := range state {
		if c.InputPreprocessor != nil {
			v = c.InputPreprocessor(v)
		}
		c.Cmd = strings.ReplaceAll(c.Cmd, "$"+k, v)
	}

	fmt.Println(c.Cmd)

	var out bytes.Buffer
	exec := exec.Command("bash", "-c", c.Cmd)

	exec.Stdout = io.MultiWriter(os.Stdout, &out)
	exec.Stderr = io.MultiWriter(os.Stderr, &out)

	err = exec.Run()
	val = out.String()
	key = c.OutLabel

	return
}

func (c ShellCommand) ToShellCommand() (cmd string) {
	cmd = c.Cmd
	if c.OutLabel != "" {
		cmd = fmt.Sprintf("%s=$(%s)", c.OutLabel, c.Cmd)
	}
	return
}

func Sleep(secs int) Command {
	return ShellCommand{
		Label: fmt.Sprintf("Sleep for %d seconds", secs),
		Cmd:   fmt.Sprintf("sleep %ds", secs),
	}
}

func CreateDir(dir string) Command {
	return ShellCommand{
		Label: fmt.Sprintf("Create %s if it doesn't already exist", dir),
		Cmd:   fmt.Sprintf("mkdir -p %s", dir),
	}
}

func MountDir(from, to string) []Command {
	return []Command{
		CreateDir(to),
		ShellCommand{
			Label: fmt.Sprintf("Mounting %s to %s", from, to),
			Cmd:   fmt.Sprintf("mount %s %s", from, to),
		},
	}
}

func MountByLabel(label, to string) []Command {
	return []Command{
		CreateDir(to),
		ShellCommand{
			Label: fmt.Sprintf("Mounting %s to %s", label, to),
			Cmd:   fmt.Sprintf("mount -L %s %s", label, to),
		},
	}
}

func Unmount(dir string) Command {
	return ShellCommand{
		Label: "Unmounting " + dir,
		Cmd:   "umount " + dir,
	}
}

func WriteToFile(label, s, file string) Command {
	return ShellCommand{
		Label: label,
		Cmd:   fmt.Sprintf(`echo "%s" > %s`, util.EscapeBashDoubleQuotes(s), file),
	}
}

func AppendToFile(label, s, file string) Command {
	return ShellCommand{
		Label: label,
		Cmd:   fmt.Sprintf(`echo "%s" >> %s`, util.EscapeBashDoubleQuotes(s), file),
	}
}
