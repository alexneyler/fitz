package cliapp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var errNotInZellij = errors.New("not in a zellij session")

var zellijRenameTab = func(name string) error {
	if strings.TrimSpace(os.Getenv("ZELLIJ")) == "" {
		return errNotInZellij
	}
	zellijPath, err := lookPath("zellij")
	if err != nil {
		return errNotInZellij
	}
	return runCommand(zellijPath, []string{"action", "rename-tab", name}, "")
}

func AgentNotify(w io.Writer, clear bool) error {
	branch, err := resolveCurrentBranch()
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}

	tabName := "* " + branch
	if clear {
		tabName = branch
	}

	if err := zellijRenameTab(tabName); err != nil {
		if errors.Is(err, errNotInZellij) {
			// Fall back to terminal bell for non-Zellij environments.
			if !clear {
				fmt.Fprint(w, "\a")
			}
			return nil
		}
		return fmt.Errorf("rename tab: %w", err)
	}
	return nil
}
