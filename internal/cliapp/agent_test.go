package cliapp

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"fitz/internal/status"
)

func TestAgentStatusSetsMessage(t *testing.T) {
	origPath := resolveAgentStatusStorePath
	origBranch := resolveCurrentBranch
	origSetStatus := setAgentBranchStatus
	origSetPR := setAgentBranchPR
	t.Cleanup(func() {
		resolveAgentStatusStorePath = origPath
		resolveCurrentBranch = origBranch
		setAgentBranchStatus = origSetStatus
		setAgentBranchPR = origSetPR
	})

	resolveAgentStatusStorePath = func() (string, error) { return "/tmp/status.json", nil }
	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }

	var statusCall struct {
		path, branch, message string
	}
	setAgentBranchStatus = func(path, branch, message string) (status.BranchStatus, error) {
		statusCall = struct {
			path, branch, message string
		}{path: path, branch: branch, message: message}
		return status.BranchStatus{}, nil
	}
	setAgentBranchPR = func(path, branch, prURL string) (status.BranchStatus, error) {
		t.Fatal("set pr should not be called")
		return status.BranchStatus{}, nil
	}

	var out bytes.Buffer
	if err := AgentStatus(&out, "Implementing auth", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if statusCall.path != "/tmp/status.json" || statusCall.branch != "feature-auth" || statusCall.message != "Implementing auth" {
		t.Fatalf("status call = %+v", statusCall)
	}
	if !strings.Contains(out.String(), "updated status for feature-auth") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestAgentStatusSetsPR(t *testing.T) {
	origPath := resolveAgentStatusStorePath
	origBranch := resolveCurrentBranch
	origSetStatus := setAgentBranchStatus
	origSetPR := setAgentBranchPR
	t.Cleanup(func() {
		resolveAgentStatusStorePath = origPath
		resolveCurrentBranch = origBranch
		setAgentBranchStatus = origSetStatus
		setAgentBranchPR = origSetPR
	})

	resolveAgentStatusStorePath = func() (string, error) { return "/tmp/status.json", nil }
	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }
	setAgentBranchStatus = func(path, branch, message string) (status.BranchStatus, error) {
		t.Fatal("set status should not be called")
		return status.BranchStatus{}, nil
	}
	var prCall struct {
		path, branch, prURL string
	}
	setAgentBranchPR = func(path, branch, prURL string) (status.BranchStatus, error) {
		prCall = struct {
			path, branch, prURL string
		}{path: path, branch: branch, prURL: prURL}
		return status.BranchStatus{}, nil
	}

	var out bytes.Buffer
	if err := AgentStatus(&out, "", "https://github.com/acme/repo/pull/42"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prCall.prURL != "https://github.com/acme/repo/pull/42" {
		t.Fatalf("pr call = %+v", prCall)
	}
}

func TestAgentStatusTruncatesMessage(t *testing.T) {
	origPath := resolveAgentStatusStorePath
	origBranch := resolveCurrentBranch
	origSetStatus := setAgentBranchStatus
	origSetPR := setAgentBranchPR
	t.Cleanup(func() {
		resolveAgentStatusStorePath = origPath
		resolveCurrentBranch = origBranch
		setAgentBranchStatus = origSetStatus
		setAgentBranchPR = origSetPR
	})

	resolveAgentStatusStorePath = func() (string, error) { return "/tmp/status.json", nil }
	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }
	long := strings.Repeat("x", 120)

	var got string
	setAgentBranchStatus = func(path, branch, message string) (status.BranchStatus, error) {
		got = message
		return status.BranchStatus{}, nil
	}
	setAgentBranchPR = func(path, branch, prURL string) (status.BranchStatus, error) {
		return status.BranchStatus{}, nil
	}

	var out bytes.Buffer
	if err := AgentStatus(&out, long, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 80 {
		t.Fatalf("len(message) = %d, want 80", len(got))
	}
}

func TestAgentStatusRequiresUpdate(t *testing.T) {
	var out bytes.Buffer
	err := AgentStatus(&out, "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAgentStatusPropagatesErrors(t *testing.T) {
	origPath := resolveAgentStatusStorePath
	origBranch := resolveCurrentBranch
	origSetStatus := setAgentBranchStatus
	origSetPR := setAgentBranchPR
	t.Cleanup(func() {
		resolveAgentStatusStorePath = origPath
		resolveCurrentBranch = origBranch
		setAgentBranchStatus = origSetStatus
		setAgentBranchPR = origSetPR
	})

	resolveAgentStatusStorePath = func() (string, error) { return "", fmt.Errorf("bad repo") }
	resolveCurrentBranch = func() (string, error) { return "feature-auth", nil }
	setAgentBranchStatus = func(path, branch, message string) (status.BranchStatus, error) {
		return status.BranchStatus{}, nil
	}
	setAgentBranchPR = func(path, branch, prURL string) (status.BranchStatus, error) {
		return status.BranchStatus{}, nil
	}

	var out bytes.Buffer
	err := AgentStatus(&out, "hello", "")
	if err == nil || !strings.Contains(err.Error(), "bad repo") {
		t.Fatalf("error = %v", err)
	}
}
