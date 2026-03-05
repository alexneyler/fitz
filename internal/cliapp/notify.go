package cliapp

import (
	"errors"
	"fmt"
	"io"
)

func AgentNotify(w io.Writer, clear bool) error {
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
