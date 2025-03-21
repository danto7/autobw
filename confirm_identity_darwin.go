package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aethiopicuschan/touchid-go"
)

var ErrorAuthenticationFailed = fmt.Errorf("Authentication failed")
var ErrorAuthenticationTimedOut = fmt.Errorf("Authentication timed out")

func confirmIdentity() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ok, err := touchid.Authenticate(ctx, "confirm that it's you")
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ErrorAuthenticationTimedOut
		}
		return fmt.Errorf("Error authenticating: %w", err)
	}

	if !ok {
		slog.Debug("Authentication failed")
		return ErrorAuthenticationFailed
	}
	slog.Debug("Authentication successful")
	return nil
}
