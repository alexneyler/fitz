package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/kong"

	"fitz/internal/cliapp"
)

var Version = "dev"
var runUpdate = cliapp.Update

type commandLine struct {
	Version    struct{} `cmd:"" help:"Print version information."`
	Update     struct{} `cmd:"" help:"Update fitz to the latest release."`
	Completion struct {
		Shell string `arg:"" optional:"" help:"Target shell (bash or zsh)."`
	} `cmd:"" help:"Print shell completion script."`
	Br struct{} `cmd:"" help:"Manage worktrees."`
}

func Execute(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printUsage(stdout)
		return nil
	}

	commandName := strings.TrimSpace(args[0])
	if commandName == "br" {
		return handleBr(args[1:], stdout, stderr)
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
	fmt.Fprintln(w, "  update        Update fitz to the latest release")
	fmt.Fprintln(w, "  version       Print version information")
}

func handleBr(args []string, stdout, stderr io.Writer) error {
	ctx := context.Background()

	if len(args) == 0 {
		return cliapp.BrCurrent(ctx, stdout)
	}

	subcommand := args[0]
	switch subcommand {
	case "new":
		if len(args) < 2 {
			return fmt.Errorf("usage: fitz br new <name> [base]")
		}
		name := args[1]
		base := ""
		if len(args) > 2 {
			base = args[2]
		}
		return cliapp.BrNew(ctx, stdout, name, base)

	case "go":
		if len(args) < 2 {
			return fmt.Errorf("usage: fitz br go <name>")
		}
		return cliapp.BrGo(ctx, stdout, args[1])

	case "rm":
		if len(args) < 2 {
			return fmt.Errorf("usage: fitz br rm <name> [--force]")
		}
		name := args[1]
		force := false

		if len(args) > 2 {
			if len(args) == 3 && args[2] == "--force" {
				force = true
			} else {
				return fmt.Errorf("usage: fitz br rm <name> [--force]")
			}
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
		return fmt.Errorf("unknown br subcommand: %s", subcommand)
	}
}

func currentVersion() string {
	if strings.TrimSpace(Version) == "" {
		return "dev"
	}
	return Version
}
