package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"fitz/internal/config"
	"fitz/internal/worktree"
)

type configCommand struct {
	// homeDir is overrideable for testing.
	homeDir string
}

func (configCommand) Help(w io.Writer) {
	fmt.Fprintln(w, "Usage: fitz config [--global] <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  get <key>           Get a config value")
	fmt.Fprintln(w, "  set <key> <value>   Set a config value")
	fmt.Fprintln(w, "  unset <key>         Remove a config value")
	fmt.Fprintln(w, "  list                List all config values")
	fmt.Fprintln(w, "  help                Show this help message")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --global    Operate on global config (~/.fitz/config.json)")
	fmt.Fprintln(w, "              Default: repo-level config (~/.fitz/<owner>/<repo>/config.json)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Valid keys: model, agent")
}

func (c configCommand) Run(_ context.Context, args []string, _ io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		c.Help(stdout)
		return nil
	}

	// Parse --global flag (may appear anywhere before the subcommand).
	global := false
	filtered := args[:0:0]
	for _, a := range args {
		if a == "--global" {
			global = true
		} else {
			filtered = append(filtered, a)
		}
	}
	args = filtered

	if len(args) == 0 {
		c.Help(stdout)
		return nil
	}

	subcommand := args[0]
	rest := args[1:]

	if isHelpArg(subcommand) {
		c.Help(stdout)
		return nil
	}

	configPath, err := c.resolvePath(global)
	if err != nil {
		return err
	}

	switch subcommand {
	case "get":
		return c.runGet(stdout, configPath, rest)
	case "set":
		return c.runSet(configPath, rest)
	case "unset":
		return c.runUnset(configPath, rest)
	case "list":
		return c.runList(stdout, configPath)
	default:
		c.Help(stderr)
		return fmt.Errorf("unknown config subcommand: %s", subcommand)
	}
}

// resolvePath returns the config file path based on scope.
func (c configCommand) resolvePath(global bool) (string, error) {
	homeDir := c.homeDir

	if global {
		return config.GlobalConfigPath(homeDir)
	}

	// Repo-level: need owner/repo from git context.
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	git := worktree.ShellGit{}
	owner, repo, err := worktree.RepoID(git, cwd)
	if err != nil || (owner == "" && repo == "") {
		return "", fmt.Errorf("could not determine repo (are you in a git repo with an origin remote?); use --global for global config")
	}

	return config.RepoConfigPath(homeDir, owner, repo)
}

func (c configCommand) runGet(w io.Writer, configPath string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: fitz config get <key>")
	}
	key := args[0]

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	value, ok := config.Get(cfg, key)
	if !ok {
		return fmt.Errorf("unknown config key: %s (valid keys: model, agent)", key)
	}
	if value == "" {
		fmt.Fprintf(w, "(not set)\n")
	} else {
		fmt.Fprintln(w, value)
	}
	return nil
}

func (c configCommand) runSet(configPath string, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: fitz config set <key> <value>")
	}
	key, value := args[0], args[1]

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	cfg, err = config.Set(cfg, key, value)
	if err != nil {
		return err
	}

	return config.Save(configPath, cfg)
}

func (c configCommand) runUnset(configPath string, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: fitz config unset <key>")
	}
	key := args[0]

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	cfg, err = config.Unset(cfg, key)
	if err != nil {
		return err
	}

	return config.Save(configPath, cfg)
}

func (c configCommand) runList(w io.Writer, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	for _, key := range config.Keys {
		value, _ := config.Get(cfg, key)
		if value == "" {
			fmt.Fprintf(w, "%s=(not set)\n", key)
		} else {
			fmt.Fprintf(w, "%s=%s\n", key, value)
		}
	}
	return nil
}
