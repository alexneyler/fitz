package cliapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"fitz/internal/config"
	"fitz/internal/session"
	"fitz/internal/status"
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

var runGh = func(dir string, args ...string) (string, error) {
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		return "", errors.New("gh CLI not found in PATH (install from https://cli.github.com)")
	}
	cmd := exec.Command(ghPath, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh %v: %w: %s", args, err, stderr.String())
	}
	return stdout.String(), nil
}

var runCopilot = func(dir string, args ...string) (string, error) {
	copilotPath, err := exec.LookPath("copilot")
	if err != nil {
		return "", errors.New("copilot not found in PATH")
	}
	cmd := exec.Command(copilotPath, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("copilot %v: %w: %s", args, err, stderr.String())
	}
	return stdout.String(), nil
}

var runBackground = func(binary string, args []string, dir string) error {
	cmd := exec.Command(binary, args[1:]...)
	cmd.Dir = dir
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

var runCommand = func(binary string, args []string, dir string) error {
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w: %s", filepath.Base(binary), args, err, stderr.String())
	}
	return nil
}

var prURLPattern = regexp.MustCompile(`/pull/(\d+)`)
var prRepoPattern = regexp.MustCompile(`github\.com[/:]([^/]+)/([^/]+?)(?:\.git)?/pull/`)

// parsePRNumber extracts a pull request number from various formats:
// "42", "#42", or a full GitHub PR URL.
func parsePRNumber(input string) (int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, errors.New("empty PR identifier")
	}

	// Try full URL first.
	if m := prURLPattern.FindStringSubmatch(input); len(m) == 2 {
		return strconv.Atoi(m[1])
	}

	// Strip leading '#'.
	input = strings.TrimPrefix(input, "#")

	n, err := strconv.Atoi(input)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid PR number: %s", input)
	}
	return n, nil
}

type prInfo struct {
	HeadRefName string `json:"headRefName"`
	URL         string `json:"url"`
}

func BrCheckout(ctx context.Context, w io.Writer, pr string) error {
	prNumber, err := parsePRNumber(pr)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// When the input is a full URL, pass it directly to gh so it resolves the
	// correct repository (supports cross-repo URLs). Otherwise use the number.
	ghArg := strconv.Itoa(prNumber)
	if strings.Contains(pr, "/pull/") {
		ghArg = strings.TrimSpace(pr)
	}

	// Fetch PR metadata via gh CLI.
	out, err := runGh(cwd, "pr", "view", ghArg, "--json", "headRefName,url")
	if err != nil {
		return fmt.Errorf("get PR info: %w", err)
	}

	var info prInfo
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		return fmt.Errorf("parse PR info: %w", err)
	}
	if info.HeadRefName == "" {
		return fmt.Errorf("PR #%d has no head branch", prNumber)
	}

	git := worktree.ShellGit{}

	_, repo, _ := worktree.RepoID(git, cwd)

	// Verify the PR belongs to the current repo.
	if m := prRepoPattern.FindStringSubmatch(info.URL); len(m) == 3 {
		prOwner, prRepo := strings.ToLower(m[1]), strings.ToLower(m[2])
		curOwner, curRepo, _ := worktree.RepoID(git, cwd)
		curOwner, curRepo = strings.ToLower(curOwner), strings.ToLower(curRepo)
		if curOwner != "" && (prOwner != curOwner || prRepo != curRepo) {
			return fmt.Errorf("PR #%d belongs to %s/%s but you are in %s/%s; run from a %s/%s worktree",
				prNumber, m[1], m[2], curOwner, curRepo, m[1], m[2])
		}
	}

	// Fetch via the pull/<N>/head ref — this always works for PRs in the
	// current repo, including fork PRs where the branch isn't on origin.
	prRef := fmt.Sprintf("pull/%d/head", prNumber)
	_, err = git.Run(cwd, "fetch", "origin", prRef)
	if err != nil {
		return fmt.Errorf("fetch PR #%d: %w", prNumber, err)
	}

	// Create a worktree with a local branch starting at the fetched PR head.
	// Use CreateForce (-B) so the branch is reset if it already exists locally.
	mgr := &worktree.Manager{Git: git}
	path, err := mgr.CreateForce(cwd, info.HeadRefName, "FETCH_HEAD")
	if err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}

	// Store PR URL so br list shows it.
	if statusPath, err := resolveAgentStatusPath(); err == nil {
		_, _ = status.SetPR(statusPath, info.HeadRefName, info.URL)
	}

	cfg := loadEffectiveConfig(cwd)

	fmt.Fprintf(w, "checked out PR #%d (%s)\n", prNumber, info.HeadRefName)
	return launchBranchInteractive(w, path, info.HeadRefName, repo, cfg)
}

