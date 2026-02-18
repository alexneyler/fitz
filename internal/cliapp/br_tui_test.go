package cliapp

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"fitz/internal/session"
	"fitz/internal/worktree"
)

func TestBrNavigationSkipsRoot(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo", Bare: false}, // root
		{Path: "/repo/.fitz/owner/repo/feature-1", Branch: "feature-1", Name: "feature-1"},
		{Path: "/repo/.fitz/owner/repo/feature-2", Branch: "feature-2", Name: "feature-2"},
	}
	m := newBrModel(worktrees, "root", nil)

	// Cursor should start at 1 (first non-root).
	if m.cursor != 1 {
		t.Fatalf("initial cursor = %d, want 1", m.cursor)
	}

	// Press 'up' — should stay at 1 (can't go to root).
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	model := updated.(brModel)
	if model.cursor != 1 {
		t.Fatalf("cursor after up = %d, want 1 (cannot move to root)", model.cursor)
	}

	// Press 'down' — should move to 2.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(brModel)
	if model.cursor != 2 {
		t.Fatalf("cursor after down = %d, want 2", model.cursor)
	}

	// Press 'down' again — should stay at 2 (last item).
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(brModel)
	if model.cursor != 2 {
		t.Fatalf("cursor after second down = %d, want 2 (at end)", model.cursor)
	}
}

func TestBrEnterSetsGoResult(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
		{Path: "/repo/.fitz/owner/repo/feature-1", Branch: "feature-1", Name: "feature-1"},
	}
	m := newBrModel(worktrees, "root", nil)

	// Press enter on the first selectable worktree.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(brModel)

	if model.result.Action != BrActionGo {
		t.Fatalf("action = %d, want BrActionGo", model.result.Action)
	}
	if model.result.Name != "feature-1" {
		t.Fatalf("name = %q, want 'feature-1'", model.result.Name)
	}
	if !model.quitting {
		t.Fatal("expected quitting after enter")
	}
}

func TestBrDeleteRequiresConfirmation(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
		{Path: "/repo/.fitz/owner/repo/feature-1", Branch: "feature-1", Name: "feature-1"},
	}
	m := newBrModel(worktrees, "root", nil)

	// Press 'd' to start delete.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := updated.(brModel)

	if model.state != brStateConfirmDelete {
		t.Fatalf("state = %d, want brStateConfirmDelete", model.state)
	}
	if model.confirmName != "feature-1" {
		t.Fatalf("confirmName = %q, want 'feature-1'", model.confirmName)
	}
}

func TestBrDeleteConfirmationNo(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
		{Path: "/repo/.fitz/owner/repo/feature-1", Branch: "feature-1", Name: "feature-1"},
	}
	m := newBrModel(worktrees, "root", nil)

	// Start delete.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := updated.(brModel)

	// Press 'n' to cancel.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model = updated.(brModel)

	if model.state != brStateList {
		t.Fatalf("state = %d, want brStateList after cancel", model.state)
	}
	if len(model.worktrees) != 2 {
		t.Fatalf("worktrees count = %d, should still be 2", len(model.worktrees))
	}
}

func TestBrDeleteConfirmationYes(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
		{Path: "/repo/.fitz/owner/repo/feature-1", Branch: "feature-1", Name: "feature-1"},
		{Path: "/repo/.fitz/owner/repo/feature-2", Branch: "feature-2", Name: "feature-2"},
	}
	m := newBrModel(worktrees, "root", nil)

	// Start delete.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := updated.(brModel)

	// Press 'y' to confirm — should start dissolve.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = updated.(brModel)

	if model.dissolving != 1 {
		t.Fatalf("dissolving = %d, want 1", model.dissolving)
	}
	if model.dissolveFrame != 1 {
		t.Fatalf("dissolveFrame = %d, want 1", model.dissolveFrame)
	}
}

