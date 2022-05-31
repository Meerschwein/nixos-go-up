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

func SelectDisk(conf configuration.Conf) (configuration.Conf, error) {
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

func SelectDiskEncryption(conf configuration.Conf) (configuration.Conf, error) {
	conf.Disk.Encrypt = YesNoDialog(fmt.Sprintf("Encrypt disk %s?", conf.Disk.Name))
	if !conf.Disk.Encrypt {
		return conf, nil
	}

	conf.Disk.EncryptionPasswd = SecretDialog("Encryption Password")
	conf.Disk.Yubikey = YesNoDialog("Do you want to use a Yubikey for Encryption?")

	return conf, nil
}

func SelectUsername(conf configuration.Conf) (configuration.Conf, error) {
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

func SelectPassword(conf configuration.Conf) (configuration.Conf, error) {
	pass := SecretDialog("Password")

	conf.Password = pass

	return conf, nil
}

func SelectHostname(conf configuration.Conf) (configuration.Conf, error) {
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

func SelectTimezone(conf configuration.Conf) (configuration.Conf, error) {
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

func SelectKeyboardLayout(conf configuration.Conf) (configuration.Conf, error) {
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

func SelectDesktopEnviroment(conf configuration.Conf) (configuration.Conf, error) {
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
