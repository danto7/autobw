package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
)

type BitwardenStatus struct {
	ServerUrl string `json:"serverUrl"`
	LastSync  string `json:"lastSync"`
	UserEmail string `json:"userEmail"`
	UserId    string `json:"userId"`
	Status    string `json:"status"`
}

func status(session string) (*BitwardenStatus, error) {
	stdout := new(bytes.Buffer)

	cmd := exec.Command(bwBinary, "status")
	cmd.Stderr = os.Stderr
	cmd.Stdout = stdout
	cmd.Env = append(os.Environ(), "BW_SESSION="+session)
	err := cmd.Run()
	if err != nil {
		return nil, err
	}
	var status BitwardenStatus
	err = json.Unmarshal(stdout.Bytes(), &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}