func TestBrNewFlowBranchInput(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
	}
	m := newBrModel(worktrees, "root", nil)

	// Press 'n' to create new worktree.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model := updated.(brModel)

	if model.state != brStateNewBranch {
		t.Fatalf("state = %d, want brStateNewBranch", model.state)
	}
}

func TestBrNewFlowActionChoice(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
	}
	m := newBrModel(worktrees, "root", nil)

	// Press 'n', then enter branch name.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model := updated.(brModel)
	model.branchInput.SetValue("my-feature")

	// Press enter to confirm branch name.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(brModel)

	if model.state != brStateNewAction {
		t.Fatalf("state = %d, want brStateNewAction", model.state)
	}
	if model.result.BranchName != "my-feature" {
		t.Fatalf("branchName = %q, want 'my-feature'", model.result.BranchName)
	}
}

func TestBrNewFlowGoAction(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
	}
	m := newBrModel(worktrees, "root", nil)

	// Navigate to action choice.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model := updated.(brModel)
	model.branchInput.SetValue("my-feature")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(brModel)

	// Press enter on "Create and go" (action cursor = 0).
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(brModel)

	if model.result.Action != BrActionNew {
		t.Fatalf("action = %d, want BrActionNew", model.result.Action)
	}
	if model.result.Prompt != "" {
		t.Fatalf("prompt = %q, want empty for 'go' action", model.result.Prompt)
	}
	if !model.quitting {
		t.Fatal("expected quitting after action selection")
	}
}

func TestBrNewFlowKickoffAction(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
	}
	m := newBrModel(worktrees, "root", nil)

	// Navigate to action choice.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model := updated.(brModel)
	model.branchInput.SetValue("my-feature")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(brModel)

	// Move down to "Create and kickoff".
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(brModel)

	// Press enter to select kickoff.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(brModel)

	if model.state != brStateNewPrompt {
		t.Fatalf("state = %d, want brStateNewPrompt", model.state)
	}
}

func TestBrNewFlowKickoffPrompt(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
	}
	m := newBrModel(worktrees, "root", nil)

	// Navigate to prompt state.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model := updated.(brModel)
	model.branchInput.SetValue("my-feature")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(brModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(brModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(brModel)

	// Enter a prompt.
	model.promptInput.SetValue("implement login feature")

	// Press enter to confirm.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(brModel)

	if model.result.Action != BrActionNewKickoff {
		t.Fatalf("action = %d, want BrActionNewKickoff", model.result.Action)
	}
	if model.result.Prompt != "implement login feature" {
		t.Fatalf("prompt = %q, want 'implement login feature'", model.result.Prompt)
	}
	if !model.quitting {
		t.Fatal("expected quitting after prompt confirmation")
	}
}

func TestBrPublishSetsResult(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
		{Path: "/repo/.fitz/owner/repo/feature-1", Branch: "feature-1", Name: "feature-1"},
	}
	m := newBrModel(worktrees, "root", nil)

	// Press 'p' to publish.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	model := updated.(brModel)

	if model.result.Action != BrActionPublish {
		t.Fatalf("action = %d, want BrActionPublish", model.result.Action)
	}
	if model.result.Name != "feature-1" {
		t.Fatalf("name = %q, want 'feature-1'", model.result.Name)
	}
	if !model.quitting {
		t.Fatal("expected quitting after publish")
	}
}

func TestBrQuitKeys(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
		{Path: "/repo/.fitz/owner/repo/feature-1", Branch: "feature-1", Name: "feature-1"},
	}

	tests := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"q", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{"esc", tea.KeyMsg{Type: tea.KeyEsc}},
		{"ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newBrModel(worktrees, "root", nil)
			updated, _ := m.Update(tt.key)
			model := updated.(brModel)
			if !model.quitting {
				t.Fatalf("expected quitting after %s", tt.name)
			}
			if model.result.Action != BrActionNone {
				t.Fatalf("action = %d, want BrActionNone", model.result.Action)
			}
		})
	}
}