func BrNew(ctx context.Context, w io.Writer, name, base, prompt string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	git := worktree.ShellGit{}
	mgr := &worktree.Manager{Git: git}

	path, err := mgr.Create(cwd, name, base)
	if err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}

	_, repo, _ := worktree.RepoID(git, cwd)
	cfg := loadEffectiveConfig(cwd)

	if prompt != "" {
		copilotPath, err := lookPath("copilot")
		if err != nil {
			return errors.New("copilot not found in PATH")
		}
		args := append(copilotBaseArgs(cfg), "--yolo", "-p", prompt)
		if err := runBackground(copilotPath, args, path); err != nil {
			return fmt.Errorf("start copilot: %w", err)
		}
		fmt.Fprintf(w, "worktree created: %s\n", name)
		fmt.Fprintf(w, "copilot is working on it in the background\n")
		fmt.Fprintf(w, "run `fitz br go %s` to navigate to it\n", name)
		return nil
	}

	return launchBranchInteractive(w, path, name, repo, cfg)
}

// copilotConfigDir returns the Copilot configuration directory (~/.copilot).
var copilotConfigDir = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".copilot")
}

// loadEffectiveConfig loads the merged config for the repo at dir.
// Non-fatal: returns defaults on any error.
var loadEffectiveConfig = func(dir string) config.Config {
	git := worktree.ShellGit{}
	owner, repo, _ := worktree.RepoID(git, dir)
	cfg, err := config.LoadEffective("", owner, repo)
	if err != nil {
		return config.DefaultConfig()
	}
	return cfg
}

// copilotBaseArgs returns the base args for invoking copilot with model if configured.
func copilotBaseArgs(cfg config.Config) []string {
	args := []string{"copilot"}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	return args
}

func launchBranchInteractive(w io.Writer, path, name, repo string, cfg config.Config) error {
	mode := strings.TrimSpace(cfg.BranchOpenMode)
	if mode == "" {
		mode = "zellij"
	}

	switch mode {
	case "standard":
		copilotPath, err := lookPath("copilot")
		if err != nil {
			return errors.New("copilot not found in PATH")
		}
		if err := os.Chdir(path); err != nil {
			return fmt.Errorf("cd to worktree: %w", err)
		}
		return runExec(copilotPath, copilotBaseArgs(cfg), os.Environ())
	case "zellij":
		return openBranchInZellij(w, path, name, repo, cfg)
	default:
		return fmt.Errorf("invalid branch-open-mode: %s (valid values: zellij, standard)", mode)
	}
}

func openBranchInZellij(w io.Writer, path, name, repo string, cfg config.Config) error {
	if _, err := lookPath("copilot"); err != nil {
		return errors.New("copilot not found in PATH")
	}
	zellijPath, err := lookPath("zellij")
	if err != nil {
		return errors.New("zellij not found in PATH")
	}
	inZellij := strings.TrimSpace(os.Getenv("ZELLIJ")) != ""
	sessionName := strings.TrimSpace(os.Getenv("ZELLIJ_SESSION_NAME"))
	if !inZellij && sessionName == "" {
		return errors.New("zellij mode requires an active zellij session; run from inside zellij or set branch-open-mode=standard")
	}

	splitDirection, err := zellijSplitDirection(cfg)
	if err != nil {
		return err
	}

	layoutPath, err := writeZellijBranchLayout(copilotBaseArgs(cfg), splitDirection)
	if err != nil {
		return fmt.Errorf("create zellij layout: %w", err)
	}
	defer os.Remove(layoutPath)

	args := []string{}
	if sessionName != "" {
		args = append(args, "--session", sessionName)
	}
	tabName := name
	if repo != "" {
		tabName = repo + "/" + name
	}
	args = append(args, "action", "new-tab", "--name", tabName, "--cwd", path, "--layout", layoutPath)
	if err := runCommand(zellijPath, args, path); err != nil {
		return fmt.Errorf("open zellij tab: %w", err)
	}

	fmt.Fprintf(w, "worktree created: %s\n", name)
	fmt.Fprintln(w, "opened in zellij")
	return nil
}

func zellijSplitDirection(cfg config.Config) (string, error) {
	layout := strings.TrimSpace(cfg.BranchZellijLayout)
	if layout == "" {
		layout = "vertical"
	}
	if layout != "vertical" && layout != "horizontal" {
		return "", fmt.Errorf("invalid branch-zellij-layout: %s (valid values: vertical, horizontal)", layout)
	}
	return layout, nil
}

func writeZellijBranchLayout(copilotArgs []string, splitDirection string) (string, error) {
	file, err := os.CreateTemp("", "fitz-zellij-*.kdl")
	if err != nil {
		return "", err
	}
	layoutPath := file.Name()
	if _, err := file.WriteString(zellijBranchLayout(copilotArgs, splitDirection)); err != nil {
		file.Close()
		_ = os.Remove(layoutPath)
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(layoutPath)
		return "", err
	}
	return layoutPath, nil
}

