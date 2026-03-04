package cliapp

import (
	"errors"
	"os"
	"strings"
)

var errNotInZellij = errors.New("not in a zellij session")

// zellijRun runs a zellij command. Returns errNotInZellij when
// the ZELLIJ environment variable is not set or zellij is not in PATH.
var zellijRun = func(args ...string) error {
	if !isZellij() {
		return errNotInZellij
	}
	zellijPath, err := lookPath("zellij")
	if err != nil {
		return errNotInZellij
	}
	return runCommand(zellijPath, args, "")
}

func isZellij() bool {
	return strings.TrimSpace(os.Getenv("ZELLIJ")) != ""
}

func zellijSessionName() string {
	return strings.TrimSpace(os.Getenv("ZELLIJ_SESSION_NAME"))
}
