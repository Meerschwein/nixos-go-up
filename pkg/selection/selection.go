package selection

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/manifoldco/promptui"
)

type DesktopEnviroment string

const (
	XFCE  DesktopEnviroment = "xfce"
	GNOME DesktopEnviroment = "gnome"
	NONE  DesktopEnviroment = "none"
)

func getDesktopEnviroments() []DesktopEnviroment {
	return []DesktopEnviroment{XFCE, GNOME, NONE}
}

func NixConfiguration(dm DesktopEnviroment) (config string) {
	switch dm {
	case XFCE:
		config = "services.xserver.desktopManager.xfce.enable = true;\n  " +
			"services.xserver.displayManager.defaultSession = \"xfce\";"
	case GNOME:
		config = "services.xserver.desktopManager.gnome.enable = true;\n  " +
			"services.xserver.displayManager.gdm.enable = true;"
	case NONE:
		config = ""
	default:
		panic("Unknown Desktop Enviroment: " + string(dm) + "!")
	}
	return
}

func ConfirmationDialog(label string) bool {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}

	_, err := prompt.Run()

	return err == nil
}

func SelectDisk(selection Selection) (Selection, error) {
	disks := disk.GetDisks()

	prompt := promptui.Select{
		Label: "Select disk to install NixOS to",
		Items: disk.DisplayDisks(disks),
		Size:  len(disks),
	}

	i, _, err := prompt.Run()

	if err != nil {
		return Selection{}, fmt.Errorf("selecting disk failed: %s", err.Error())
	}

	selection.Disk = disks[i]

	return selection, nil
}

func SelectDiskEncryption(sel Selection) (Selection, error) {
	prompt := promptui.Select{
		Label: "Do you want to encrypt that disk?",
		Items: []string{"Yes", "No"},
		Size:  2,
	}

	i, _, err := prompt.Run()

	if err != nil {
		return Selection{}, fmt.Errorf("selecting disk encryption failed: %s", err.Error())
	}

	if i == 1 { // No
		return sel, nil
	}

	sel.Disk.Encrypt = true

	prompt2 := promptui.Prompt{
		HideEntered: true,
		Mask:        '*',
	}

	pwd1, pwd2 := "a", "b"
	for pwd1 != pwd2 {
		prompt2.Label = "Choose Password"
		pwd1, err = prompt2.Run()
		if err != nil {
			return Selection{}, fmt.Errorf("deciding Encryption Password failed: %s", err.Error())
		}

		prompt2.Label = "Repeat Password"
		pwd2, err = prompt2.Run()
		if err != nil {
			return Selection{}, fmt.Errorf("deciding Encryption Password failed: %s", err.Error())
		}

		if pwd1 != pwd2 {
			fmt.Println("Passwords don't match! Try again!")
		}
	}

	sel.Disk.EncryptionPasswd = pwd1

	prompt = promptui.Select{
		Label: "Do you want to use a Yubikey?",
		Items: []string{"Yes", "No"},
		Size:  2,
	}

	i, _, err = prompt.Run()

	if err != nil {
		return Selection{}, fmt.Errorf("selecting yubikey failed: %s", err.Error())
	}

	if i == 1 { // No
		return sel, nil
	}

	sel.Disk.Yubikey = true

	return sel, nil
}

func SelectUsername(selection Selection) (Selection, error) {
	validUser, _ := regexp.Compile("^[a-z_][a-z0-9_-]*[$]?$")

	prompt := promptui.Prompt{
		Label: "Username",
		Validate: func(s string) error {
			if !validUser.MatchString(s) {
				return fmt.Errorf("invalid Username")
			}
			return nil
		},
	}

	username, err := prompt.Run()
	if err != nil {
		return Selection{}, fmt.Errorf("deciding username failed: %s", err.Error())
	}

	selection.Username = username

	return selection, nil
}

func SelectPassword(selection Selection) (Selection, error) {
	prompt := promptui.Prompt{
		HideEntered: true,
		Mask:        '*',
	}

	var err error
	pwd1, pwd2 := "a", "b"
	for pwd1 != pwd2 {
		prompt.Label = "Choose Password"
		pwd1, err = prompt.Run()
		if err != nil {
			return Selection{}, fmt.Errorf("deciding Password failed: %s", err.Error())
		}

		prompt.Label = "Repeat Password"
		pwd2, err = prompt.Run()
		if err != nil {
			return Selection{}, fmt.Errorf("deciding Password failed: %s", err.Error())
		}

		if pwd1 != pwd2 {
			fmt.Println("Passwords don't match! Try again!")
		}
	}

	selection.Password = pwd1

	return selection, nil
}

func SelectHostname(selection Selection) (Selection, error) {
	validHostname, _ := regexp.Compile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9])$`)

	prompt := promptui.Prompt{
		Label: "Hostname",
		Validate: func(s string) error {
			if !validHostname.MatchString(s) {
				return fmt.Errorf("invalid Username")
			}
			return nil
		},
	}

	hostname, err := prompt.Run()
	if err != nil {
		return Selection{}, fmt.Errorf("deciding hostname failed: %s", err.Error())
	}

	selection.Hostname = hostname

	return selection, nil
}

func SelectTimezone(selection Selection) (Selection, error) {
	prompt := promptui.Prompt{
		Label:   "Timezone",
		Default: "Europe/Berlin",
		Validate: func(s string) error {
			if s == "" || s == "UTC" || s == "Local" {
				return fmt.Errorf("not allowed")
			}
			_, err := time.LoadLocation(s)
			if err != nil {
				return err
			}
			return nil
		},
	}

	timezone, err := prompt.Run()
	if err != nil {
		return Selection{}, fmt.Errorf("deciding timezone failed: %s", err.Error())
	}

	selection.Timezone = timezone

	return selection, nil
}

func SelectKeyboardLayout(selection Selection) (Selection, error) {
	prompt := promptui.Prompt{
		Label:   "Keyboard Layout",
		Default: "us",
	}

	layout, err := prompt.Run()
	if err != nil {
		return Selection{}, fmt.Errorf("deciding Keyboard layout failed: %s", err.Error())
	}

	selection.KeyboardLayout = layout

	return selection, nil
}

func SelectDesktopEnviroment(selection Selection) (Selection, error) {
	dms := getDesktopEnviroments()

	prompt := promptui.Select{
		Label: "Select disk to install NixOS to",
		Items: dms,
		Size:  len(dms),
	}

	i, _, err := prompt.Run()

	if err != nil {
		return Selection{}, fmt.Errorf("selecting disk failed: %s", err.Error())
	}

	selection.DesktopEnviroment = dms[i]

	return selection, nil
}

type SelectionStep func(Selection) (Selection, error)

type Selection struct {
	Disk              disk.Disk
	Hostname          string
	Timezone          string
	Username          string
	Password          string
	DesktopEnviroment DesktopEnviroment
	KeyboardLayout    string
}

func (s Selection) String() (res string) {
	return fmt.Sprintf("Disk:\t\t%s\nEncyrpt disk:\t%v\nHostname:\t%s\nTimezone:\t%s\nDesktop:\t%s\nKeyboard:\t%s\nUsername:\t%s\nPassword:\t%s",
		s.Disk.Name,
		s.Disk.Encrypt,
		s.Hostname,
		s.Timezone,
		s.DesktopEnviroment,
		s.KeyboardLayout,
		s.Username,
		strings.Repeat("*", len(s.Password)),
	)
}

func GetSelections(steps []SelectionStep) (sel Selection, err error) {
	for _, step := range steps {
		sel, err = step(sel)
		if err != nil {
			return
		}
	}
	return
}
