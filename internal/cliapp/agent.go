package cliapp

import (
	"fmt"
	"io"
	"os"
	"strings"

	"fitz/internal/status"
	"fitz/internal/worktree"
)

var resolveAgentStatusStorePath = resolveAgentStatusPath
var resolveCurrentBranch = currentBranch
var setAgentBranchStatus = status.SetStatus
var setAgentBranchPR = status.SetPR

func AgentStatus(w io.Writer, message, prURL string) error {
	message = strings.TrimSpace(message)
	prURL = strings.TrimSpace(prURL)
	if message == "" && prURL == "" {
		return fmt.Errorf("usage: fitz agent status [--pr <url>] [message]")
	}

	storePath, err := resolveAgentStatusStorePath()
	if err != nil {
		return err
	}

	branch, err := resolveCurrentBranch()
	if err != nil {
		return err
	}
	if branch == "" || branch == "HEAD" {
		return fmt.Errorf("cannot set agent status on detached HEAD")
	}

	if message != "" {
		message = truncateStatusMessage(message)
		if _, err := setAgentBranchStatus(storePath, branch, message); err != nil {
			return fmt.Errorf("update status: %w", err)
		}
	}
	if prURL != "" {
		if _, err := setAgentBranchPR(storePath, branch, prURL); err != nil {
			return fmt.Errorf("update pull request: %w", err)
		}
	}

	fmt.Fprintf(w, "updated status for %s\n", branch)
	return nil
}

func truncateStatusMessage(message string) string {
	if len(message) <= 80 {
		return message
	}
	return message[:80]
}

func resolveAgentStatusPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	git := worktree.ShellGit{}
	owner, repo, err := worktree.RepoID(git, cwd)
	if err != nil {
		return "", fmt.Errorf("identify repository: %w", err)
	}

	path, err := status.StorePath("", owner, repo)
	if err != nil {
		return "", fmt.Errorf("resolve status store path: %w", err)
	}
	return path, nil
}

func currentBranch() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	git := worktree.ShellGit{}
	branch, err := git.Run(cwd, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}
	return strings.TrimSpace(branch), nil
}
