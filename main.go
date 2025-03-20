package main

import (
	"errors"
	"log/slog"
	"os"
	"os/exec"

	"github.com/keybase/go-keychain"
)

const DEBUG = false

func main() {
	lvl := new(slog.LevelVar)
	if DEBUG {
		lvl.Set(slog.LevelDebug)
	} else {
		lvl.Set(slog.LevelInfo)
	}
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(l)

	secret, err := getSecret()
	if err == keychain.ErrorItemNotFound {
		slog.Debug("No session found")
	} else if err != nil {
		slog.Error("Error getting secret", "err", err.Error())
		panic(err)
	}
	session := string(secret)

	args := os.Args[1:]
	if isUnlocked(session) {
		slog.Debug("Already unlocked", "bw args", args)
		run(args, session)
		return
	}

	status, err := status(session)
	if err != nil {
		slog.Error("Error getting status", "err", err.Error())
		panic(err)
	}
	slog.Debug("Bitwarden status", "status", status.Status, "lastSync", status.LastSync, "serverUrl", status.ServerUrl, "userId", status.UserId)
	switch status.Status {
	case "locked":
		session, err := unlock()
		if errors.Is(err, ErrorCanceledByUser) {
			slog.Error("Unlock canceled by user")
			os.Exit(1)
		} else if err != nil {
			slog.Error("Error unlocking", "err", err.Error())
			os.Exit(1)
		}
		slog.Debug("Unlock successfull", "bw args", args)
		run(args, session)
	default:
		// TODO: implement info if is not logged in
		slog.Error("Unknown status", "status", status.Status)
		os.Exit(1)
	}
}

func isUnlocked(session string) bool {
	cmd := exec.Command("bw", "unlock", "--check")
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
	cmd := exec.Command("bw")
	cmd.Args = append([]string{"--nointeraction"}, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = append(os.Environ(), "BW_SESSION="+session)
	err := cmd.Run()
	if exitError, ok := err.(*exec.ExitError); ok {
		slog.Error("bw not found in path")
		os.Exit(exitError.ExitCode())
	} else if err != nil {
		slog.Error("Error running bw", "err", err.Error())
		os.Exit(1)
	}
}
