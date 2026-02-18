package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeWorkspace(t *testing.T, dir, id, cwd, updatedAt string) {
	t.Helper()
	writeWorkspaceFull(t, dir, id, cwd, updatedAt, "")
}

func writeWorkspaceFull(t *testing.T, dir, id, cwd, updatedAt, summary string) {
	t.Helper()
	sessionDir := filepath.Join(dir, id)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "id: " + id + "\ncwd: " + cwd + "\nupdated_at: " + updatedAt + "\n"
	if summary != "" {
		content += "summary: " + summary + "\n"
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "workspace.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeWorkspaceBlockSummary(t *testing.T, dir, id, cwd, updatedAt string, summaryLines []string) {
	t.Helper()
	sessionDir := filepath.Join(dir, id)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "id: " + id + "\ncwd: " + cwd + "\nupdated_at: " + updatedAt + "\n"
	content += "summary: |-\n"
	for _, line := range summaryLines {
		content += "  " + line + "\n"
	}
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

func TestFindSessionInfo_NoSession(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "session-state"), 0o755)

	info, err := FindSessionInfo(dir, "/my/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.SessionID != "" {
		t.Errorf("expected empty SessionID, got %q", info.SessionID)
	}
	if !info.UpdatedAt.IsZero() {
		t.Errorf("expected zero UpdatedAt, got %v", info.UpdatedAt)
	}
}

func TestFindSessionInfo_InlineSummary(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "session-state")
	writeWorkspaceFull(t, stateDir, "sess-1", "/my/worktree", "2026-02-01T10:00:00.000Z", "Implement auth feature")

	info, err := FindSessionInfo(dir, "/my/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", info.SessionID, "sess-1")
	}
	if info.Summary != "Implement auth feature" {
		t.Errorf("Summary = %q, want %q", info.Summary, "Implement auth feature")
	}
	if info.UpdatedAt.IsZero() {
		t.Error("expected non-zero UpdatedAt")
	}
}

func TestFindSessionInfo_BlockSummaryFirstLine(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "session-state")
	writeWorkspaceBlockSummary(t, stateDir, "sess-1", "/my/worktree", "2026-02-01T10:00:00.000Z", []string{
		"Refactor the authentication module",
		"Added unit tests for login flow",
		"Updated documentation",
	})

	info, err := FindSessionInfo(dir, "/my/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Summary != "Refactor the authentication module" {
		t.Errorf("Summary = %q, want first line only", info.Summary)
	}
}

func TestFindSessionInfo_NoSummary(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "session-state")
	writeWorkspace(t, stateDir, "sess-1", "/my/worktree", "2026-02-01T10:00:00.000Z")

	info, err := FindSessionInfo(dir, "/my/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Summary != "" {
		t.Errorf("Summary = %q, want empty", info.Summary)
	}
}

func TestFindAllSessionInfos_MultipleWorktrees(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "session-state")
	writeWorkspaceFull(t, stateDir, "sess-a", "/repo/feature-a", "2026-02-01T10:00:00.000Z", "Feature A work")
	writeWorkspaceFull(t, stateDir, "sess-b", "/repo/feature-b", "2026-02-02T10:00:00.000Z", "Feature B work")
	writeWorkspaceFull(t, stateDir, "sess-other", "/other/path", "2026-02-03T10:00:00.000Z", "Irrelevant")

	cwds := []string{"/repo/feature-a", "/repo/feature-b", "/repo/feature-c"}
	infos, err := FindAllSessionInfos(dir, cwds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("got %d entries, want 2", len(infos))
	}
	if infos["/repo/feature-a"].SessionID != "sess-a" {
		t.Errorf("feature-a SessionID = %q, want sess-a", infos["/repo/feature-a"].SessionID)
	}
	if infos["/repo/feature-a"].Summary != "Feature A work" {
		t.Errorf("feature-a Summary = %q, want 'Feature A work'", infos["/repo/feature-a"].Summary)
	}
	if infos["/repo/feature-b"].SessionID != "sess-b" {
		t.Errorf("feature-b SessionID = %q, want sess-b", infos["/repo/feature-b"].SessionID)
	}
	// feature-c has no session — should not appear in map
	if _, ok := infos["/repo/feature-c"]; ok {
		t.Error("feature-c should not have an entry")
	}
}

func TestFindAllSessionInfos_PicksLatestPerCwd(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "session-state")
	writeWorkspaceFull(t, stateDir, "sess-old", "/repo/feature-a", "2026-01-01T00:00:00.000Z", "Old work")
	writeWorkspaceFull(t, stateDir, "sess-new", "/repo/feature-a", "2026-02-15T12:00:00.000Z", "New work")

	infos, err := FindAllSessionInfos(dir, []string{"/repo/feature-a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if infos["/repo/feature-a"].SessionID != "sess-new" {
		t.Errorf("SessionID = %q, want sess-new", infos["/repo/feature-a"].SessionID)
	}
}

func TestFindAllSessionInfos_EmptyCwds(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "session-state"), 0o755)

	infos, err := FindAllSessionInfos(dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("got %d entries, want 0", len(infos))
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
