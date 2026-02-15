package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestExecuteKnownCommands(t *testing.T) {
	prev := runUpdate
	runUpdate = func(_ context.Context, w io.Writer) error {
		_, err := fmt.Fprintln(w, "updated from fitz_test")
		return err
	}
	t.Cleanup(func() { runUpdate = prev })

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "help", args: []string{"help"}, want: "fitz dev"},
		{name: "help long flag", args: []string{"--help"}, want: "fitz dev"},
		{name: "version", args: []string{"version"}, want: "fitz dev"},
		{name: "update", args: []string{"update"}, want: "updated from fitz_test"},
		{name: "completion bash", args: []string{"completion", "bash"}, want: "complete -F _fitz_completion fitz"},
		{name: "completion zsh", args: []string{"completion", "zsh"}, want: "compdef _fitz fitz"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			err := Execute(tc.args, &out, &errOut)
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("stdout = %q, want substring %q", out.String(), tc.want)
			}
			if tc.name == "help" && !strings.Contains(out.String(), "Usage: fitz <command>") {
				t.Fatalf("stdout = %q, want usage header", out.String())
			}
			if tc.name == "help" && !strings.Contains(out.String(), "br") {
				t.Fatalf("stdout = %q, want br command listed", out.String())
			}
		})
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	err := Execute([]string{"wat"}, &out, &errOut)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if !strings.Contains(errOut.String(), "fitz dev") {
		t.Fatalf("stderr = %q, want version", errOut.String())
	}
	if !strings.Contains(errOut.String(), "Usage: fitz <command>") {
		t.Fatalf("stderr = %q, want usage header", errOut.String())
	}
}

func TestExecuteCompletionWithoutShell(t *testing.T) {
	var out, errOut bytes.Buffer
	err := Execute([]string{"completion"}, &out, &errOut)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "completion failed: usage: fitz completion <bash|zsh>") {
		t.Fatalf("error = %q", err.Error())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", out.String())
	}
}

func TestExecuteBrCommands(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "br no args", args: []string{"br"}, wantErr: false},
		{name: "br list", args: []string{"br", "list"}, wantErr: false},
		{name: "br new missing name", args: []string{"br", "new"}, wantErr: true},
		{name: "br go missing name", args: []string{"br", "go"}, wantErr: true},
		{name: "br rm missing name", args: []string{"br", "rm"}, wantErr: true},
		{name: "br cd missing name", args: []string{"br", "cd"}, wantErr: true},
		{name: "br unknown subcommand", args: []string{"br", "wat"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			err := Execute(tc.args, &out, &errOut)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAllSubcommandsHandleHelp(t *testing.T) {
	for name := range subcommands {
		for _, helpArg := range []string{"help", "--help", "-h"} {
			t.Run(name+"/"+helpArg, func(t *testing.T) {
				var out, errOut bytes.Buffer
				err := Execute([]string{name, helpArg}, &out, &errOut)
				if err != nil {
					t.Fatalf("Execute(%s %s) returned error: %v", name, helpArg, err)
				}
				if !strings.Contains(out.String(), "Usage:") {
					t.Fatalf("stdout = %q, want Usage header", out.String())
				}
			})
		}
	}
}

func TestExecuteBrRmExtraArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "br rm with unknown flag",
			args:    []string{"br", "rm", "foo", "--oops"},
			wantErr: true,
			errMsg:  "usage: fitz br rm <name> [--force]",
		},
		{
			name:    "br rm with two names",
			args:    []string{"br", "rm", "foo", "bar"},
			wantErr: true,
			errMsg:  "usage: fitz br rm <name> [--force]",
		},
		{
			name:    "br rm with too many args",
			args:    []string{"br", "rm", "foo", "--force", "extra"},
			wantErr: true,
			errMsg:  "usage: fitz br rm <name> [--force]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			err := Execute(tc.args, &out, &errOut)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr && tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.errMsg)
			}
		})
	}
}
