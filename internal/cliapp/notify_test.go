package cliapp

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestAgentNotifyRenamesTab(t *testing.T) {
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
