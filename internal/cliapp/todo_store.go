package cliapp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type TodoItem struct {
	ID      string    `json:"id"`
	Text    string    `json:"text"`
	Created time.Time `json:"created"`
}

func TodoStorePath(homeDir, owner, repo string) (string, error) {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
	}
	return filepath.Join(homeDir, ".fitz", owner, repo, "todos.json"), nil
}

func LoadTodos(path string) ([]TodoItem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read todos: %w", err)
	}

	var items []TodoItem
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse todos: %w", err)
	}
	return items, nil
}

func SaveTodos(path string, items []TodoItem) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("encode todos: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write todos: %w", err)
	}
	return nil
}

func AddTodoItem(path, text string) (TodoItem, error) {
	items, err := LoadTodos(path)
	if err != nil {
		return TodoItem{}, err
	}

	item := TodoItem{
		ID:      shortID(),
		Text:    text,
		Created: time.Now().UTC(),
	}
	items = append(items, item)

	if err := SaveTodos(path, items); err != nil {
		return TodoItem{}, err
	}
	return item, nil
}

func RemoveTodoItem(path, id string) error {
	items, err := LoadTodos(path)
	if err != nil {
		return err
	}

	idx := -1
	for i, item := range items {
		if item.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("todo %q not found", id)
	}

	items = append(items[:idx], items[idx+1:]...)
	return SaveTodos(path, items)
}

func shortID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