func TestBrDissolveAnimationCompletesRemoval(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
		{Path: "/repo/.fitz/owner/repo/feature-1", Branch: "feature-1", Name: "feature-1"},
		{Path: "/repo/.fitz/owner/repo/feature-2", Branch: "feature-2", Name: "feature-2"},
	}
	m := newBrModel(worktrees, "root", nil)
	m.onRemove = func(name string) error { return nil } // Mock removal

	// Start delete → confirm.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := updated.(brModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = updated.(brModel)

	// Send tick messages until animation completes.
	for i := 0; i < dissolveFrames; i++ {
		updated, _ = model.Update(dissolveTickMsg{})
		model = updated.(brModel)
	}

	if model.dissolving != -1 {
		t.Fatalf("dissolving = %d, want -1 after animation", model.dissolving)
	}
	if len(model.worktrees) != 2 {
		t.Fatalf("worktrees count = %d, want 2 after removal", len(model.worktrees))
	}
	// Should still be in list state, not quitting (unless that was the last non-root worktree).
	if model.state != brStateList {
		t.Fatalf("state = %d, want brStateList after removal", model.state)
	}
}

func TestBrOnlyRootShowsNoWorktrees(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
	}
	m := newBrModel(worktrees, "root", nil)

	view := m.View()
	if view == "" {
		t.Fatal("expected view to show something")
	}
	// Should show a message about no worktrees and hint to press 'n'.
	// Exact text TBD in implementation.
}

func TestBrEscFromConfirmGoesBackToList(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
		{Path: "/repo/.fitz/owner/repo/feature-1", Branch: "feature-1", Name: "feature-1"},
	}
	m := newBrModel(worktrees, "root", nil)

	// Start delete.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := updated.(brModel)

	// Press esc to cancel.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(brModel)

	if model.state != brStateList {
		t.Fatalf("state = %d, want brStateList after esc", model.state)
	}
}

func TestViewListShowsSessionBadge(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
		{Path: "/repo/.fitz/feature-a", Branch: "feature-a", Name: "feature-a"},
		{Path: "/repo/.fitz/feature-b", Branch: "feature-b", Name: "feature-b"},
	}
	sessions := map[string]session.SessionInfo{
		"/repo/.fitz/feature-a": {
			SessionID: "sess-1",
			Summary:   "Added auth middleware",
			UpdatedAt: time.Now().Add(-10 * time.Minute),
		},
	}
	m := newBrModel(worktrees, "root", sessions)

	view := m.View()

	if !strings.Contains(view, "10m ago") {
		t.Errorf("expected age in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Added auth middleware") {
		t.Errorf("expected summary in view, got:\n%s", view)
	}
}

func TestViewListNoSessionNoBadge(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
		{Path: "/repo/.fitz/feature-a", Branch: "feature-a", Name: "feature-a"},
	}
	m := newBrModel(worktrees, "root", nil)

	view := m.View()

	if strings.Contains(view, "ago") || strings.Contains(view, "⚡") {
		t.Errorf("expected no badge for worktree with no session, got:\n%s", view)
	}
}

func TestViewListWorkingBadge(t *testing.T) {
	worktrees := []worktree.WorktreeInfo{
		{Path: "/repo", Branch: "", Name: "repo"},
		{Path: "/repo/.fitz/feature-a", Branch: "feature-a", Name: "feature-a"},
	}
	sessions := map[string]session.SessionInfo{
		"/repo/.fitz/feature-a": {
			SessionID: "sess-1",
			Summary:   "Doing stuff",
			UpdatedAt: time.Now().Add(-30 * time.Second),
		},
	}
	m := newBrModel(worktrees, "root", sessions)

	view := m.View()

	if !strings.Contains(view, "⚡ working") {
		t.Errorf("expected working badge for recent session, got:\n%s", view)
	}
}
