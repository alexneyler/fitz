package cliapp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"fitz/internal/worktree"
)

var execCommand = syscall.Exec
var lookPath = exec.LookPath

var runExec = func(binary string, args []string, env []string) error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command(binary, args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env
		return cmd.Run()
	}
	return syscall.Exec(binary, args, env)
}

var runBackground = func(binary string, args []string, dir string) error {
	cmd := exec.Command(binary, args[1:]...)
	cmd.Dir = dir
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func BrNew(ctx context.Context, w io.Writer, name, prompt string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	git := worktree.ShellGit{}
	mgr := &worktree.Manager{Git: git}

	path, err := mgr.Create(cwd, name, "")
	if err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}

	copilotPath, err := lookPath("copilot")
	if err != nil {
		return errors.New("copilot not found in PATH")
	}

	if prompt != "" {
		args := []string{"copilot", "--yolo", "-p", prompt}
		if err := runBackground(copilotPath, args, path); err != nil {
			return fmt.Errorf("start copilot: %w", err)
		}
		fmt.Fprintf(w, "worktree created: %s\n", name)
		fmt.Fprintf(w, "copilot is working on it in the background\n")
		fmt.Fprintf(w, "run `fitz br go %s` to navigate to it\n", name)
		return nil
	}

	if err := os.Chdir(path); err != nil {
		return fmt.Errorf("cd to worktree: %w", err)
	}

	return runExec(copilotPath, []string{"copilot"}, os.Environ())
}

func BrGo(ctx context.Context, w io.Writer, name string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	git := worktree.ShellGit{}
	mgr := &worktree.Manager{Git: git}

	path, err := mgr.Path(cwd, name)
	if err != nil {
		return fmt.Errorf("get worktree path: %w", err)
	}

	copilotPath, err := lookPath("copilot")
	if err != nil {
		return errors.New("copilot not found in PATH")
	}

	if err := os.Chdir(path); err != nil {
		return fmt.Errorf("cd to worktree: %w", err)
	}

	return runExec(copilotPath, []string{"copilot", "--continue"}, os.Environ())
}

func BrRemove(ctx context.Context, w io.Writer, name string, force bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	git := worktree.ShellGit{}
	mgr := &worktree.Manager{Git: git}

	if err := mgr.Remove(cwd, name, force); err != nil {
		return fmt.Errorf("remove worktree: %w", err)
	}

	fmt.Fprintf(w, "removed worktree: %s\n", name)
	return nil
}

func BrList(ctx context.Context, w io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	git := worktree.ShellGit{}
	mgr := &worktree.Manager{Git: git}

	list, err := mgr.List(cwd)
	if err != nil {
		return fmt.Errorf("list worktrees: %w", err)
	}

	for _, wt := range list {
		branch := wt.Branch
		if branch == "" {
			branch = "(detached)"
		}
		fmt.Fprintf(w, "%s\t%s\n", wt.Name, branch)
	}

	return nil
}

func BrCurrent(ctx context.Context, w io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	git := worktree.ShellGit{}
	mgr := &worktree.Manager{Git: git}

	current, err := mgr.Current(cwd)
	if err != nil {
		return fmt.Errorf("get current worktree: %w", err)
	}

	fmt.Fprintln(w, current)
	return nil
}

func BrCd(ctx context.Context, w io.Writer, name string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	git := worktree.ShellGit{}
	mgr := &worktree.Manager{Git: git}

	path, err := mgr.Path(cwd, name)
	if err != nil {
		return fmt.Errorf("get worktree path: %w", err)
	}

	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("worktree not found: %s", name)
	}

	fmt.Fprintln(w, path)
	return nil
}
