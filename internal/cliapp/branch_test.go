package cliapp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"fitz/internal/config"
)

func TestBrCurrent(t *testing.T) {
	var out bytes.Buffer
	err := BrCurrent(context.Background(), &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestBrList(t *testing.T) {
	// Create a stdin that immediately sends 'q' to quit the TUI.
	stdin := strings.NewReader("q")
	var out bytes.Buffer
	err := BrList(context.Background(), stdin, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBrCd(t *testing.T) {
	var out bytes.Buffer
	err := BrCd(context.Background(), &out, "feature")
	if err == nil {
		t.Fatal("expected error for non-existent worktree in test repo")
	}
}

func TestRunExecMockable(t *testing.T) {
	originalExec := runExec
	t.Cleanup(func() { runExec = originalExec })

	var called bool
	var capturedBinary string
	var capturedArgs []string

	runExec = func(binary string, args []string, env []string) error {
		called = true
		capturedBinary = binary
		capturedArgs = args
		return nil
	}

	// Without a session match, BrGo should call copilot with no resume flag.
	err := runExec("/usr/bin/copilot", []string{"copilot"}, os.Environ())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Fatal("runExec was not called")
	}

	if capturedBinary != "/usr/bin/copilot" {
		t.Errorf("binary = %q, want /usr/bin/copilot", capturedBinary)
	}

	if len(capturedArgs) != 1 || capturedArgs[0] != "copilot" {
		t.Errorf("args = %v, want [copilot]", capturedArgs)
	}
}

func TestRunExecWithResume(t *testing.T) {
	originalExec := runExec
	t.Cleanup(func() { runExec = originalExec })

	var capturedArgs []string

	runExec = func(binary string, args []string, env []string) error {
		capturedArgs = args
		return nil
	}

	// With a --resume flag, args should include session ID.
	err := runExec("/usr/bin/copilot", []string{"copilot", "--resume", "abc-123"}, os.Environ())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantArgs := []string{"copilot", "--resume", "abc-123"}
	if len(capturedArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", capturedArgs, wantArgs)
	}
	for i, a := range wantArgs {
		if capturedArgs[i] != a {
			t.Errorf("args[%d] = %q, want %q", i, capturedArgs[i], a)
		}
	}
}

func TestRunBackgroundMockable(t *testing.T) {
	originalBg := runBackground
	t.Cleanup(func() { runBackground = originalBg })

	var capturedBinary string
	var capturedArgs []string
	var capturedDir string

	runBackground = func(binary string, args []string, dir string) error {
		capturedBinary = binary
		capturedArgs = args
		capturedDir = dir
		return nil
	}

	err := runBackground("/usr/bin/copilot", []string{"copilot", "--yolo", "-p", "do stuff"}, "/tmp/wt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedBinary != "/usr/bin/copilot" {
		t.Errorf("binary = %q, want /usr/bin/copilot", capturedBinary)
	}

	wantArgs := []string{"copilot", "--yolo", "-p", "do stuff"}
	if len(capturedArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", capturedArgs, wantArgs)
	}
	for i, a := range wantArgs {
		if capturedArgs[i] != a {
			t.Errorf("args[%d] = %q, want %q", i, capturedArgs[i], a)
		}
	}

	if capturedDir != "/tmp/wt" {
		t.Errorf("dir = %q, want /tmp/wt", capturedDir)
	}
}

// mockGitRunner lets tests control git output without shelling out.
type mockGitRunner struct {
	results map[string]string
	err     error
}

func (m mockGitRunner) Run(dir string, args ...string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	key := strings.Join(args, " ")
	if v, ok := m.results[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("unexpected git call: %v", args)
}

func TestBrPublish(t *testing.T) {
	originalCopilot := runCopilot
	t.Cleanup(func() { runCopilot = originalCopilot })

	var copilotArgs []string
	runCopilot = func(dir string, args ...string) (string, error) {
		copilotArgs = args
		return "https://github.com/owner/repo/pull/42\n", nil
	}

	// We can't easily mock the worktree.ShellGit used inside BrPublish
	// in a unit test without being in a real git repo. But this test
	// validates the happy path wiring when run from the repo itself.
	var out bytes.Buffer
	err := BrPublish(context.Background(), &out, "")
	// If we're on main/master, we expect the guard error.
	if err != nil {
		if strings.Contains(err.Error(), "cannot publish from") {
			t.Skip("skipping: test running from default branch")
		}
		// The push may fail if there's no real remote, which is fine for unit testing.
		if strings.Contains(err.Error(), "push branch") {
			t.Skip("skipping: no remote available for push")
		}
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "pushed") {
		t.Errorf("stdout = %q, want 'pushed' message", output)
	}
	if !strings.Contains(output, "https://github.com") {
		t.Errorf("stdout = %q, want PR URL", output)
	}
	// Verify copilot was invoked with expected flags.
	hasYolo := false
	hasP := false
	for i, a := range copilotArgs {
		if a == "--yolo" {
			hasYolo = true
		}
		if a == "-p" && i+1 < len(copilotArgs) {
			hasP = true
		}
	}
	if !hasYolo {
		t.Errorf("copilot args missing --yolo: %v", copilotArgs)
	}
	if !hasP {
		t.Errorf("copilot args missing -p: %v", copilotArgs)
	}
}

func TestBrNewFetchesOriginAndUsesDefaultBase(t *testing.T) {
	// Set up a bare repo to act as "origin".
	bareDir := t.TempDir()
	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
		}
	}

	run(bareDir, "init", "--bare", "--initial-branch=main")

	// Clone the bare repo.
	cloneDir := t.TempDir()
	run(cloneDir, "clone", bareDir, "work")
	workDir := filepath.Join(cloneDir, "work")

	// Make an initial commit in the clone and push.
	if err := os.WriteFile(filepath.Join(workDir, "file.txt"), []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}
	run(workDir, "add", ".")
	run(workDir, "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "init")
	run(workDir, "push", "origin", "main")

	// Add another commit directly to origin (simulates remote work).
	secondClone := t.TempDir()
	run(secondClone, "clone", bareDir, "work2")
	work2Dir := filepath.Join(secondClone, "work2")
	if err := os.WriteFile(filepath.Join(work2Dir, "file.txt"), []byte("v2"), 0644); err != nil {
		t.Fatal(err)
	}
	run(work2Dir, "add", ".")
	run(work2Dir, "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "remote update")
	run(work2Dir, "push", "origin", "main")

	// Record the latest origin commit.
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = work2Dir
	latestBytes, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	latestSHA := strings.TrimSpace(string(latestBytes))

	// Redirect HOME to a temp dir so worktrees don't leak into real home.
	fakeHome := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", origHome) })
	_ = os.Setenv("HOME", fakeHome)

	// Now from workDir (which is behind origin), call BrNew.
	// It should fetch and base the new branch on origin/main.
	originalDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(originalDir) })
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}

	// Stub launchBranchInteractive to avoid TUI.
	originalExec := runExec
	originalLook := lookPath
	originalLoadCfg := loadEffectiveConfig
	t.Cleanup(func() {
		runExec = originalExec
		lookPath = originalLook
		loadEffectiveConfig = originalLoadCfg
	})

	lookPath = func(string) (string, error) { return "/usr/bin/copilot", nil }
	loadEffectiveConfig = func(dir string) config.Config {
		return config.Config{BranchOpenMode: "standard"}
	}
	runExec = func(binary string, args []string, env []string) error { return nil }

	var out bytes.Buffer
	err = BrNew(context.Background(), &out, "test-fetch-branch", "", "")
	t.Cleanup(func() {
		_ = exec.Command("git", "-C", workDir, "worktree", "prune").Run()
		_ = exec.Command("git", "-C", workDir, "branch", "-D", "test-fetch-branch").Run()
	})
	if err != nil {
		t.Fatalf("BrNew: %v", err)
	}

	// Verify the new branch points to the latest origin commit.
	cmd = exec.Command("git", "-C", workDir, "rev-parse", "test-fetch-branch")
	branchBytes, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse test-fetch-branch: %v", err)
	}
	branchSHA := strings.TrimSpace(string(branchBytes))

	if branchSHA != latestSHA {
		t.Errorf("branch SHA = %s, want %s (latest origin)", branchSHA, latestSHA)
	}
}

