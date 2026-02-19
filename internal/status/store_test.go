package status

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	entries, err := Load(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(entries))
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	want := map[string]BranchStatus{
		"feature-auth": {
			Message:   "Implementing auth",
			PRURL:     "https://github.com/acme/repo/pull/42",
			UpdatedAt: time.Date(2026, 2, 19, 6, 0, 0, 0, time.UTC),
		},
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("save error: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if got["feature-auth"].Message != want["feature-auth"].Message {
		t.Fatalf("message = %q, want %q", got["feature-auth"].Message, want["feature-auth"].Message)
	}
	if got["feature-auth"].PRURL != want["feature-auth"].PRURL {
		t.Fatalf("pr url = %q, want %q", got["feature-auth"].PRURL, want["feature-auth"].PRURL)
	}
}

func TestSetStatusPreservesPR(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	initial := map[string]BranchStatus{
		"feature-auth": {PRURL: "https://github.com/acme/repo/pull/42"},
	}
	if err := Save(path, initial); err != nil {
		t.Fatalf("save error: %v", err)
	}

	_, err := SetStatus(path, "feature-auth", "Implementing auth")
	if err != nil {
		t.Fatalf("set status error: %v", err)
	}

	entries, err := Load(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	got := entries["feature-auth"]
	if got.Message != "Implementing auth" {
		t.Fatalf("message = %q", got.Message)
	}
	if got.PRURL != "https://github.com/acme/repo/pull/42" {
		t.Fatalf("pr url = %q", got.PRURL)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatal("expected updated_at to be set")
	}
}

func TestSetPRPreservesMessage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status.json")
	initial := map[string]BranchStatus{
		"feature-auth": {Message: "Implementing auth"},
	}
	if err := Save(path, initial); err != nil {
		t.Fatalf("save error: %v", err)
	}

	_, err := SetPR(path, "feature-auth", "https://github.com/acme/repo/pull/42")
	if err != nil {
		t.Fatalf("set pr error: %v", err)
	}

	entries, err := Load(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	got := entries["feature-auth"]
	if got.Message != "Implementing auth" {
		t.Fatalf("message = %q", got.Message)
	}
	if got.PRURL != "https://github.com/acme/repo/pull/42" {
		t.Fatalf("pr url = %q", got.PRURL)
	}
	if got.UpdatedAt.IsZero() {
		t.Fatal("expected updated_at to be set")
	}
}

func TestStorePath(t *testing.T) {
	path, err := StorePath("/fakehome", "myowner", "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join("/fakehome", ".fitz", "myowner", "myrepo", "status.json")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}
