package cliapp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var getwd = os.Getwd
var userHomeDir = os.UserHomeDir

func AgentNotify(w io.Writer, clear bool) error {
	cwd, err := getwd()
	if err != nil {
		return nil
	}

	homeDir, err := userHomeDir()
	if err != nil {
		return nil
	}

	fitzDir := filepath.Join(homeDir, ".fitz")
	if !strings.HasPrefix(cwd, fitzDir+string(filepath.Separator)) {
		return nil
	}

	branch, err := resolveCurrentBranch()
	if err != nil {
		// Not in a git repo / worktree — silently no-op.
		return nil
	}

	tabName := "* " + branch
	if clear {
		tabName = branch
	}

	if err := zellijRun("action", "rename-tab", tabName); err != nil {
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
