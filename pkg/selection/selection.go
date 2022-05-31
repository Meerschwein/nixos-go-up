package selection

import (
	"fmt"
	"log"
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
		log.Panicf("Unknown Desktop Enviroment: %s!", string(dm))
	}
	return
}

func SecretDialog(label string) (secret string, err error) {
	prompt := promptui.Prompt{
		HideEntered: true,
		Mask:        '*',
	}

	check := "*"
	for secret != check {
		prompt.Label = "Choose " + label
		secret, err = prompt.Run()
		if err != nil {
			return
		}

		prompt.Label = "Repeat " + label
		check, err = prompt.Run()
		if err != nil {
			return
		}

		if secret != check {
			fmt.Println("Secrets don't match! Try again!")
		}
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

func YesNoDialog(label string) (success bool, err error) {
	prompt := promptui.Select{
		Label: label,
		Items: []string{"Yes", "No"},
		Size:  2,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return
	}

	success = i == 0

	return
}

func SelectionStepError(step string, err error) error {
	return fmt.Errorf("an error occured during %s: %s", step, err)
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

func SelectDisk(sel Selection) (Selection, error) {
	disks := disk.GetDisks()

	prompt := promptui.Select{
		Label: "Select disk to install NixOS to",
		Items: disk.DisplayDisks(disks),
		Size:  len(disks),
	}

	i, _, err := prompt.Run()
	if err != nil {
		return Selection{}, SelectionStepError("Select Disk", err)
	}

	sel.Disk = disks[i]

	return sel, nil
}

func SelectDiskEncryption(sel Selection) (Selection, error) {
	encrypt, err := YesNoDialog(fmt.Sprintf("Encrypt disk %s ?", sel.Disk.Name))
	if err != nil {
		return Selection{}, SelectionStepError("Encyrpt Disk", err)
	}
	if !encrypt {
		sel.Disk.Encrypt = false
		return sel, nil
	}

	sel.Disk.Encrypt = true

	pass, err := SecretDialog("Encryption Password")
	if err != nil {
		return Selection{}, SelectionStepError("Disk encrytion password", err)
	}

	sel.Disk.EncryptionPasswd = pass

	useYubikey, err := YesNoDialog("Do you want to use a Yubikey for Encryption?")
	if err != nil {
		return Selection{}, SelectionStepError("Use Yubikey", err)
	}

	sel.Disk.Yubikey = useYubikey

	return sel, nil
}

func SelectUsername(sel Selection) (Selection, error) {
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
		return Selection{}, SelectionStepError("Username", err)
	}

	sel.Username = username

	return sel, nil
}

func SelectPassword(sel Selection) (Selection, error) {
	pass, err := SecretDialog("Password")
	if err != nil {
		return sel, SelectionStepError("User password", err)
	}

	sel.Password = pass

	return sel, nil
}

func SelectHostname(sel Selection) (Selection, error) {
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
		return Selection{}, SelectionStepError("Hostname", err)
	}

	sel.Hostname = hostname

	return sel, nil
}

func SelectTimezone(sel Selection) (Selection, error) {
	prompt := promptui.Prompt{
		Label:   "Timezone",
		Default: "Europe/Berlin",
		Validate: func(s string) error {
			// These are allowed by time.LoadLocation but not in nix
			if s == "" || s == "UTC" || s == "Local" {
				return fmt.Errorf("not allowed")
			}
			_, err := time.LoadLocation(s)
			return err
		},
	}

	timezone, err := prompt.Run()
	if err != nil {
		return Selection{}, SelectionStepError("Timezone", err)
	}

	sel.Timezone = timezone

	return sel, nil
}

func SelectKeyboardLayout(sel Selection) (Selection, error) {
	prompt := promptui.Prompt{
		Label:   "Keyboard Layout",
		Default: "de",
	}

	layout, err := prompt.Run()
	if err != nil {
		return Selection{}, SelectionStepError("Keyboard layout", err)
	}

	sel.KeyboardLayout = layout

	return sel, nil
}

func SelectDesktopEnviroment(sel Selection) (Selection, error) {
	dms := getDesktopEnviroments()

	prompt := promptui.Select{
		Label: "Select disk to install NixOS to",
		Items: dms,
		Size:  len(dms),
	}

	i, _, err := prompt.Run()
	if err != nil {
		return Selection{}, SelectionStepError("Desktop Enviroment", err)
	}

	sel.DesktopEnviroment = dms[i]

	return sel, nil
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
