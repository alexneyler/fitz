package worktree

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Manager struct {
	Git     GitRunner
	HomeDir string
}

type WorktreeInfo struct {
	Path   string
	Branch string
	Name   string
	Bare   bool
}

func ValidateName(name string) error {
	if name == "" {
		return errors.New("worktree name cannot be empty")
	}

	trimmed := strings.TrimSpace(name)
	if len(trimmed) == 0 {
		return errors.New("worktree name cannot be only whitespace")
	}

	if strings.HasPrefix(name, "-") {
		return errors.New("worktree name cannot start with dash")
	}

	if strings.Contains(name, "/") {
		return errors.New("worktree name cannot contain forward slash")
	}

	if strings.Contains(name, "\\") {
		return errors.New("worktree name cannot contain backslash")
	}

	if strings.Contains(name, "..") {
		return errors.New("worktree name cannot contain double dot")
	}

	return nil
}

func (m *Manager) Create(dir, name, base string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	owner, repo, err := RepoID(m.Git, dir)
	if err != nil {
		return "", err
	}

	homeDir := m.HomeDir
	if homeDir == "" {
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
	}

	path := filepath.Join(homeDir, ".fitz", owner, repo, name)

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	expectedPrefix := filepath.Join(homeDir, ".fitz", owner, repo)
	cleanAbsPath := filepath.Clean(absPath)
	cleanPrefix := filepath.Clean(expectedPrefix)

	if !strings.HasPrefix(cleanAbsPath+string(filepath.Separator), cleanPrefix+string(filepath.Separator)) {
		return "", errors.New("worktree path would escape .fitz directory")
	}

	args := []string{"worktree", "add", path, "-b", name}
	if base != "" {
		args = append(args, base)
	}

	_, err = m.Git.Run(dir, args...)
	if err != nil {
		return "", err
	}

	return path, nil
}

func (m *Manager) Remove(dir, name string, force bool) error {
	if err := ValidateName(name); err != nil {
		return err
	}

	path, err := m.Path(dir, name)
	if err != nil {
		return err
	}

	args := []string{"worktree", "remove", path}
	if force {
		args = append(args, "--force")
	}

	_, err = m.Git.Run(dir, args...)
	if err != nil {
		return err
	}

	_, err = m.Git.Run(dir, "worktree", "prune")
	return err
}

func (m *Manager) List(dir string) ([]WorktreeInfo, error) {
	output, err := m.Git.Run(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	return parseWorktreeList(output), nil
}

func (m *Manager) Path(dir, name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}

	owner, repo, err := RepoID(m.Git, dir)
	if err != nil {
		return "", err
	}

	homeDir := m.HomeDir
	if homeDir == "" {
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
	}

	path := filepath.Join(homeDir, ".fitz", owner, repo, name)

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	expectedPrefix := filepath.Join(homeDir, ".fitz", owner, repo)
	cleanAbsPath := filepath.Clean(absPath)
	cleanPrefix := filepath.Clean(expectedPrefix)

	if !strings.HasPrefix(cleanAbsPath+string(filepath.Separator), cleanPrefix+string(filepath.Separator)) {
		return "", errors.New("worktree path would escape .fitz directory")
	}

	return path, nil
}

func (m *Manager) Current(dir string) (string, error) {
	isWt, err := IsWorktree(m.Git, dir)
	if err != nil {
		return "", err
	}

	if !isWt {
		return "root", nil
	}

	root, err := GitRoot(m.Git, dir)
	if err != nil {
		return "", err
	}

	list, err := m.List(root)
	if err != nil {
		return "", err
	}

	currentPath, err := filepath.Abs(root)
	if err != nil {
		currentPath = root
	}

	for _, wt := range list {
		wtPath, err := filepath.Abs(wt.Path)
		if err != nil {
			wtPath = wt.Path
		}
		if wtPath == currentPath {
			return wt.Name, nil
		}
	}

	return "root", nil
}

func parseWorktreeList(output string) []WorktreeInfo {
	var list []WorktreeInfo
	var current WorktreeInfo

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Path != "" {
				list = append(list, current)
				current = WorktreeInfo{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			current.Path = path
			current.Name = filepath.Base(path)
		} else if strings.HasPrefix(line, "branch refs/heads/") {
			branch := strings.TrimPrefix(line, "branch refs/heads/")
			current.Branch = branch
		} else if line == "bare" {
			current.Bare = true
		}
	}

	if current.Path != "" {
		list = append(list, current)
	}

	return list
}
