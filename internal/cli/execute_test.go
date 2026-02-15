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
			if tc.name == "help" && !strings.Contains(out.String(), "Usage: fitz <help|version|update|completion>") {
				t.Fatalf("stdout = %q, want usage", out.String())
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
	if !strings.Contains(errOut.String(), "Usage: fitz <help|version|update|completion>") {
		t.Fatalf("stderr = %q, want usage", errOut.String())
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
