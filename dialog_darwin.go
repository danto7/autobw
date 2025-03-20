package main

import (
	"fmt"
	"os/exec"
	"strings"
)

var ErrorCanceledByUser = fmt.Errorf("canceled by user")

func dialog() (string, error) {
	cmd := exec.Command("osascript", "-e", "Tell application \"System Events\" to display dialog \"Enter the bitwarden password:\" with hidden answer default answer \"\"", "-e", "text returned of result")
	stdout, err := cmd.Output()
	if exitError, ok := err.(*exec.ExitError); ok {
		return "", fmt.Errorf("%w: command '%v': %w", ErrorCanceledByUser, cmd, exitError)
	} else if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(stdout)), nil
}
