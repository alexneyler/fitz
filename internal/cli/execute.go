package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"fitz/internal/cliapp"
)

var Version = "dev"
var runUpdate = cliapp.Update

func Execute(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] == "help" {
		printUsage(stdout)
		return nil
	}

	var err error
	switch args[0] {
	case "version":
		err = cliapp.Version(context.Background(), stdout, currentVersion())
	case "update":
		err = runUpdate(context.Background(), stdout)
	case "completion":
		err = cliapp.Completion(context.Background(), stdout, args[1:])
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command: %s", args[0])
	}

	if err != nil {
		return fmt.Errorf("%s failed: %w", args[0], err)
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
