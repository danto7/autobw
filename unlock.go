package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/danto7/autobw/state"
)

const passwordEnv = "PASSWORD"

func unlock(state *state.State) error {
	password, err := dialog()
	if err != nil {
		return err
	}

	cmd := exec.Command(bwBinary, "--nointeraction", "unlock", "--passwordenv", passwordEnv, "--raw")
	cmd.Env = append(os.Environ(), passwordEnv+"="+password)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bw cli exited '%v' stdout: '%s': %w", cmd, out, err)
	}
	session := strings.TrimSpace(string(out))

	state.BitwardenSession = session
	state.LastUnlock = time.Now()
	state.UnlockTimeout = 30 * time.Minute

	if err := state.Write(); err != nil {
		return fmt.Errorf("error updating secret: %w", err)
	}

	return nil
}
