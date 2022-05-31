package command

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type ShellCommand struct {
	Label    string
	Cmd      string
	OutLabel string
}

func (c ShellCommand) Message() string {
	return c.Label
}

func (c ShellCommand) Execute(state map[string]string) (key string, val string, err error) {
	fmt.Println(c.Cmd)

	for k, v := range state {
		c.Cmd = strings.ReplaceAll(c.Cmd, "$"+k, v)
	}

	var out bytes.Buffer
	exec := exec.Command("bash", "-c", c.Cmd)

	exec.Stdout = io.MultiWriter(os.Stdout, &out)
	exec.Stderr = io.MultiWriter(os.Stderr, &out)

	err = exec.Run()
	val = out.String()
	key = c.OutLabel

	return
}

func (c ShellCommand) DryRun() (cmd string) {
	cmd = c.Cmd
	if c.OutLabel != "" {
		cmd = c.OutLabel + "=$(" + cmd + ")"
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
