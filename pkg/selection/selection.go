package selection

import (
	"fmt"
	"regexp"
	"time"

	"github.com/Meerschwein/nixos-go-up/pkg/configuration"
	"github.com/Meerschwein/nixos-go-up/pkg/disk"
	"github.com/manifoldco/promptui"
)

type SelectionStep func(before configuration.Conf) (after configuration.Conf, err error)

func Disk(conf configuration.Conf) (configuration.Conf, error) {
	disks := disk.GetDisks()

	prompt := promptui.Select{
		Label: "Select disk to install NixOS to",
		Items: disk.DisplayDisks(disks),
		Size:  len(disks),
	}

	i, _, err := prompt.Run()
	if err != nil {
		return configuration.Conf{}, SelectionStepError("Select Disk", err)
	}

	conf.Disk = disks[i]

	return conf, nil
}

func DiskEncryption(conf configuration.Conf) (configuration.Conf, error) {
	conf.Disk.Encrypt = YesNoDialog(fmt.Sprintf("Encrypt disk %s?", conf.Disk.Name))
	if !conf.Disk.Encrypt {
		return conf, nil
	}

	conf.Disk.EncryptionPasswd = SecretDialog("Encryption Password")
	conf.Yubikey = YesNoDialog("Do you want to use a Yubikey for Encryption?")
	if conf.Yubikey {
		fmt.Println("Warning! You must have the Yubikey plugged and a slot configured for challenge response!")
		prompt := promptui.Select{
			Label:     "Select the challenge response slot",
			Items:     []string{"1", "2"},
			Size:      2,
			CursorPos: 1,
		}
		i, _, err := prompt.Run()
		if err != nil {
			return configuration.Conf{}, SelectionStepError("Select Yubikey Slot", err)
		}
		conf.YubikeySlot = i + 1
	}

	return conf, nil
}

func Username(conf configuration.Conf) (configuration.Conf, error) {
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
		return configuration.Conf{}, SelectionStepError("Username", err)
	}

	conf.Username = username

	return conf, nil
}

func Password(conf configuration.Conf) (configuration.Conf, error) {
	pass := SecretDialog("Password")

	conf.Password = pass

	return conf, nil
}

func Hostname(conf configuration.Conf) (configuration.Conf, error) {
	// https://wiert.me/2017/08/29/regex-regular-expression-to-match-dns-hostname-or-ip-address-stack-overflow/
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
		return configuration.Conf{}, SelectionStepError("Hostname", err)
	}

	conf.Hostname = hostname

	return conf, nil
}

func Timezone(conf configuration.Conf) (configuration.Conf, error) {
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
		return configuration.Conf{}, SelectionStepError("Timezone", err)
	}

	conf.Timezone = timezone

	return conf, nil
}

func Keyboardlayout(conf configuration.Conf) (configuration.Conf, error) {
	prompt := promptui.Prompt{
		Label:   "Keyboard Layout",
		Default: "de",
	}

	layout, err := prompt.Run()
	if err != nil {
		return configuration.Conf{}, SelectionStepError("Keyboard layout", err)
	}

	conf.KeyboardLayout = layout

	return conf, nil
}

func DesktopEnviroment(conf configuration.Conf) (configuration.Conf, error) {
	dms := configuration.DesktopEnviroments()

	prompt := promptui.Select{
		Label: "Select disk to install NixOS to",
		Items: dms,
		Size:  len(dms),
	}

	i, _, err := prompt.Run()
	if err != nil {
		return configuration.Conf{}, SelectionStepError("Desktop Enviroment", err)
	}

	conf.DesktopEnviroment = dms[i]

	return conf, nil
}

func GetSelections(c configuration.Conf, steps []SelectionStep) (conf configuration.Conf, err error) {
	conf = c
	for _, step := range steps {
		conf, err = step(conf)
		if err != nil {
			return
		}
	}
	return
}

func SelectionStepError(step string, err error) error {
	return fmt.Errorf("an error occured during %s: %s", step, err)
}
