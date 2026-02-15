package cliapp

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestTodoAddWritesToStdout(t *testing.T) {
	orig := resolveTodoStorePath
	t.Cleanup(func() { resolveTodoStorePath = orig })

	dir := t.TempDir()
	storePath := dir + "/todos.json"
	resolveTodoStorePath = func() (string, error) { return storePath, nil }

	var out bytes.Buffer
	err := TodoAdd(context.Background(), &out, "test todo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "added:") {
		t.Fatalf("stdout = %q, want 'added:'", out.String())
	}
	if !strings.Contains(out.String(), "test todo") {
		t.Fatalf("stdout = %q, want 'test todo'", out.String())
	}
}

func TestTodoAddError(t *testing.T) {
	orig := resolveTodoStorePath
	t.Cleanup(func() { resolveTodoStorePath = orig })

	resolveTodoStorePath = func() (string, error) {
		return "", fmt.Errorf("identify repository: no git repo")
	}

	var out bytes.Buffer
	err := TodoAdd(context.Background(), &out, "test todo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "repository") {
		t.Fatalf("error = %q, want it to mention 'repository'", err.Error())
	}
}
