package cliapp

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func stubFitzDir(t *testing.T) {
	t.Helper()
	origGetwd := getwd
	origHome := userHomeDir
	t.Cleanup(func() {
		getwd = origGetwd
		userHomeDir = origHome
	})
	home := "/fake/home"
	userHomeDir = func() (string, error) { return home, nil }
	getwd = func() (string, error) { return home + "/.fitz/owner/repo/branch", nil }
}

func TestAgentNotifyNoOpWhenNotInFitzDir(t *testing.T) {
	origBranch := resolveCurrentBranch
	origRun := zellijRun
	origGetwd := getwd
	t.Cleanup(func() {
		resolveCurrentBranch = origBranch
		zellijRun = origRun
		getwd = origGetwd
	})

	getwd = func() (string, error) { return "/some/random/dir", nil }
	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }

	called := false
	zellijRun = func(args ...string) error {
		called = true
		return nil
	}

	var out bytes.Buffer
	if err := AgentNotify(&out, false); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if called {
		t.Fatal("zellijRun should not have been called")
	}
	if out.String() != "" {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
}

func TestAgentNotifyRenamesTab(t *testing.T) {
	stubFitzDir(t)
	origBranch := resolveCurrentBranch
	origRun := zellijRun
	t.Cleanup(func() {
		resolveCurrentBranch = origBranch
		zellijRun = origRun
	})

	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }

	var gotArgs []string
	zellijRun = func(args ...string) error {
		gotArgs = args
		return nil
	}

	var out bytes.Buffer
	if err := AgentNotify(&out, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotArgs) != 3 || gotArgs[0] != "action" || gotArgs[1] != "rename-tab" || gotArgs[2] != "* feature-auth" {
		t.Fatalf("zellijRun args = %v, want [action rename-tab * feature-auth]", gotArgs)
	}
}

func TestAgentNotifyClearRenamesTab(t *testing.T) {
	stubFitzDir(t)
	origBranch := resolveCurrentBranch
	origRun := zellijRun
	t.Cleanup(func() {
		resolveCurrentBranch = origBranch
		zellijRun = origRun
	})

	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }

	var gotArgs []string
	zellijRun = func(args ...string) error {
		gotArgs = args
		return nil
	}

	var out bytes.Buffer
	if err := AgentNotify(&out, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotArgs) != 3 || gotArgs[2] != "feature-auth" {
		t.Fatalf("zellijRun args = %v, want [action rename-tab feature-auth]", gotArgs)
	}
}

func TestAgentNotifyBellWhenNotZellij(t *testing.T) {
	stubFitzDir(t)
	origBranch := resolveCurrentBranch
	origRun := zellijRun
	t.Cleanup(func() {
		resolveCurrentBranch = origBranch
		zellijRun = origRun
	})

	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }
	zellijRun = func(args ...string) error {
		return errNotInZellij
	}

	var out bytes.Buffer
	if err := AgentNotify(&out, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "\a") {
		t.Fatalf("stdout = %q, want BEL character", out.String())
	}
}

func TestAgentNotifyClearNoOutputWhenNotZellij(t *testing.T) {
	stubFitzDir(t)
	origBranch := resolveCurrentBranch
	origRun := zellijRun
	t.Cleanup(func() {
		resolveCurrentBranch = origBranch
		zellijRun = origRun
	})

	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }
	zellijRun = func(args ...string) error {
		return errNotInZellij
	}

	var out bytes.Buffer
	if err := AgentNotify(&out, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.String() != "" {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
}

func TestAgentNotifyNoOpWhenNotInRepo(t *testing.T) {
	stubFitzDir(t)
	origBranch := resolveCurrentBranch
	origRun := zellijRun
	t.Cleanup(func() {
		resolveCurrentBranch = origBranch
		zellijRun = origRun
	})

	resolveCurrentBranch = func() (string, error) { return "", fmt.Errorf("not a repo") }

	called := false
	zellijRun = func(args ...string) error {
		called = true
		return nil
	}

	var out bytes.Buffer
	if err := AgentNotify(&out, false); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if called {
		t.Fatal("zellijRun should not have been called")
	}
	if out.String() != "" {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
}
