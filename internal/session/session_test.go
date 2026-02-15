package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeWorkspace(t *testing.T, dir, id, cwd, updatedAt string) {
	t.Helper()
	sessionDir := filepath.Join(dir, id)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "id: " + id + "\ncwd: " + cwd + "\nupdated_at: " + updatedAt + "\n"
	if err := os.WriteFile(filepath.Join(sessionDir, "workspace.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindLatestSession_NoSessions(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "session-state")
	os.MkdirAll(stateDir, 0o755)

	got, err := FindLatestSession(dir, "/some/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestFindLatestSession_OneMatch(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "session-state")

	writeWorkspace(t, stateDir, "aaa-111", "/my/worktree", "2026-01-10T10:00:00.000Z")

	got, err := FindLatestSession(dir, "/my/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "aaa-111" {
		t.Errorf("got %q, want %q", got, "aaa-111")
	}
}

func TestFindLatestSession_MultipleMatchesPicksLatest(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "session-state")

	writeWorkspace(t, stateDir, "old-session", "/my/worktree", "2026-01-01T00:00:00.000Z")
	writeWorkspace(t, stateDir, "new-session", "/my/worktree", "2026-02-15T12:00:00.000Z")

	got, err := FindLatestSession(dir, "/my/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "new-session" {
		t.Errorf("got %q, want %q", got, "new-session")
	}
}

func TestFindLatestSession_NoMatch(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "session-state")

	writeWorkspace(t, stateDir, "aaa-111", "/other/path", "2026-01-10T10:00:00.000Z")

	got, err := FindLatestSession(dir, "/my/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestFindLatestSession_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "session-state")

	// Write a malformed file
	badDir := filepath.Join(stateDir, "bad-session")
	os.MkdirAll(badDir, 0o755)
	os.WriteFile(filepath.Join(badDir, "workspace.yaml"), []byte("not valid yaml at all\x00\x01"), 0o644)

	// Write a good one
	writeWorkspace(t, stateDir, "good-session", "/my/worktree", "2026-01-10T10:00:00.000Z")

	got, err := FindLatestSession(dir, "/my/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "good-session" {
		t.Errorf("got %q, want %q", got, "good-session")
	}
}

func TestFindLatestSession_MissingStateDir(t *testing.T) {
	dir := t.TempDir()
	// Don't create session-state at all

	got, err := FindLatestSession(dir, "/my/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestFindLatestSession_FallbackToModTime(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "session-state")

	// Write sessions without updated_at — should fall back to file mod time
	s1Dir := filepath.Join(stateDir, "session-a")
	os.MkdirAll(s1Dir, 0o755)
	os.WriteFile(filepath.Join(s1Dir, "workspace.yaml"), []byte("id: session-a\ncwd: /my/worktree\n"), 0o644)

	s2Dir := filepath.Join(stateDir, "session-b")
	os.MkdirAll(s2Dir, 0o755)
	os.WriteFile(filepath.Join(s2Dir, "workspace.yaml"), []byte("id: session-b\ncwd: /my/worktree\n"), 0o644)

	// Set mod times so session-b is newer
	past := time.Now().Add(-1 * time.Hour)
	os.Chtimes(filepath.Join(s1Dir, "workspace.yaml"), past, past)

	got, err := FindLatestSession(dir, "/my/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "session-b" {
		t.Errorf("got %q, want %q", got, "session-b")
	}
}
