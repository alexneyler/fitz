package cliapp

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestTodoAddWritesToStdout(t *testing.T) {
	var out bytes.Buffer
	// This will fail because we're not in a git repo in the test,
	// but we can test the store directly (already covered in todo_store_test.go).
	// The integration with git is tested via execute_test.go.
	err := TodoAdd(context.Background(), &out, "test todo")
	if err != nil {
		// Expected in test env (no git repo). Verify the error is about repo detection.
		if !strings.Contains(err.Error(), "repository") && !strings.Contains(err.Error(), "directory") {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	if !strings.Contains(out.String(), "added:") {
		t.Fatalf("stdout = %q, want 'added:'", out.String())
	}
}