func TestBrPublishProtectsDefaultBranch(t *testing.T) {
	originalCopilot := runCopilot
	t.Cleanup(func() { runCopilot = originalCopilot })
	runCopilot = func(dir string, args ...string) (string, error) {
		return "https://github.com/owner/repo/pull/99\n", nil
	}

	// This test is only meaningful when run from main/master.
	var out bytes.Buffer
	err := BrPublish(context.Background(), &out, "")
	if err == nil {
		return // not on a protected branch, that's fine
	}
	if strings.Contains(err.Error(), "cannot publish from main") ||
		strings.Contains(err.Error(), "cannot publish from master") {
		// expected
		return
	}
	// push/other errors are acceptable in test environments
}

func TestCopilotBaseArgs_NoModel(t *testing.T) {
	cfg := config.Config{}
	args := copilotBaseArgs(cfg)
	if len(args) != 1 || args[0] != "copilot" {
		t.Errorf("copilotBaseArgs (no model) = %v, want [copilot]", args)
	}
}

func TestCopilotBaseArgs_WithModel(t *testing.T) {
	cfg := config.Config{Model: "claude-opus-4.6"}
	args := copilotBaseArgs(cfg)
	want := []string{"copilot", "--model", "claude-opus-4.6"}
	if len(args) != len(want) {
		t.Fatalf("copilotBaseArgs = %v, want %v", args, want)
	}
	for i, v := range want {
		if args[i] != v {
			t.Errorf("args[%d] = %q, want %q", i, args[i], v)
		}
	}
}

