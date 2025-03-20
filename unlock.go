package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const passwordEnv = "PASSWORD"

func unlock() (string, error) {
	password, err := dialog()
	if err != nil {
		return "", err
	}

	cmd := exec.Command("bw", "--nointeraction", "unlock", "--passwordenv", passwordEnv, "--raw")
	cmd.Env = append(os.Environ(), passwordEnv+"="+password)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("bw cli exited '%v' stdout: '%s': %w", cmd, out, err)
	}
	session := strings.TrimSpace(string(out))

	if err := updateSecret(out); err != nil {
		return "", fmt.Errorf("error updating secret: %w", err)
	}

	return session, nil
}
