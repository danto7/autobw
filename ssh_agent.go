package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"

	"github.com/go-errors/errors"
	"golang.org/x/crypto/ssh"
	sshagent "golang.org/x/crypto/ssh/agent"
)

type BitwardenItemType int

const (
	SshKeyItemType BitwardenItemType = 5
)

type Item struct {
	Name   string            `json:"name"`
	Type   BitwardenItemType `json:"type"`
	SshKey SshKey            `json:"sshKey"`
}

type SshKey struct {
	Fingerprint string `json:"keyFingerprint"`
	Public      string `json:"publicKey"`
	Private     string `json:"privateKey"`
}

func startListener(ctx context.Context, session string) error {
	cmd := exec.Command(bwBinary, "list", "items")
	cmd.Env = append(os.Environ(), "BW_SESSION="+session)
	out, err := cmd.Output()
	if err != nil {
		return errors.Errorf("failed to list items: %w", err)
	}
	var items []Item
	err = json.Unmarshal(out, &items)
	if err != nil {
		return errors.Errorf("failed to unmarshal output: %w", err)
	}

	agent := sshagent.NewKeyring()
	for _, item := range items {
		if item.Type != SshKeyItemType {
			continue
		}
		key, err := ssh.ParseRawPrivateKey([]byte(item.SshKey.Private))
		if err != nil {
			slog.Info("failed to parse private key", "err", err.Error())
			continue
		}
		err = agent.Add(sshagent.AddedKey{
			Comment:    item.Name,
			PrivateKey: key,
		})
		if err != nil {
			slog.Info("failed to add key to agent", "err", err.Error())
			continue
		}
	}

	socketPath := fmt.Sprintf("%s/.ssh/autobw.sock", os.Getenv("HOME"))
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return errors.Errorf("failed to listen on socket: %w", err)
	}
	defer func() {
		println("Cleaning up")
		if err := listener.Close(); err != nil {
			slog.Error("failed to close listener", "err", err.Error())
		}
		if err := os.Remove(socketPath); err != nil {
			slog.Error("failed to remove socket", "err", err.Error())
		}
	}()

	go func() {
		slog.Info("SSH agent started", "socket", socketPath)
		for {
			connection, err := listener.Accept()
			if err != nil {
				slog.Error("accept error", "err", err.Error())
				return
			}

			go func() {
				sshagent.ServeAgent(agent, connection)
			}()
		}
	}()

	<-ctx.Done()
	return nil
}
