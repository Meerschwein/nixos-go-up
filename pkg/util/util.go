package util

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strings"

	"github.com/itsyouonline/identityserver/credentials/password/keyderivation/crypt/sha512crypt"
)

// https://www.gnu.org/software/bash/manual/html_node/Double-Quotes.html
func EscapeBashDoubleQuotes(s string) string {
	replacements := []string{"\\", "$", "`", "\""}

	for _, rep := range replacements {
		s = strings.ReplaceAll(s, rep, "\\"+rep)
	}

	return s
}

func RemoveLinebreaks(s string) string {
	re := regexp.MustCompile(`\x{000D}\x{000A}|[\x{000A}\x{000B}\x{000C}\x{000D}\x{0085}\x{2028}\x{2029}]`)
	return re.ReplaceAllString(s, ``)
}

func ExitIfErr(err error) {
	if err != nil {
		fmt.Println(err.Error())
		if strings.HasSuffix(err.Error(), "^C") {
			fmt.Println("User Interruption")
			os.Exit(0)
		}
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

func GetInterfaces() []string {
	interfaces, err := net.Interfaces()
	ExitIfErr(err)
	res := []string{}
	for _, inter := range interfaces {
		if !strings.Contains(inter.Flags.String(), "loopback") &&
			!strings.Contains(inter.Flags.String(), "pointtopoint") {
			res = append(res, inter.Name)
		}
	}
	return res
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

func MkPasswd(key string) string {
	hash, _ := sha512crypt.New().Generate([]byte(key), []byte{})
	return hash
}
