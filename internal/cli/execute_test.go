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
			stdin := strings.NewReader("")
			err := Execute(tc.args, stdin, &out, &errOut)
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
	stdin := strings.NewReader("")
	err := Execute([]string{"wat"}, stdin, &out, &errOut)
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
	stdin := strings.NewReader("")
	err := Execute([]string{"completion"}, stdin, &out, &errOut)
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
			stdin := strings.NewReader("q")
			err := Execute(tc.args, stdin, &out, &errOut)
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
				stdin := strings.NewReader("")
				err := Execute([]string{name, helpArg}, stdin, &out, &errOut)
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

func TestParseBrNewArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantName   string
		wantBase   string
		wantPrompt string
		wantErr    bool
	}{
		{name: "name only", args: []string{"new", "feat"}, wantName: "feat"},
		{name: "name and prompt", args: []string{"new", "feat", "do stuff"}, wantName: "feat", wantPrompt: "do stuff"},
		{name: "base flag before name", args: []string{"new", "--base", "develop", "feat"}, wantName: "feat", wantBase: "develop"},
		{name: "base flag after name", args: []string{"new", "feat", "--base", "develop"}, wantName: "feat", wantBase: "develop"},
		{name: "base flag with prompt", args: []string{"new", "--base", "develop", "feat", "do stuff"}, wantName: "feat", wantBase: "develop", wantPrompt: "do stuff"},
		{name: "multi-word prompt unquoted", args: []string{"new", "feat", "fix", "the", "bug"}, wantName: "feat", wantPrompt: "fix the bug"},
		{name: "multi-word prompt with base", args: []string{"new", "--base", "develop", "feat", "fix", "the", "bug"}, wantName: "feat", wantBase: "develop", wantPrompt: "fix the bug"},
		{name: "missing name", args: []string{"new"}, wantErr: true},
		{name: "base flag missing value", args: []string{"new", "--base"}, wantErr: true},
		{name: "base flag missing value then name", args: []string{"new", "feat", "--base"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, base, prompt, err := parseBrNewArgs(tc.args[1:])
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tc.wantName {
				t.Errorf("name = %q, want %q", name, tc.wantName)
			}
			if base != tc.wantBase {
				t.Errorf("base = %q, want %q", base, tc.wantBase)
			}
			if prompt != tc.wantPrompt {
				t.Errorf("prompt = %q, want %q", prompt, tc.wantPrompt)
			}
		})
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
		{
			name:    "br rm --all with name",
			args:    []string{"br", "rm", "--all", "foo"},
			wantErr: true,
			errMsg:  "usage: fitz br rm",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			stdin := strings.NewReader("")
			err := Execute(tc.args, stdin, &out, &errOut)
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

func TestExecuteTodoHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("")
	err := Execute([]string{"todo"}, stdin, &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Usage: fitz todo") {
		t.Fatalf("stdout = %q, want usage header", out.String())
	}
}

func TestExecuteHelpListsTodo(t *testing.T) {
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("")
	err := Execute([]string{"help"}, stdin, &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "todo") {
		t.Fatalf("stdout = %q, want 'todo' listed", out.String())
	}
}

func TestExecuteHelpListsAgent(t *testing.T) {
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("")
	err := Execute([]string{"help"}, stdin, &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "agent") {
		t.Fatalf("stdout = %q, want 'agent' listed", out.String())
	}
}

func TestExecuteAgentStatus(t *testing.T) {
	prev := runAgentStatus
	t.Cleanup(func() { runAgentStatus = prev })

	var gotMessage, gotPR string
	runAgentStatus = func(w io.Writer, message, prURL string) error {
		gotMessage = message
		gotPR = prURL
		return nil
	}

	var out, errOut bytes.Buffer
	err := Execute([]string{"agent", "status", "--pr", "https://github.com/acme/repo/pull/42", "ready for review"}, strings.NewReader(""), &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMessage != "ready for review" {
		t.Fatalf("message = %q", gotMessage)
	}
	if gotPR != "https://github.com/acme/repo/pull/42" {
		t.Fatalf("pr = %q", gotPR)
	}
}

func TestParseAgentStatusArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantMessage string
		wantPR      string
		wantErr     bool
	}{
		{name: "message only", args: []string{"Implementing auth"}, wantMessage: "Implementing auth"},
		{name: "pr only", args: []string{"--pr", "https://github.com/acme/repo/pull/42"}, wantPR: "https://github.com/acme/repo/pull/42"},
		{name: "message and pr", args: []string{"--pr", "https://github.com/acme/repo/pull/42", "Ready"}, wantMessage: "Ready", wantPR: "https://github.com/acme/repo/pull/42"},
		{name: "missing pr value", args: []string{"--pr"}, wantErr: true},
		{name: "empty update", args: nil, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			message, prURL, err := parseAgentStatusArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if message != tc.wantMessage {
				t.Fatalf("message = %q, want %q", message, tc.wantMessage)
			}
			if prURL != tc.wantPR {
				t.Fatalf("pr = %q, want %q", prURL, tc.wantPR)
			}
		})
	}
}