func zellijBranchLayout(copilotArgs []string, splitDirection string) string {
	if len(copilotArgs) == 0 {
		copilotArgs = []string{"copilot"}
	}

	argsLine := ""
	if len(copilotArgs) > 1 {
		quotedArgs := make([]string, 0, len(copilotArgs)-1)
		for _, arg := range copilotArgs[1:] {
			quotedArgs = append(quotedArgs, strconv.Quote(arg))
		}
		argsLine = fmt.Sprintf("            args %s\n", strings.Join(quotedArgs, " "))
	}

	return fmt.Sprintf(`layout {
    pane size=1 borderless=true {
        plugin location="tab-bar"
    }
    pane split_direction=%s {
        pane command=%s {
%s        }
        pane
    }
    pane size=1 borderless=true {
        plugin location="status-bar"
    }
}
`, strconv.Quote(splitDirection), strconv.Quote(copilotArgs[0]), argsLine)
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

	cfg := loadEffectiveConfig(cwd)
	args := copilotBaseArgs(cfg)
	if configDir := copilotConfigDir(); configDir != "" {
		if sessionID, err := session.FindLatestSession(configDir, path); err == nil && sessionID != "" {
			args = append(args, "--resume", sessionID)
		}
	}

	return runExec(copilotPath, args, os.Environ())
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

	fmt.Fprintf(w, "removed worktree and branch: %s\n", name)
	return nil
}

func BrRemoveAll(ctx context.Context, w io.Writer, force bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	git := worktree.ShellGit{}
	mgr := &worktree.Manager{Git: git}

	removed, err := mgr.RemoveAll(cwd, force)
	if err != nil {
		return fmt.Errorf("remove worktrees: %w", err)
	}

	if len(removed) == 0 {
		fmt.Fprintln(w, "no worktrees to remove")
		return nil
	}

	for _, name := range removed {
		fmt.Fprintf(w, "removed worktree and branch: %s\n", name)
	}
	return nil
}

func BrList(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
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

	list, err := mgr.List(cwd)
	if err != nil {
		return fmt.Errorf("list worktrees: %w", err)
	}

	// Collect worktree paths and look up session info in a single pass.
	cwds := make([]string, len(list))
	for i, wt := range list {
		cwds[i] = wt.Path
	}
	sessions := map[string]session.SessionInfo{}
	if configDir := copilotConfigDir(); configDir != "" {
		if s, err := session.FindAllSessionInfos(configDir, cwds); err == nil {
			sessions = s
		}
	}
	statuses := map[string]status.BranchStatus{}
	if statusPath, err := resolveAgentStatusPath(); err == nil {
		if s, err := status.Load(statusPath); err == nil {
			statuses = s
		}
	}

	// Launch interactive TUI.
	model := newBrModel(list, current, sessions)
	model.statuses = statuses
	model.onRemove = func(name string) error {
		return mgr.Remove(cwd, name, false)
	}

	p := tea.NewProgram(model, tea.WithInput(stdin), tea.WithOutput(stdout))
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	m, ok := finalModel.(brModel)
	if !ok {
		return nil
	}

	// Dispatch result actions.
	switch m.result.Action {
	case BrActionGo:
		return BrGo(ctx, stdout, m.result.Name)
	case BrActionNew:
		return BrNew(ctx, stdout, m.result.BranchName, "", "")
	case BrActionNewKickoff:
		return BrNew(ctx, stdout, m.result.BranchName, "", m.result.Prompt)
	case BrActionPublish:
		return BrPublish(ctx, stdout, m.result.Name)
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

func BrPublish(ctx context.Context, w io.Writer, name string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	git := worktree.ShellGit{}

	// If a worktree name was given, resolve its path and operate there.
	if name != "" {
		mgr := &worktree.Manager{Git: git}
		cwd, err = mgr.Path(cwd, name)
		if err != nil {
			return fmt.Errorf("get worktree path: %w", err)
		}
	}

	branch, err := git.Run(cwd, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}
	branch = strings.TrimSpace(branch)

	defaultBranch := detectDefaultBranch(git, cwd)
	if branch == defaultBranch || branch == "HEAD" {
		return fmt.Errorf("cannot publish from %s; switch to a feature branch first", branch)
	}

	_, err = git.Run(cwd, "push", "-u", "origin", branch)
	if err != nil {
		return fmt.Errorf("push branch: %w", err)
	}
	fmt.Fprintf(w, "pushed %s to origin\n", branch)

	cfg := loadEffectiveConfig(cwd)
	copilotArgs := append(copilotBaseArgs(cfg)[1:], "--yolo", "-p", "Create a PR for this branch")
	output, err := runCopilot(cwd, copilotArgs...)
	if err != nil {
		return fmt.Errorf("create pull request: %w", err)
	}
	output = strings.TrimSpace(output)
	fmt.Fprintf(w, "%s\n", output)

	return nil
}

// detectDefaultBranch returns the repo's default branch by inspecting
// origin/HEAD. Falls back to "main" if the ref is not set.
func detectDefaultBranch(git worktree.ShellGit, dir string) string {
	out, err := git.Run(dir, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		ref := strings.TrimSpace(out)
		if i := strings.LastIndex(ref, "/"); i >= 0 {
			return ref[i+1:]
		}
	}
	return "main"
}