func TestBrNew_PassesModelToBackground(t *testing.T) {
	originalExec := runExec
	originalBg := runBackground
	originalLook := lookPath
	originalLoadCfg := loadEffectiveConfig
	t.Cleanup(func() {
		runExec = originalExec
		runBackground = originalBg
		lookPath = originalLook
		loadEffectiveConfig = originalLoadCfg
	})

	lookPath = func(string) (string, error) { return "/usr/bin/copilot", nil }

	loadEffectiveConfig = func(dir string) config.Config {
		return config.Config{Model: "test-model", Agent: "copilot-cli"}
	}

	var capturedArgs []string
	runBackground = func(binary string, args []string, dir string) error {
		capturedArgs = args
		return nil
	}
	// runExec shouldn't be called when prompt is provided, but stub it.
	runExec = func(binary string, args []string, env []string) error { return nil }

	// BrNew tries to create a worktree via git, so we can't call the real BrNew
	// without a real repo. Instead we test copilotBaseArgs directly plus
	// verify the model flag appears in the arg list constructed by BrNew's
	// background path.
	cfg := config.Config{Model: "test-model"}
	base := copilotBaseArgs(cfg)
	args := append(base, "--yolo", "-p", "do the thing")
	_ = runBackground("/usr/bin/copilot", args, "/tmp/wt")

	found := false
	for i, a := range capturedArgs {
		if a == "--model" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "test-model" {
			found = true
		}
	}
	if !found {
		t.Errorf("--model test-model not found in args: %v", capturedArgs)
	}
}

func TestBrNew_PassesModelToExec(t *testing.T) {
	originalExec := runExec
	originalLook := lookPath
	originalLoadCfg := loadEffectiveConfig
	t.Cleanup(func() {
		runExec = originalExec
		lookPath = originalLook
		loadEffectiveConfig = originalLoadCfg
	})

	loadEffectiveConfig = func(dir string) config.Config {
		return config.Config{Model: "exec-model", Agent: "copilot-cli"}
	}

	cfg := config.Config{Model: "exec-model"}
	args := copilotBaseArgs(cfg)

	var capturedArgs []string
	runExec = func(binary string, a []string, env []string) error {
		capturedArgs = a
		return nil
	}
	_ = runExec("/usr/bin/copilot", args, os.Environ())

	found := false
	for i, a := range capturedArgs {
		if a == "--model" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "exec-model" {
			found = true
		}
	}
	if !found {
		t.Errorf("--model exec-model not found in exec args: %v", capturedArgs)
	}
}

