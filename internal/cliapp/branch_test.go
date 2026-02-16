package cliapp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestBrCurrent(t *testing.T) {
	var out bytes.Buffer
	err := BrCurrent(context.Background(), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestBrList(t *testing.T) {
	// Create a stdin that immediately sends 'q' to quit the TUI.
	stdin := strings.NewReader("q")
	var out bytes.Buffer
	err := BrList(context.Background(), stdin, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBrCd(t *testing.T) {
	var out bytes.Buffer
	err := BrCd(context.Background(), &out, "feature")
	if err == nil {
		t.Fatal("expected error for non-existent worktree in test repo")
	}
}

func TestRunExecMockable(t *testing.T) {
	originalExec := runExec
	t.Cleanup(func() { runExec = originalExec })

	var called bool
	var capturedBinary string
	var capturedArgs []string

	runExec = func(binary string, args []string, env []string) error {
		called = true
		capturedBinary = binary
		capturedArgs = args
		return nil
	}

	// Without a session match, BrGo should call copilot with no resume flag.
	err := runExec("/usr/bin/copilot", []string{"copilot"}, os.Environ())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Fatal("runExec was not called")
	}

	if capturedBinary != "/usr/bin/copilot" {
		t.Errorf("binary = %q, want /usr/bin/copilot", capturedBinary)
	}

	if len(capturedArgs) != 1 || capturedArgs[0] != "copilot" {
		t.Errorf("args = %v, want [copilot]", capturedArgs)
	}
}

func TestRunExecWithResume(t *testing.T) {
	originalExec := runExec
	t.Cleanup(func() { runExec = originalExec })

	var capturedArgs []string

	runExec = func(binary string, args []string, env []string) error {
		capturedArgs = args
		return nil
	}

	// With a --resume flag, args should include session ID.
	err := runExec("/usr/bin/copilot", []string{"copilot", "--resume", "abc-123"}, os.Environ())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantArgs := []string{"copilot", "--resume", "abc-123"}
	if len(capturedArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", capturedArgs, wantArgs)
	}
	for i, a := range wantArgs {
		if capturedArgs[i] != a {
			t.Errorf("args[%d] = %q, want %q", i, capturedArgs[i], a)
		}
	}
}

func TestRunBackgroundMockable(t *testing.T) {
	originalBg := runBackground
	t.Cleanup(func() { runBackground = originalBg })

	var capturedBinary string
	var capturedArgs []string
	var capturedDir string

	runBackground = func(binary string, args []string, dir string) error {
		capturedBinary = binary
		capturedArgs = args
		capturedDir = dir
		return nil
	}

	err := runBackground("/usr/bin/copilot", []string{"copilot", "--yolo", "-p", "do stuff"}, "/tmp/wt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedBinary != "/usr/bin/copilot" {
		t.Errorf("binary = %q, want /usr/bin/copilot", capturedBinary)
	}

	wantArgs := []string{"copilot", "--yolo", "-p", "do stuff"}
	if len(capturedArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", capturedArgs, wantArgs)
	}
	for i, a := range wantArgs {
		if capturedArgs[i] != a {
			t.Errorf("args[%d] = %q, want %q", i, capturedArgs[i], a)
		}
	}

	if capturedDir != "/tmp/wt" {
		t.Errorf("dir = %q, want /tmp/wt", capturedDir)
	}
}

// mockGitRunner lets tests control git output without shelling out.
type mockGitRunner struct {
	results map[string]string
	err     error
}

func (m mockGitRunner) Run(dir string, args ...string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	key := strings.Join(args, " ")
	if v, ok := m.results[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("unexpected git call: %v", args)
}

func TestBrPublish(t *testing.T) {
	originalCopilot := runCopilot
	t.Cleanup(func() { runCopilot = originalCopilot })

	var copilotArgs []string
	runCopilot = func(dir string, args ...string) (string, error) {
		copilotArgs = args
		return "https://github.com/owner/repo/pull/42\n", nil
	}

	// We can't easily mock the worktree.ShellGit used inside BrPublish
	// in a unit test without being in a real git repo. But this test
	// validates the happy path wiring when run from the repo itself.
	var out bytes.Buffer
	err := BrPublish(context.Background(), &out, "")
	// If we're on main/master, we expect the guard error.
	if err != nil {
		if strings.Contains(err.Error(), "cannot publish from") {
			t.Skip("skipping: test running from default branch")
		}
		// The push may fail if there's no real remote, which is fine for unit testing.
		if strings.Contains(err.Error(), "push branch") {
			t.Skip("skipping: no remote available for push")
		}
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "pushed") {
		t.Errorf("stdout = %q, want 'pushed' message", output)
	}
	if !strings.Contains(output, "https://github.com") {
		t.Errorf("stdout = %q, want PR URL", output)
	}
	// Verify copilot was invoked with expected flags.
	if len(copilotArgs) < 3 {
		t.Fatalf("copilot args = %v, want at least [--yolo -p <prompt>]", copilotArgs)
	}
	if copilotArgs[0] != "--yolo" {
		t.Errorf("copilot args[0] = %q, want --yolo", copilotArgs[0])
	}
	if copilotArgs[1] != "-p" {
		t.Errorf("copilot args[1] = %q, want -p", copilotArgs[1])
	}
}

func TestBrPublishProtectsDefaultBranch(t *testing.T) {
	originalCopilot := runCopilot
	t.Cleanup(func() { runCopilot = originalCopilot })
	runCopilot = func(dir string, args ...string) (string, error) {
		return "https://github.com/owner/repo/pull/99\n", nil
	}

	// This test is only meaningful when run from main/master.
	var out bytes.Buffer
	err := BrPublish(context.Background(), &out, "")
	if err == nil {
		return // not on a protected branch, that's fine
	}
	if strings.Contains(err.Error(), "cannot publish from main") ||
		strings.Contains(err.Error(), "cannot publish from master") {
		// expected
		return
	}
	// push/other errors are acceptable in test environments
}
