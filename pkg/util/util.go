package util

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func EscapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}

func ExitIfErr(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func GetFirstLineOfFile(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}

	return ""
}

func DoesDirExist(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func GetInterfaces() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	res := []string{}
	for _, inter := range interfaces {
		if !strings.Contains(inter.Flags.String(), "loopback") &&
			!strings.Contains(inter.Flags.String(), "pointtopoint") {
			res = append(res, inter.Name)
		}
	}
	return res, nil
}

func IsUefiSystem() bool {
	return DoesDirExist("/sys/firmware/efi/")
}

func MountIsUsed() bool {
	return exec.Command("mountpoint", "/mnt").Run() == nil
}

func WasRunAsRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		fmt.Printf("Unable to get current user: %s\n", err)
	}
	return currentUser.Username == "root"
}