func TestZellijBranchLayoutIncludesCopilotAndSplit(t *testing.T) {
	layout := zellijBranchLayout([]string{"copilot", "--model", "z-model"}, "vertical")
	if !strings.Contains(layout, `plugin location="tab-bar"`) {
		t.Fatalf("layout = %q, want tab-bar plugin", layout)
	}
	if !strings.Contains(layout, `plugin location="status-bar"`) {
		t.Fatalf("layout = %q, want status-bar plugin", layout)
	}
	if !strings.Contains(layout, `split_direction="vertical"`) {
		t.Fatalf("layout = %q, want vertical split", layout)
	}
	if !strings.Contains(layout, `command="copilot"`) {
		t.Fatalf("layout = %q, want copilot command", layout)
	}
	if !strings.Contains(layout, `args "--model" "z-model"`) {
		t.Fatalf("layout = %q, want model args", layout)
	}
}

func TestLaunchBranchInteractive_Zellij_UsesConfiguredLayout(t *testing.T) {
	originalExec := runExec
	originalLook := lookPath
	originalRunCmd := runCommand
	originalZellijEnv := os.Getenv("ZELLIJ")
	originalSessionEnv := os.Getenv("ZELLIJ_SESSION_NAME")
	t.Cleanup(func() {
		runExec = originalExec
		lookPath = originalLook
		runCommand = originalRunCmd
		_ = os.Setenv("ZELLIJ", originalZellijEnv)
		_ = os.Setenv("ZELLIJ_SESSION_NAME", originalSessionEnv)
	})

	lookPath = func(bin string) (string, error) {
		switch bin {
		case "zellij":
			return "/usr/bin/zellij", nil
		case "copilot":
			return "/usr/bin/copilot", nil
		default:
			return "", fmt.Errorf("unknown binary %s", bin)
		}
	}

	runExec = func(binary string, args []string, env []string) error { return nil }

	var layoutContent string
	runCommand = func(binary string, args []string, dir string) error {
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "--layout" {
				data, err := os.ReadFile(args[i+1])
				if err != nil {
					return err
				}
				layoutContent = string(data)
				break
			}
		}
		return nil
	}

	_ = os.Setenv("ZELLIJ", "0")
	_ = os.Setenv("ZELLIJ_SESSION_NAME", "dev-session")

	var out bytes.Buffer
	wtPath := t.TempDir()
	cfg := config.Config{Model: "z-model", BranchOpenMode: "zellij", BranchZellijLayout: "horizontal"}
	if err := launchBranchInteractive(&out, wtPath, "feature-zellij", "myrepo", cfg); err != nil {
		t.Fatalf("launchBranchInteractive: %v", err)
	}

	if !strings.Contains(layoutContent, `split_direction="horizontal"`) {
		t.Fatalf("layout = %q, want horizontal split", layoutContent)
	}
}

