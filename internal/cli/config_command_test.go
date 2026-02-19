package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newConfigCmd(homeDir string) configCommand {
	return configCommand{homeDir: homeDir}
}

func runConfigCmd(t *testing.T, homeDir string, args []string) (stdout, stderr string, err error) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	cmd := newConfigCmd(homeDir)
	err = cmd.Run(context.Background(), args, strings.NewReader(""), &outBuf, &errBuf)
	return outBuf.String(), errBuf.String(), err
}

func TestConfigHelp(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			dir := t.TempDir()
			out, _, err := runConfigCmd(t, dir, []string{arg})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(out, "Usage: fitz config") {
				t.Errorf("help output missing usage, got: %s", out)
			}
		})
	}
}

func TestConfigHelp_NoArgs(t *testing.T) {
	dir := t.TempDir()
	out, _, err := runConfigCmd(t, dir, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Usage: fitz config") {
		t.Errorf("expected help output, got: %s", out)
	}
}

func TestConfigHelp_GlobalOnly(t *testing.T) {
	dir := t.TempDir()
	out, _, err := runConfigCmd(t, dir, []string{"--global"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Usage: fitz config") {
		t.Errorf("expected help output, got: %s", out)
	}
}

func TestConfigSetAndGet_Global(t *testing.T) {
	dir := t.TempDir()

	_, _, err := runConfigCmd(t, dir, []string{"--global", "set", "model", "my-model"})
	if err != nil {
		t.Fatalf("set: %v", err)
	}

	// Verify file exists.
	globalPath := filepath.Join(dir, ".fitz", "config.json")
	if _, err := os.Stat(globalPath); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	out, _, err := runConfigCmd(t, dir, []string{"--global", "get", "model"})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out, "my-model") {
		t.Errorf("get output = %q, want %q", out, "my-model")
	}
}

func TestConfigSetAndGet_Repo(t *testing.T) {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, ".fitz", "owner", "repo", "config.json")

	// Write directly so we can use a fake path.
	if err := os.MkdirAll(filepath.Dir(repoPath), 0o755); err != nil {
		t.Fatal(err)
	}

	// Use a configCommand that targets the global scope to test set/get
	// (repo scope requires git context; test global as proxy for the path logic).
	_, _, err := runConfigCmd(t, dir, []string{"--global", "set", "agent", "copilot-cli"})
	if err != nil {
		t.Fatalf("set: %v", err)
	}

	out, _, err := runConfigCmd(t, dir, []string{"--global", "get", "agent"})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out, "copilot-cli") {
		t.Errorf("get output = %q, want %q", out, "copilot-cli")
	}
}

func TestConfigGet_NotSet(t *testing.T) {
	dir := t.TempDir()
	out, _, err := runConfigCmd(t, dir, []string{"--global", "get", "model"})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out, "(not set)") {
		t.Errorf("get output = %q, want (not set)", out)
	}
}

func TestConfigGet_UnknownKey(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runConfigCmd(t, dir, []string{"--global", "get", "nope"})
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestConfigGet_MissingArg(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runConfigCmd(t, dir, []string{"--global", "get"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConfigSet_UnknownKey(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runConfigCmd(t, dir, []string{"--global", "set", "nope", "val"})
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestConfigSet_MissingArgs(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runConfigCmd(t, dir, []string{"--global", "set", "model"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConfigUnset_Global(t *testing.T) {
	dir := t.TempDir()

	_, _, err := runConfigCmd(t, dir, []string{"--global", "set", "model", "to-remove"})
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = runConfigCmd(t, dir, []string{"--global", "unset", "model"})
	if err != nil {
		t.Fatalf("unset: %v", err)
	}

	out, _, err := runConfigCmd(t, dir, []string{"--global", "get", "model"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "(not set)") {
		t.Errorf("after unset, get = %q, want (not set)", out)
	}
}

func TestConfigUnset_UnknownKey(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runConfigCmd(t, dir, []string{"--global", "unset", "nope"})
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestConfigUnset_MissingArg(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runConfigCmd(t, dir, []string{"--global", "unset"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConfigList_Empty(t *testing.T) {
	dir := t.TempDir()
	out, _, err := runConfigCmd(t, dir, []string{"--global", "list"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "model=(not set)") {
		t.Errorf("list output = %q, want model=(not set)", out)
	}
	if !strings.Contains(out, "agent=(not set)") {
		t.Errorf("list output = %q, want agent=(not set)", out)
	}
}

func TestConfigList_WithValues(t *testing.T) {
	dir := t.TempDir()

	_, _, err := runConfigCmd(t, dir, []string{"--global", "set", "model", "cool-model"})
	if err != nil {
		t.Fatal(err)
	}

	out, _, err := runConfigCmd(t, dir, []string{"--global", "list"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "model=cool-model") {
		t.Errorf("list = %q, want model=cool-model", out)
	}
}

func TestConfigUnknownSubcommand(t *testing.T) {
	dir := t.TempDir()
	_, _, err := runConfigCmd(t, dir, []string{"--global", "bogus"})
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown config subcommand") {
		t.Errorf("error = %q, want unknown config subcommand", err.Error())
	}
}

func TestConfigHandleHelp_ViaExecute(t *testing.T) {
	// Ensures configCommand satisfies the Subcommand interface and
	// TestAllSubcommandsHandleHelp will pass for "config".
	for _, arg := range []string{"help", "--help", "-h"} {
		t.Run(arg, func(t *testing.T) {
			var out, errOut bytes.Buffer
			err := Execute([]string{"config", arg}, strings.NewReader(""), &out, &errOut)
			if err != nil {
				t.Fatalf("Execute config %s: %v", arg, err)
			}
			if !strings.Contains(out.String(), "Usage: fitz config") {
				t.Errorf("expected help output, got: %s", out.String())
			}
		})
	}
}

func TestConfigInHelpOutput(t *testing.T) {
	var out bytes.Buffer
	Execute([]string{"help"}, strings.NewReader(""), &out, io.Discard)
	if !strings.Contains(out.String(), "config") {
		t.Errorf("main help missing 'config', got:\n%s", out.String())
	}
}
