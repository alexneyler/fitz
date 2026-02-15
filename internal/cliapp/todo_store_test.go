package cliapp

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadTodosEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")

	todos, err := LoadTodos(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(todos) != 0 {
		t.Fatalf("expected empty list, got %d", len(todos))
	}
}

func TestSaveAndLoadTodos(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")

	items := []TodoItem{
		{ID: "abc", Text: "first todo", Created: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "def", Text: "second todo", Created: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
	}

	if err := SaveTodos(path, items); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadTodos(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 items, got %d", len(loaded))
	}
	if loaded[0].ID != "abc" || loaded[0].Text != "first todo" {
		t.Fatalf("item 0 = %+v", loaded[0])
	}
	if loaded[1].ID != "def" || loaded[1].Text != "second todo" {
		t.Fatalf("item 1 = %+v", loaded[1])
	}
}

func TestAddTodoItem(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")

	item, err := AddTodoItem(path, "my new todo")
	if err != nil {
		t.Fatalf("add error: %v", err)
	}
	if item.Text != "my new todo" {
		t.Fatalf("text = %q", item.Text)
	}
	if item.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	loaded, err := LoadTodos(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 item, got %d", len(loaded))
	}
}

func TestRemoveTodoItem(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")

	item1, _ := AddTodoItem(path, "first")
	_, _ = AddTodoItem(path, "second")

	if err := RemoveTodoItem(path, item1.ID); err != nil {
		t.Fatalf("remove error: %v", err)
	}

	loaded, err := LoadTodos(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 item, got %d", len(loaded))
	}
	if loaded[0].Text != "second" {
		t.Fatalf("remaining item = %q, want 'second'", loaded[0].Text)
	}
}

func TestRemoveTodoItemNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "todos.json")

	_, _ = AddTodoItem(path, "something")

	err := RemoveTodoItem(path, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

func TestSaveTodosCreatesDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "todos.json")

	items := []TodoItem{{ID: "x", Text: "hello", Created: time.Now()}}
	if err := SaveTodos(nested, items); err != nil {
		t.Fatalf("save error: %v", err)
	}

	if _, err := os.Stat(nested); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestTodoStorePath(t *testing.T) {
	path, err := TodoStorePath("/fakehome", "myowner", "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join("/fakehome", ".fitz", "myowner", "myrepo", "todos.json")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}
