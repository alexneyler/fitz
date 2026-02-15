package worktree

import (
	"errors"
	"fmt"
	"io"
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

	if strings.Contains(name, "\\") {
		return errors.New("worktree name cannot contain backslash")
	}

	if strings.Contains(name, "..") {
		return errors.New("worktree name cannot contain double dot")
	}

	return nil
}

// DirName converts a worktree/branch name to a directory-safe name by
// replacing forward slashes with dashes.
func DirName(name string) string {
	return strings.ReplaceAll(name, "/", "-")
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

	dirName := DirName(name)
	path := filepath.Join(homeDir, ".fitz", owner, repo, dirName)

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
	if err != nil {
		return err
	}

	_, err = m.Git.Run(dir, "branch", "-D", name)
	return err
}

// RemoveAll removes all worktrees (except root) and their branches.
// Returns the names of removed worktrees.
func (m *Manager) RemoveAll(dir string, force bool) ([]string, error) {
	list, err := m.List(dir)
	if err != nil {
		return nil, err
	}

	var removed []string
	for i, wt := range list {
		if i == 0 {
			continue // skip root
		}
		name := wt.Branch
		if name == "" {
			name = wt.Name
		}

		removeArgs := []string{"worktree", "remove", wt.Path}
		if force {
			removeArgs = append(removeArgs, "--force")
		}
		if _, err := m.Git.Run(dir, removeArgs...); err != nil {
			return removed, fmt.Errorf("remove worktree %s: %w", name, err)
		}

		if wt.Branch != "" {
			if _, err := m.Git.Run(dir, "branch", "-D", wt.Branch); err != nil {
				return removed, fmt.Errorf("delete branch %s: %w", wt.Branch, err)
			}
		}

		removed = append(removed, name)
	}

	if len(removed) > 0 {
		_, _ = m.Git.Run(dir, "worktree", "prune")
	}

	return removed, nil
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

	path := filepath.Join(homeDir, ".fitz", owner, repo, DirName(name))

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

// FormatList writes the worktree list to w, highlighting the current worktree
// with a blue "* " prefix. The first entry is always labeled "root". Others
// show their branch name (or directory name if detached). current should be
// "root" or a worktree Name (directory basename) as returned by Current().
func FormatList(w io.Writer, list []WorktreeInfo, current string) {
	const blue = "\x1b[34m"
	const reset = "\x1b[0m"

	for i, wt := range list {
		name := wt.Branch
		if i == 0 {
			name = "root"
		} else if name == "" {
			name = wt.Name
		}

		isCurrent := (i == 0 && current == "root") || (i > 0 && current == wt.Name)
		if isCurrent {
			fmt.Fprintf(w, "%s* %s%s\n", blue, name, reset)
		} else {
			fmt.Fprintf(w, "  %s\n", name)
		}
	}
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
