package main

import (
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/danto7/autobw/state"
)

var (
	version  = "dev"
	commit   = "none"
	date     = "unknown"
	debug    = version == "dev"
	bwBinary = "bw"
)

func main() {
	lvl := new(slog.LevelVar)
	if debug {
		lvl.Set(slog.LevelDebug)
	} else {
		lvl.Set(slog.LevelInfo)
	}
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(l)

	var s state.State
	err := s.Load()
	if err == state.ErrorNotFound {
		slog.Debug("No session found")
	} else if err != nil {
		slog.Error("Error getting secret", "err", err.Error())
		panic(err)
	}

	if time.Now().Sub(s.LastUnlock) > s.UnlockTimeout {
		switch confirmIdentity() {
		case ErrorAuthenticationFailed:
			slog.Error("Authentication failed")
			return
		case ErrorAuthenticationTimedOut:
			slog.Error("Authentication timed out")
			return
		}

		slog.Debug("Identity confirmed, updating lastUnlock in state")
		s.LastUnlock = time.Now()
		if err := s.Write(); err != nil {
			slog.Error("Error updating state")
			panic(err)
		}
	}

	args := os.Args[1:]
	if isUnlocked(s.BitwardenSession) {
		slog.Debug("Already unlocked", "bw args", args)
		run(args, s.BitwardenSession)
		return
	}

	status, err := status(s.BitwardenSession)
	if err != nil {
		slog.Error("Error getting status", "err", err.Error())
		panic(err)
	}
	slog.Debug("Bitwarden status", "status", status.Status, "lastSync", status.LastSync, "serverUrl", status.ServerUrl, "userId", status.UserId)
	switch status.Status {
	case "locked":
		if err := unlock(&s); errors.Is(err, ErrorCanceledByUser) {
			slog.Error("Unlock canceled by user")
			os.Exit(1)
		} else if err != nil {
			slog.Error("Error unlocking", "err", err.Error())
			os.Exit(1)
		}
		slog.Debug("Unlock successfull", "bw args", args)
		run(args, s.BitwardenSession)
	default:
		// TODO: implement info if is not logged in
		slog.Error("Unknown status", "status", status.Status)
		os.Exit(1)
	}
}

func isUnlocked(session string) bool {
	cmd := exec.Command(bwBinary, "unlock", "--check")
	cmd.Env = append(os.Environ(), "BW_SESSION="+session)
	err := cmd.Run()
	if _, ok := err.(*exec.ExitError); ok {
		return false
	} else if err != nil {
		panic(err)
	}
	return true
}

func run(args []string, session string) {
	cmd := exec.Command(bwBinary, "--nointeraction")
	cmd.Args = append(cmd.Args, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = append(os.Environ(), "BW_SESSION="+session)
	err := cmd.Run()
	if exitError, ok := err.(*exec.ExitError); ok {
		slog.Debug("bw exited with non zero exit code", "exit code", exitError.ExitCode())
		os.Exit(exitError.ExitCode())
	} else if err != nil {
		slog.Error("Error running bw", "err", err.Error())
		os.Exit(1)
	}
}
