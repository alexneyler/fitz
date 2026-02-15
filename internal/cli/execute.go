package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/kong"

	"fitz/internal/cliapp"
)

var Version = "dev"
var runUpdate = cliapp.Update

// Subcommand represents a command that has its own sub-subcommands.
// Any such command must provide a Help method.
type Subcommand interface {
	Help(w io.Writer)
	Run(ctx context.Context, args []string, stdout, stderr io.Writer) error
}

// All commands with sub-subcommands must be registered here.
var subcommands = map[string]Subcommand{
	"br":   brCommand{},
	"todo": todoCommand{},
}

type commandLine struct {
	Version    struct{} `cmd:"" help:"Print version information."`
	Update     struct{} `cmd:"" help:"Update fitz to the latest release."`
	Completion struct {
		Shell string `arg:"" optional:"" help:"Target shell (bash or zsh)."`
	} `cmd:"" help:"Print shell completion script."`
	Br   struct{} `cmd:"" help:"Manage worktrees."`
	Todo struct{} `cmd:"" help:"Quick per-repo todo list."`
}

func Execute(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printUsage(stdout)
		return nil
	}

	commandName := strings.TrimSpace(args[0])
	if sub, ok := subcommands[commandName]; ok {
		subArgs := args[1:]
		if len(subArgs) > 0 && isHelpArg(subArgs[0]) {
			sub.Help(stdout)
			return nil
		}
		return sub.Run(context.Background(), subArgs, stdout, stderr)
	}

	cli := commandLine{}
	parser, err := kong.New(
		&cli,
		kong.Name("fitz"),
		kong.Writers(stdout, stderr),
		kong.NoDefaultHelp(),
		kong.UsageOnError(),
	)
	if err != nil {
		return fmt.Errorf("init parser: %w", err)
	}

	_, err = parser.Parse(args)
	if err != nil {
		printUsage(stderr)
		return err
	}

	switch commandName {
	case "version":
		err = cliapp.Version(context.Background(), stdout, currentVersion())
	case "update":
		err = runUpdate(context.Background(), stdout)
	case "completion":
		completionArgs := []string{}
		if shell := strings.TrimSpace(cli.Completion.Shell); shell != "" {
			completionArgs = append(completionArgs, shell)
		}
		err = cliapp.Completion(context.Background(), stdout, completionArgs)
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command: %s", commandName)
	}

	if err != nil {
		return fmt.Errorf("%s failed: %w", commandName, err)
	}

	return nil
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "fitz %s\n", currentVersion())
	fmt.Fprintln(w, "Usage: fitz <command>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  br            Manage worktrees")
	fmt.Fprintln(w, "  completion    Print shell completion script")
	fmt.Fprintln(w, "  help          Show this help message")
	fmt.Fprintln(w, "  todo          Quick per-repo todo list")
	fmt.Fprintln(w, "  update        Update fitz to the latest release")
	fmt.Fprintln(w, "  version       Print version information")
}

func isHelpArg(s string) bool {
	return s == "help" || s == "--help" || s == "-h"
}

// parseBrNewArgs extracts name, --base value, and optional prompt from
// the arguments after "new".  Returns an error when required values are
// missing.
func parseBrNewArgs(args []string) (name, base, prompt string, err error) {
	var positional []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--base" {
			i++
			if i >= len(args) {
				return "", "", "", fmt.Errorf("usage: fitz br new [--base <branch>] <name> [prompt...]")
			}
			base = args[i]
		} else {
			positional = append(positional, args[i])
		}
	}
	if len(positional) == 0 {
		return "", "", "", fmt.Errorf("usage: fitz br new [--base <branch>] <name> [prompt...]")
	}
	name = positional[0]
	if len(positional) > 1 {
		prompt = strings.Join(positional[1:], " ")
	}
	return name, base, prompt, nil
}

type brCommand struct{}

func (brCommand) Help(w io.Writer) {
	fmt.Fprintln(w, "Usage: fitz br <command>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  cd      Print the path to a worktree")
	fmt.Fprintln(w, "  go      Switch to an existing worktree")
	fmt.Fprintln(w, "  help    Show this help message")
	fmt.Fprintln(w, "  list    List all worktrees")
	fmt.Fprintln(w, "  new     Create a new worktree (optionally with --base and/or prompt)")
	fmt.Fprintln(w, "  rm      Remove a worktree and its branch (--all to remove all)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run with no command to show the current worktree.")
}

func (b brCommand) Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return cliapp.BrCurrent(ctx, stdout)
	}

	subcommand := args[0]
	switch subcommand {
	case "new":
		name, base, prompt, err := parseBrNewArgs(args[1:])
		if err != nil {
			return err
		}
		return cliapp.BrNew(ctx, stdout, name, base, prompt)

	case "go":
		if len(args) < 2 {
			return fmt.Errorf("usage: fitz br go <name>")
		}
		return cliapp.BrGo(ctx, stdout, args[1])

	case "rm":
		if len(args) < 2 {
			return fmt.Errorf("usage: fitz br rm <name> [--force]\n       fitz br rm --all [--force]")
		}

		all := false
		force := false
		var name string

		for _, arg := range args[1:] {
			switch arg {
			case "--all":
				all = true
			case "--force":
				force = true
			default:
				if name != "" {
					return fmt.Errorf("usage: fitz br rm <name> [--force]\n       fitz br rm --all [--force]")
				}
				name = arg
			}
		}

		if all && name != "" {
			return fmt.Errorf("usage: fitz br rm <name> [--force]\n       fitz br rm --all [--force]")
		}
		if !all && name == "" {
			return fmt.Errorf("usage: fitz br rm <name> [--force]\n       fitz br rm --all [--force]")
		}

		if all {
			return cliapp.BrRemoveAll(ctx, stdout, force)
		}
		return cliapp.BrRemove(ctx, stdout, name, force)

	case "list":
		return cliapp.BrList(ctx, stdout)

	case "cd":
		if len(args) < 2 {
			return fmt.Errorf("usage: fitz br cd <name>")
		}
		return cliapp.BrCd(ctx, stdout, args[1])

	default:
		b.Help(stderr)
		return fmt.Errorf("unknown br subcommand: %s", subcommand)
	}
}

func currentVersion() string {
	if strings.TrimSpace(Version) == "" {
		return "dev"
	}
	return Version
}

type todoCommand struct{}

func (todoCommand) Help(w io.Writer) {
	fmt.Fprintln(w, "Usage: fitz todo <command>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  <text>    Add a new todo item")
	fmt.Fprintln(w, "  help      Show this help message")
	fmt.Fprintln(w, "  list      Interactive todo list (enter: create worktree, d: done)")
}

func (t todoCommand) Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		t.Help(stdout)
		return nil
	}

	switch args[0] {
	case "list":
		return cliapp.TodoList(ctx, os.Stdin, stdout)
	default:
		text := strings.Join(args, " ")
		return cliapp.TodoAdd(ctx, stdout, text)
	}
}
