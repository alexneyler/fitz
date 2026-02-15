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
}

func Execute(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printUsage(stdout)
		return nil
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

	commandName := strings.TrimSpace(args[0])
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
	fmt.Fprintln(w, "Usage: fitz <help|version|update|completion>")
}

func currentVersion() string {
	if strings.TrimSpace(Version) == "" {
		return "dev"
	}
	return Version
}