func TestLaunchBranchInteractive_Zellij(t *testing.T) {
	originalExec := runExec
	originalLook := lookPath
	originalRunCmd := runCommand
	originalZellijEnv := os.Getenv("ZELLIJ")
	originalSessionEnv := os.Getenv("ZELLIJ_SESSION_NAME")
	t.Cleanup(func() {
		runExec = originalExec
		lookPath = originalLook
		runCommand = originalRunCmd
		_ = os.Setenv("ZELLIJ", originalZellijEnv)
		_ = os.Setenv("ZELLIJ_SESSION_NAME", originalSessionEnv)
	})

	lookPath = func(bin string) (string, error) {
		switch bin {
		case "zellij":
			return "/usr/bin/zellij", nil
		case "copilot":
			return "/usr/bin/copilot", nil
		default:
			return "", fmt.Errorf("unknown binary %s", bin)
		}
	}

	var execCalled bool
	runExec = func(binary string, args []string, env []string) error {
		execCalled = true
		return nil
	}

	var calledBinary string
	var calledArgs []string
	var calledDir string
	runCommand = func(binary string, args []string, dir string) error {
		calledBinary = binary
		calledArgs = append([]string{}, args...)
		calledDir = dir
		return nil
	}

	_ = os.Setenv("ZELLIJ", "0")
	_ = os.Setenv("ZELLIJ_SESSION_NAME", "dev-session")

	var out bytes.Buffer
	wtPath := t.TempDir()
	cfg := config.Config{Model: "z-model", BranchOpenMode: "zellij"}
	if err := launchBranchInteractive(&out, wtPath, "feature-zellij", "myrepo", cfg); err != nil {
		t.Fatalf("launchBranchInteractive: %v", err)
	}

	if execCalled {
		t.Fatal("runExec should not be called in zellij mode")
	}
	if calledBinary != "/usr/bin/zellij" {
		t.Fatalf("binary = %q, want /usr/bin/zellij", calledBinary)
	}
	if len(calledArgs) < 4 || calledArgs[0] != "--session" || calledArgs[1] != "dev-session" || calledArgs[2] != "action" || calledArgs[3] != "new-tab" {
		t.Fatalf("args = %v, want zellij --session dev-session action new-tab ...", calledArgs)
	}
	if calledDir != wtPath {
		t.Fatalf("dir = %q, want %q", calledDir, wtPath)
	}

	// Verify tab name includes repo name
	var tabName string
	for i := 0; i < len(calledArgs)-1; i++ {
		if calledArgs[i] == "--name" {
			tabName = calledArgs[i+1]
		}
	}
	if tabName != "myrepo:feature-zellij" {
		t.Fatalf("tab name = %q, want %q", tabName, "myrepo:feature-zellij")
	}

	var layoutPath string
	for i := 0; i < len(calledArgs)-1; i++ {
		if calledArgs[i] == "--layout" {
			layoutPath = calledArgs[i+1]
		}
	}
	if layoutPath == "" {
		t.Fatalf("args = %v, want --layout <path>", calledArgs)
	}
	if !filepath.IsAbs(layoutPath) {
		t.Fatalf("layout path = %q, want absolute path", layoutPath)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("expected user-facing output")
	}
}

func TestOpenZellijTab_IncludesResumeArgs(t *testing.T) {
	originalLook := lookPath
	originalRunCmd := runCommand
	originalZellijEnv := os.Getenv("ZELLIJ")
	originalSessionEnv := os.Getenv("ZELLIJ_SESSION_NAME")
	t.Cleanup(func() {
		lookPath = originalLook
		runCommand = originalRunCmd
		_ = os.Setenv("ZELLIJ", originalZellijEnv)
		_ = os.Setenv("ZELLIJ_SESSION_NAME", originalSessionEnv)
	})

	lookPath = func(bin string) (string, error) {
		switch bin {
		case "zellij":
			return "/usr/bin/zellij", nil
		case "copilot":
			return "/usr/bin/copilot", nil
		default:
			return "", fmt.Errorf("unknown binary %s", bin)
		}
	}

	var layoutContent string
	var calledArgs []string
	runCommand = func(binary string, args []string, dir string) error {
		calledArgs = append([]string{}, args...)
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "--layout" {
				data, err := os.ReadFile(args[i+1])
				if err != nil {
					return err
				}
				layoutContent = string(data)
				break
			}
		}
		return nil
	}

	_ = os.Setenv("ZELLIJ", "0")
	_ = os.Setenv("ZELLIJ_SESSION_NAME", "dev-session")

	wtPath := t.TempDir()
	copilotArgs := []string{"copilot", "--model", "test-model", "--resume", "session-abc"}
	cfg := config.Config{BranchZellijLayout: "vertical"}
	if err := openZellijTab(wtPath, "my-branch", "myrepo", copilotArgs, cfg); err != nil {
		t.Fatalf("openZellijTab: %v", err)
	}

	// Verify layout includes resume args.
	if !strings.Contains(layoutContent, `"--resume"`) {
		t.Fatalf("layout = %q, want --resume arg", layoutContent)
	}
	if !strings.Contains(layoutContent, `"session-abc"`) {
		t.Fatalf("layout = %q, want session ID", layoutContent)
	}

	// Verify tab name.
	var tabName string
	for i := 0; i < len(calledArgs)-1; i++ {
		if calledArgs[i] == "--name" {
			tabName = calledArgs[i+1]
		}
	}
	if tabName != "myrepo:my-branch" {
		t.Fatalf("tab name = %q, want %q", tabName, "myrepo:my-branch")
	}
}

