package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/danto7/autobw/state"
	"github.com/go-errors/errors"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err := start(ctx)
	if err != nil {
		if err, ok := err.(*errors.Error); ok {
			fmt.Println()
			fmt.Println(err.ErrorStack())
			exitError := exec.ExitError{}
			if errors.As(err, &exitError) {
				os.Exit(exitError.ExitCode())
			}
			os.Exit(1)
		} else {
			panic(err)
		}
		panic("unreachable")
	}
}

func start(ctx context.Context) error {
	args := os.Args[1:]

	var bypassFlags = []string{"-h", "--help", "--version", "-v"}
	for _, arg := range args {
		if slices.Contains(bypassFlags, arg) {
			if arg == "-v" || arg == "--version" {
				fmt.Printf("autobw %s, commit %s\n", version, commit)
			}
			return errors.Wrap(run(ctx, args, ""), 0)
		}
	}

	var s state.State
	err := s.Load()
	if err == state.ErrorNotFound {
		slog.Debug("No session found")
	} else if err != nil {
		return errors.Errorf("Error loading state: %w", err)
	}

	if time.Now().Sub(s.LastUnlock) > s.UnlockTimeout {
		switch confirmIdentity(ctx) {
		case ErrorAuthenticationFailed:
			return errors.Errorf("Authentication failed")
		case ErrorAuthenticationTimedOut:
			return errors.Errorf("Authentication timed out")
		}

		slog.Debug("Identity confirmed, updating lastUnlock in state")
		s.LastUnlock = time.Now()
		if err := s.Write(); err != nil {
			return errors.Errorf("Error updating state: %w", err)
		}
	}

	if isUnlocked(s.BitwardenSession) {
		slog.Debug("Already unlocked", "bw args", args)
		return errors.Wrap(run(ctx, args, s.BitwardenSession), 0)
	}

	status, err := status(s.BitwardenSession)
	if err != nil {
		return errors.Errorf("Error getting status: %w", err)
	}
	slog.Debug("Bitwarden status", "status", status.Status, "lastSync", status.LastSync, "serverUrl", status.ServerUrl, "userId", status.UserId)
	switch status.Status {
	case "locked":
		if err := unlock(&s); errors.Is(err, ErrorCanceledByUser) {
			return errors.Errorf("Unlock canceled by user: %w", err)
		} else if err != nil {
			return errors.Errorf("Error unlocking: %w", err)
		}
		slog.Debug("Unlock successfull", "bw args", args)
		return errors.Wrap(run(ctx, args, s.BitwardenSession), 0)
	default:
		// TODO: implement info if is not logged in
		return errors.Errorf("Unknown status: %s", status.Status)
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

func run(ctx context.Context, args []string, session string) error {
	if len(args) > 0 && args[0] == "agent" {
		err := startListener(ctx, session)
		if err != nil {
			return errors.Errorf("failed to start agent: %w", err)
		}
		return nil
	}

	cmd := exec.CommandContext(ctx, bwBinary, "--nointeraction")
	cmd.Args = append(cmd.Args, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = append(os.Environ(), "BW_SESSION="+session)

	err := cmd.Run()
	if exitError, ok := err.(*exec.ExitError); ok {
		return errors.Errorf("bw exited with non zero exit code %d: %w", exitError.ExitCode(), exitError)
	} else if err != nil {
		return errors.Errorf("error running bw: %w", err)
	}
	return nil
}
