package cliapp

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestAgentNotifyRenamesTab(t *testing.T) {
	origBranch := resolveCurrentBranch
	origRenameTab := zellijRenameTab
	t.Cleanup(func() {
		resolveCurrentBranch = origBranch
		zellijRenameTab = origRenameTab
	})

	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }

	var gotName string
	zellijRenameTab = func(name string) error {
		gotName = name
		return nil
	}

	var out bytes.Buffer
	if err := AgentNotify(&out, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "* feature-auth" {
		t.Fatalf("tab name = %q, want %q", gotName, "* feature-auth")
	}
}

func TestAgentNotifyClearRenamesTab(t *testing.T) {
	origBranch := resolveCurrentBranch
	origRenameTab := zellijRenameTab
	t.Cleanup(func() {
		resolveCurrentBranch = origBranch
		zellijRenameTab = origRenameTab
	})

	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }

	var gotName string
	zellijRenameTab = func(name string) error {
		gotName = name
		return nil
	}

	var out bytes.Buffer
	if err := AgentNotify(&out, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "feature-auth" {
		t.Fatalf("tab name = %q, want %q", gotName, "feature-auth")
	}
}

func TestAgentNotifyBellWhenNotZellij(t *testing.T) {
	origBranch := resolveCurrentBranch
	origRenameTab := zellijRenameTab
	t.Cleanup(func() {
		resolveCurrentBranch = origBranch
		zellijRenameTab = origRenameTab
	})

	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }
	zellijRenameTab = func(name string) error {
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
	origBranch := resolveCurrentBranch
	origRenameTab := zellijRenameTab
	t.Cleanup(func() {
		resolveCurrentBranch = origBranch
		zellijRenameTab = origRenameTab
	})

	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }
	zellijRenameTab = func(name string) error {
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

func TestAgentNotifyPropagatesBranchError(t *testing.T) {
	origBranch := resolveCurrentBranch
	t.Cleanup(func() { resolveCurrentBranch = origBranch })

	resolveCurrentBranch = func() (string, error) { return "", fmt.Errorf("not a repo") }

	var out bytes.Buffer
	err := AgentNotify(&out, false)
	if err == nil || !strings.Contains(err.Error(), "not a repo") {
		t.Fatalf("error = %v", err)
	}
}
