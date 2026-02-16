package cliapp

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDissolveDKeyStartsAnimation(t *testing.T) {
	items := []TodoItem{
		{ID: "a", Text: "first", Created: time.Now()},
		{ID: "b", Text: "second", Created: time.Now()},
	}
	m := newTodoModel(items, t.TempDir()+"/todos.json")
	// Save items so RemoveTodoItem doesn't fail later.
	_ = SaveTodos(m.path, items)

	// Press 'd' on the first item.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := updated.(todoModel)

	if model.dissolving != 0 {
		t.Fatalf("dissolving = %d, want 0", model.dissolving)
	}
	if model.dissolveFrame != 1 {
		t.Fatalf("dissolveFrame = %d, want 1", model.dissolveFrame)
	}
	// Item should still be present during animation.
	if len(model.items) != 2 {
		t.Fatalf("items count = %d, want 2 (item should remain during animation)", len(model.items))
	}
	if cmd == nil {
		t.Fatal("expected tick cmd, got nil")
	}
}

func TestDissolveAnimationCompletesRemoval(t *testing.T) {
	items := []TodoItem{
		{ID: "a", Text: "first", Created: time.Now()},
		{ID: "b", Text: "second", Created: time.Now()},
	}
	path := t.TempDir() + "/todos.json"
	_ = SaveTodos(path, items)
	m := newTodoModel(items, path)

	// Press 'd' to start dissolve.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := updated.(todoModel)

	// Send tick messages until animation completes.
	for i := 0; i < dissolveFrames; i++ {
		updated, _ = model.Update(dissolveTickMsg{})
		model = updated.(todoModel)
	}

	if model.dissolving != -1 {
		t.Fatalf("dissolving = %d, want -1 after animation", model.dissolving)
	}
	if len(model.items) != 1 {
		t.Fatalf("items count = %d, want 1 after removal", len(model.items))
	}
	if model.items[0].ID != "b" {
		t.Fatalf("remaining item = %q, want 'b'", model.items[0].ID)
	}
}

func TestDissolveIgnoresKeypressesDuringAnimation(t *testing.T) {
	items := []TodoItem{
		{ID: "a", Text: "first", Created: time.Now()},
		{ID: "b", Text: "second", Created: time.Now()},
	}
	path := t.TempDir() + "/todos.json"
	_ = SaveTodos(path, items)
	m := newTodoModel(items, path)

	// Start dissolve.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := updated.(todoModel)

	// Try pressing 'j' (move down) — should be ignored.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(todoModel)

	if model.cursor != 0 {
		t.Fatalf("cursor moved during animation: got %d, want 0", model.cursor)
	}
	if model.dissolving != 0 {
		t.Fatalf("dissolving interrupted: got %d, want 0", model.dissolving)
	}
}

func TestKickoffTransitionsToPromptInput(t *testing.T) {
	items := []TodoItem{
		{ID: "a", Text: "fix the bug", Created: time.Now()},
	}
	m := newTodoModel(items, t.TempDir()+"/todos.json")

	// Select the todo item.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(todoModel)
	if model.state != stateBranchInput {
		t.Fatalf("state = %d, want stateBranchInput", model.state)
	}

	// Confirm the branch name.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(todoModel)
	if model.state != stateActionChoice {
		t.Fatalf("state = %d, want stateActionChoice", model.state)
	}

	// Move to "Create and kickoff" and select it.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(todoModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(todoModel)

	// Should transition to prompt input, not quit.
	if model.state != statePromptInput {
		t.Fatalf("state = %d, want statePromptInput", model.state)
	}
	if model.quitting {
		t.Fatal("should not be quitting yet")
	}
	// Prompt input should be pre-populated with todo text.
	if model.promptInput.Value() != "fix the bug" {
		t.Fatalf("prompt value = %q, want %q", model.promptInput.Value(), "fix the bug")
	}
}

func TestKickoffPromptInputSetsResult(t *testing.T) {
	items := []TodoItem{
		{ID: "a", Text: "fix the bug", Created: time.Now()},
	}
	m := newTodoModel(items, t.TempDir()+"/todos.json")

	// Navigate to prompt input state.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(todoModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(todoModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(todoModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(todoModel)

	// Clear and type a custom prompt.
	model.promptInput.SetValue("please fix the login bug in auth.go")

	// Press enter to confirm.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(todoModel)

	if model.result.Action != ActionKickoff {
		t.Fatalf("action = %d, want ActionKickoff", model.result.Action)
	}
	if model.result.Prompt != "please fix the login bug in auth.go" {
		t.Fatalf("prompt = %q, want custom prompt", model.result.Prompt)
	}
	if !model.quitting {
		t.Fatal("expected quitting after prompt confirmation")
	}
}

func TestKickoffPromptInputEscGoesBack(t *testing.T) {
	items := []TodoItem{
		{ID: "a", Text: "fix the bug", Created: time.Now()},
	}
	m := newTodoModel(items, t.TempDir()+"/todos.json")

	// Navigate to prompt input state.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(todoModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(todoModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	model = updated.(todoModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(todoModel)

	// Press esc to go back to action choice.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(todoModel)

	if model.state != stateActionChoice {
		t.Fatalf("state = %d, want stateActionChoice", model.state)
	}
}

func TestDissolveLastItemQuitsAfterAnimation(t *testing.T) {
	items := []TodoItem{
		{ID: "a", Text: "only", Created: time.Now()},
	}
	path := t.TempDir() + "/todos.json"
	_ = SaveTodos(path, items)
	m := newTodoModel(items, path)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model := updated.(todoModel)

	var cmd tea.Cmd
	for i := 0; i < dissolveFrames; i++ {
		updated, cmd = model.Update(dissolveTickMsg{})
		model = updated.(todoModel)
	}

	if !model.quitting {
		t.Fatal("expected quitting after last item dissolved")
	}
	// cmd should be tea.Quit.
	if cmd == nil {
		t.Fatal("expected quit cmd")
	}
}
