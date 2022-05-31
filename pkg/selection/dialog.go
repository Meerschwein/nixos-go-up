package selection

import (
	"fmt"

	"github.com/Meerschwein/nixos-go-up/pkg/util"
	"github.com/manifoldco/promptui"
)

func SecretDialog(label string) (secret string) {
	prompt := promptui.Prompt{
		HideEntered: true,
		Mask:        '*',
	}

	var err error
	check := "*"
	for secret != check {
		prompt.Label = "Choose " + label
		secret, err = prompt.Run()
		util.ExitIfErr(err)

		prompt.Label = "Repeat " + label
		check, err = prompt.Run()
		util.ExitIfErr(err)

		if secret != check {
			fmt.Println("Secrets don't match! Try again!")
		}
	}

	return
}

func ConfirmationDialog(label string) (success bool) {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}

	_, err := prompt.Run()

	success = err == nil

	return
}

func YesNoDialog(label string) (success bool) {
	prompt := promptui.Select{
		Label: label,
		Items: []string{"Yes", "No"},
		Size:  2,
	}

	i, _, err := prompt.Run()
	util.ExitIfErr(err)

	success = i == 0

	return
}
