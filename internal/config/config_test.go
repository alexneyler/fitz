package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"fitz/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.Model != "gpt-5.3-codex" {
		t.Errorf("default model = %q, want %q", cfg.Model, "gpt-5.3-codex")
	}
	if cfg.Agent != "copilot-cli" {
		t.Errorf("default agent = %q, want %q", cfg.Agent, "copilot-cli")
	}
}

func TestGlobalConfigPath(t *testing.T) {
	path, err := config.GlobalConfigPath("/home/user")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/home/user", ".fitz", "config.json")
	if path != want {
		t.Errorf("GlobalConfigPath = %q, want %q", path, want)
	}
}

func TestRepoConfigPath(t *testing.T) {
	path, err := config.RepoConfigPath("/home/user", "alice", "myrepo")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/home/user", ".fitz", "alice", "myrepo", "config.json")
	if path != want {
		t.Errorf("RepoConfigPath = %q, want %q", path, want)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path/config.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.Model != "" || cfg.Agent != "" {
		t.Errorf("expected empty config for missing file, got: %+v", cfg)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"model":"claude-opus","agent":"copilot-cli"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "claude-opus" {
		t.Errorf("model = %q, want %q", cfg.Model, "claude-opus")
	}
	if cfg.Agent != "copilot-cli" {
		t.Errorf("agent = %q, want %q", cfg.Agent, "copilot-cli")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`not json`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.json")

	original := config.Config{Model: "my-model", Agent: "copilot-cli"}
	if err := config.Save(path, original); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != original {
		t.Errorf("round-trip: got %+v, want %+v", loaded, original)
	}
}

func TestLoadEffective_DefaultsOnly(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.LoadEffective(dir, "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	def := config.DefaultConfig()
	if cfg != def {
		t.Errorf("effective (no files) = %+v, want defaults %+v", cfg, def)
	}
}

func TestLoadEffective_GlobalOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, ".fitz", "config.json")
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(globalPath, []byte(`{"model":"global-model"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadEffective(dir, "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "global-model" {
		t.Errorf("model = %q, want %q", cfg.Model, "global-model")
	}
	// agent not set in global, so falls back to default
	if cfg.Agent != "copilot-cli" {
		t.Errorf("agent = %q, want default %q", cfg.Agent, "copilot-cli")
	}
}

func TestLoadEffective_RepoOverridesGlobal(t *testing.T) {
	dir := t.TempDir()

	globalPath := filepath.Join(dir, ".fitz", "config.json")
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(globalPath, []byte(`{"model":"global-model","agent":"global-agent"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	repoPath := filepath.Join(dir, ".fitz", "owner", "repo", "config.json")
	if err := os.MkdirAll(filepath.Dir(repoPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repoPath, []byte(`{"model":"repo-model"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadEffective(dir, "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "repo-model" {
		t.Errorf("model = %q, want %q", cfg.Model, "repo-model")
	}
	// agent set in global, not overridden by repo
	if cfg.Agent != "global-agent" {
		t.Errorf("agent = %q, want %q", cfg.Agent, "global-agent")
	}
}

func TestLoadEffective_NoRepoOwner(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, ".fitz", "config.json")
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(globalPath, []byte(`{"model":"global-model"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// empty owner/repo skips repo layer
	cfg, err := config.LoadEffective(dir, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "global-model" {
		t.Errorf("model = %q, want %q", cfg.Model, "global-model")
	}
}

func TestGet(t *testing.T) {
	cfg := config.Config{Model: "m", Agent: "a"}

	if v, ok := config.Get(cfg, "model"); !ok || v != "m" {
		t.Errorf("Get model = %q, %v; want %q, true", v, ok, "m")
	}
	if v, ok := config.Get(cfg, "agent"); !ok || v != "a" {
		t.Errorf("Get agent = %q, %v; want %q, true", v, ok, "a")
	}
	if _, ok := config.Get(cfg, "unknown"); ok {
		t.Error("Get unknown should return ok=false")
	}
}

func TestSet(t *testing.T) {
	cfg := config.Config{}

	cfg2, err := config.Set(cfg, "model", "new-model")
	if err != nil || cfg2.Model != "new-model" {
		t.Errorf("Set model: got %+v, err=%v", cfg2, err)
	}

	cfg3, err := config.Set(cfg, "agent", "new-agent")
	if err != nil || cfg3.Agent != "new-agent" {
		t.Errorf("Set agent: got %+v, err=%v", cfg3, err)
	}

	_, err = config.Set(cfg, "unknown", "v")
	if err == nil {
		t.Error("Set unknown key should return error")
	}
}

func TestUnset(t *testing.T) {
	cfg := config.Config{Model: "m", Agent: "a"}

	cfg2, err := config.Unset(cfg, "model")
	if err != nil || cfg2.Model != "" {
		t.Errorf("Unset model: got %+v, err=%v", cfg2, err)
	}

	cfg3, err := config.Unset(cfg, "agent")
	if err != nil || cfg3.Agent != "" {
		t.Errorf("Unset agent: got %+v, err=%v", cfg3, err)
	}

	_, err = config.Unset(cfg, "unknown")
	if err == nil {
		t.Error("Unset unknown key should return error")
	}
}

func TestKeys(t *testing.T) {
	if len(config.Keys) == 0 {
		t.Error("Keys should not be empty")
	}
	found := map[string]bool{}
	for _, k := range config.Keys {
		found[k] = true
	}
	for _, required := range []string{"model", "agent"} {
		if !found[required] {
			t.Errorf("Keys missing %q", required)
		}
	}
}
