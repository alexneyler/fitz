package cliapp

import (
	"context"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"fitz/internal/worktree"
)

var resolveTodoStorePath = resolveTodoPath

func TodoAdd(_ context.Context, w io.Writer, text string) error {
	storePath, err := resolveTodoStorePath()
	if err != nil {
		return err
	}

	item, err := AddTodoItem(storePath, text)
	if err != nil {
		return fmt.Errorf("add todo: %w", err)
	}

	fmt.Fprintf(w, "added: %s (%s)\n", item.Text, item.ID)
	return nil
}

func TodoList(_ context.Context, stdin io.Reader, stdout io.Writer) error {
	storePath, err := resolveTodoStorePath()
	if err != nil {
		return err
	}

	items, err := LoadTodos(storePath)
	if err != nil {
		return fmt.Errorf("load todos: %w", err)
	}

	if len(items) == 0 {
		fmt.Fprintln(stdout, "No todos.")
		return nil
	}

	model := newTodoModel(items, storePath)
	p := tea.NewProgram(model, tea.WithInput(stdin), tea.WithOutput(stdout))
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	m, ok := finalModel.(todoModel)
	if !ok {
		return nil
	}

	switch m.result.Action {
	case ActionGo:
		return BrNew(context.Background(), stdout, m.result.BranchName, "", "")
	case ActionKickoff:
		return BrNew(context.Background(), stdout, m.result.BranchName, "", m.result.Prompt)
	}

	return nil
}

func resolveTodoPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	git := worktree.ShellGit{}
	owner, repo, err := worktree.RepoID(git, cwd)
	if err != nil {
		return "", fmt.Errorf("identify repository: %w", err)
	}

	return TodoStorePath("", owner, repo)
}
