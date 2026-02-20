package cliapp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"fitz/internal/status"
	"fitz/internal/worktree"
)

// copilotResult holds the output and error from a background Copilot run.
type copilotResult struct {
	output string
	err    error
}

// runCopilotAsync launches Copilot asynchronously and sends the result on the
// returned channel when it completes.
var runCopilotAsync = func(dir string, args ...string) <-chan copilotResult {
	ch := make(chan copilotResult, 1)
	go func() {
		output, err := runCopilot(dir, args...)
		ch <- copilotResult{output: output, err: err}
	}()
	return ch
}

// reviewGit is a seam for testing git operations during review.
var reviewGit = func() worktree.ShellGit { return worktree.ShellGit{} }

// reviewStatusInterval controls poll frequency.
var reviewStatusInterval = 500 * time.Millisecond

func Review(_ context.Context, w io.Writer, focus string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	git := reviewGit()
	branch, err := git.Run(cwd, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}
	branch = strings.TrimSpace(branch)

	defaultBranch := detectDefaultBranch(git, cwd)

	reviewDir := cwd
	if branch == defaultBranch || branch == "HEAD" {
		// On default branch — create a worktree for the review.
		mgr := &worktree.Manager{Git: git}
		name := fmt.Sprintf("review-%d", time.Now().Unix())
		path, err := mgr.Create(cwd, name, "")
		if err != nil {
			return fmt.Errorf("create review worktree: %w", err)
		}
		reviewDir = path
		branch = name
		fmt.Fprintf(w, "created review worktree: %s\n", name)
	}

	// Compute diff for the prompt.
	diff, err := computeBranchDiff(git, reviewDir, defaultBranch)
	if err != nil {
		// Non-fatal: review without diff context.
		diff = ""
	}

	prompt := buildReviewPrompt(focus, diff)
	cfg := loadEffectiveConfig(cwd)
	args := append(copilotBaseArgs(cfg)[1:], "--yolo", "-p", prompt)

	// Resolve status path for polling.
	statusPath, statusBranch := resolveReviewStatusInfo(cwd, branch)

	fmt.Fprintf(w, "⟳ Review: starting\n")

	resultCh := runCopilotAsync(reviewDir, args...)

	// Poll status.json for live updates.
	var lastMessage string
	for {
		select {
		case result := <-resultCh:
			// Copilot finished — print final output.
			if result.err != nil {
				fmt.Fprintf(w, "✗ Review: failed\n")
				return fmt.Errorf("run review: %w", result.err)
			}

			output := strings.TrimSpace(result.output)
			if output == "" {
				output = "No actionable findings."
			}
			fmt.Fprintf(w, "✓ Review: complete\n\n")
			_, err := fmt.Fprintln(w, output)
			return err

		case <-time.After(reviewStatusInterval):
			if statusPath == "" {
				continue
			}
			msg := readLatestStatus(statusPath, statusBranch)
			if msg != "" && msg != lastMessage {
				lastMessage = msg
				fmt.Fprintf(w, "⟳ %s\n", msg)
			}
		}
	}
}

func computeBranchDiff(git worktree.ShellGit, dir, defaultBranch string) (string, error) {
	out, err := git.Run(dir, "diff", defaultBranch+"...HEAD")
	if err != nil {
		return "", err
	}
	// Cap diff size to avoid enormous prompts.
	const maxDiffLen = 50000
	if len(out) > maxDiffLen {
		out = out[:maxDiffLen] + "\n... (diff truncated)\n"
	}
	return out, nil
}

func buildReviewPrompt(focus, diff string) string {
	focus = strings.TrimSpace(focus)
	var b strings.Builder

	b.WriteString("You are a code reviewer. Review the changes in this branch.\n")
	b.WriteString("Use the review skill.\n")
	b.WriteString("Use multiple sub-agents with different models for independent reviews.\n")
	b.WriteString("Then run one additional agent that reviews all findings for validity and nitpickiness.\n")
	b.WriteString("Only keep high-confidence actionable issues.\n")
	b.WriteString("At each major phase, run: fitz agent status \"Review: <phase>\" to report progress.\n")
	b.WriteString("Phases: \"running reviewers\", \"adjudicating findings\", \"finalizing\".\n")
	b.WriteString("Output only the final actionable list.\n")
	b.WriteString("If nothing actionable is found, output exactly: No actionable findings.\n")
	b.WriteString("Do not include process notes. Return only the final actionable list.\n")

	if focus != "" {
		b.WriteString("Focus area: ")
		b.WriteString(focus)
		b.WriteString("\n")
	}

	b.WriteString("Required final actionable list format:\n")
	b.WriteString("- [severity] path:line - issue summary (why it matters)\n")

	if diff != "" {
		b.WriteString("\n--- Branch diff ---\n")
		b.WriteString(diff)
		b.WriteString("\n--- End diff ---\n")
	}

	return b.String()
}

// resolveReviewStatusInfo returns the status file path and branch name for polling.
// Returns empty strings if resolution fails (non-fatal).
func resolveReviewStatusInfo(cwd, branch string) (string, string) {
	git := worktree.ShellGit{}
	owner, repo, err := worktree.RepoID(git, cwd)
	if err != nil {
		return "", ""
	}
	path, err := status.StorePath("", owner, repo)
	if err != nil {
		return "", ""
	}
	return path, branch
}

// readLatestStatus reads the current status message for a branch.
func readLatestStatus(path, branch string) string {
	entries, err := status.Load(path)
	if err != nil {
		return ""
	}
	if entry, ok := entries[branch]; ok {
		return entry.Message
	}
	return ""
}

// computeBranchDiffCmd is used for testing — wraps exec.Command for diff.
var computeBranchDiffCmd = func(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}
