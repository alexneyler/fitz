package cliapp

import (
	"bytes"
	"context"
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
	var out bytes.Buffer
	err := BrList(context.Background(), &out)
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

	err := runExec("/usr/bin/copilot", []string{"copilot", "--continue"}, os.Environ())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Fatal("runExec was not called")
	}

	if capturedBinary != "/usr/bin/copilot" {
		t.Errorf("binary = %q, want /usr/bin/copilot", capturedBinary)
	}

	if len(capturedArgs) != 2 || capturedArgs[0] != "copilot" || capturedArgs[1] != "--continue" {
		t.Errorf("args = %v, want [copilot --continue]", capturedArgs)
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