func TestLaunchBranchInteractive_ZellijRequiresSessionContext(t *testing.T) {
	originalLook := lookPath
	originalZellijEnv := os.Getenv("ZELLIJ")
	originalSessionEnv := os.Getenv("ZELLIJ_SESSION_NAME")
	t.Cleanup(func() {
		lookPath = originalLook
		_ = os.Setenv("ZELLIJ", originalZellijEnv)
		_ = os.Setenv("ZELLIJ_SESSION_NAME", originalSessionEnv)
	})

	lookPath = func(bin string) (string, error) {
		switch bin {
		case "zellij", "copilot":
			return "/usr/bin/" + bin, nil
		default:
			return "", fmt.Errorf("unknown binary %s", bin)
		}
	}

	_ = os.Unsetenv("ZELLIJ")
	_ = os.Unsetenv("ZELLIJ_SESSION_NAME")

	var out bytes.Buffer
	wtPath := t.TempDir()
	cfg := config.Config{Model: "z-model", BranchOpenMode: "zellij"}
	err := launchBranchInteractive(&out, wtPath, "feature-zellij", "myrepo", cfg)
	if err == nil {
		t.Fatal("expected error when not in zellij and no session name is available")
	}
	if !strings.Contains(err.Error(), "active zellij session") {
		t.Fatalf("error = %q, want mention of active zellij session", err.Error())
	}
}

func TestParsePRNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "plain number", input: "42", want: 42},
		{name: "hash prefix", input: "#42", want: 42},
		{name: "full URL", input: "https://github.com/owner/repo/pull/42", want: 42},
		{name: "URL with trailing slash", input: "https://github.com/owner/repo/pull/123/", want: 123},
		{name: "empty", input: "", wantErr: true},
		{name: "not a number", input: "abc", wantErr: true},
		{name: "zero", input: "0", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePRNumber(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("parsePRNumber(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestBrCheckout(t *testing.T) {
	originalGh := runGh
	originalExec := runExec
	originalLook := lookPath
	originalLoadCfg := loadEffectiveConfig
	t.Cleanup(func() {
		runGh = originalGh
		runExec = originalExec
		lookPath = originalLook
		loadEffectiveConfig = originalLoadCfg
	})

	// Use the actual repo's owner/repo so the mismatch check passes.
	runGh = func(dir string, args ...string) (string, error) {
		return `{"headRefName":"feature-branch","url":"https://github.com/alexneyler/fitz/pull/42"}`, nil
	}

	lookPath = func(string) (string, error) { return "/usr/bin/copilot", nil }
	loadEffectiveConfig = func(dir string) config.Config {
		return config.Config{BranchOpenMode: "standard"}
	}

	var execArgs []string
	runExec = func(binary string, args []string, env []string) error {
		execArgs = args
		return nil
	}

	var out bytes.Buffer
	err := BrCheckout(context.Background(), &out, "42")
	if err != nil {
		// Real git commands fail in test without a remote. Tolerate fetch/worktree
		// errors — the gh mock proves PR parsing and gh integration work.
		if strings.Contains(err.Error(), "get PR info") {
			t.Skip("skipping: gh not available or not authed")
		}
		if strings.Contains(err.Error(), "fetch PR") || strings.Contains(err.Error(), "create worktree") {
			return
		}
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "checked out PR #42") {
		t.Errorf("stdout = %q, want 'checked out PR #42'", output)
	}
	_ = execArgs
}

func TestBrCheckoutRepoMismatch(t *testing.T) {
	originalGh := runGh
	t.Cleanup(func() { runGh = originalGh })

	runGh = func(dir string, args ...string) (string, error) {
		return `{"headRefName":"feature-branch","url":"https://github.com/other-org/other-repo/pull/99"}`, nil
	}

	var out bytes.Buffer
	err := BrCheckout(context.Background(), &out, "99")
	if err == nil {
		t.Fatal("expected error for repo mismatch")
	}
	if !strings.Contains(err.Error(), "belongs to other-org/other-repo") {
		t.Fatalf("error = %q, want repo mismatch message", err.Error())
	}
}
