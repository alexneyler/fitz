package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds fitz user configuration. Zero values mean "not set".
type Config struct {
	Model string `json:"model,omitempty"`
	Agent string `json:"agent,omitempty"`
}

// DefaultConfig returns the hardcoded default configuration.
func DefaultConfig() Config {
	return Config{
		Model: "gpt-5.3-codex",
		Agent: "copilot-cli",
	}
}

// GlobalConfigPath returns the path to the global config file (~/.fitz/config.json).
func GlobalConfigPath(homeDir string) (string, error) {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
	}
	return filepath.Join(homeDir, ".fitz", "config.json"), nil
}

// RepoConfigPath returns the path to the repo-level config file (~/.fitz/<owner>/<repo>/config.json).
func RepoConfigPath(homeDir, owner, repo string) (string, error) {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
	}
	return filepath.Join(homeDir, ".fitz", owner, repo, "config.json"), nil
}

// Load reads a Config from a JSON file. Returns an empty Config if the file does not exist.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes cfg to path as JSON, creating parent directories as needed.
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

// merge overlays non-empty fields from src onto dst.
func merge(dst, src Config) Config {
	if src.Model != "" {
		dst.Model = src.Model
	}
	if src.Agent != "" {
		dst.Agent = src.Agent
	}
	return dst
}

// LoadEffective builds the effective config for a repo by merging:
// defaults <- global config <- repo config.
func LoadEffective(homeDir, owner, repo string) (Config, error) {
	cfg := DefaultConfig()

	globalPath, err := GlobalConfigPath(homeDir)
	if err != nil {
		return cfg, err
	}
	globalCfg, err := Load(globalPath)
	if err != nil {
		return cfg, err
	}
	cfg = merge(cfg, globalCfg)

	if owner != "" && repo != "" {
		repoPath, err := RepoConfigPath(homeDir, owner, repo)
		if err != nil {
			return cfg, err
		}
		repoCfg, err := Load(repoPath)
		if err != nil {
			return cfg, err
		}
		cfg = merge(cfg, repoCfg)
	}

	return cfg, nil
}

// Get returns the value of the named field, and whether the key is valid.
func Get(cfg Config, key string) (string, bool) {
	switch key {
	case "model":
		return cfg.Model, true
	case "agent":
		return cfg.Agent, true
	default:
		return "", false
	}
}

// Set returns a new Config with the named field set to value.
// Returns an error if the key is unknown.
func Set(cfg Config, key, value string) (Config, error) {
	switch key {
	case "model":
		cfg.Model = value
	case "agent":
		cfg.Agent = value
	default:
		return cfg, fmt.Errorf("unknown config key: %s (valid keys: model, agent)", key)
	}
	return cfg, nil
}

// Unset returns a new Config with the named field cleared.
// Returns an error if the key is unknown.
func Unset(cfg Config, key string) (Config, error) {
	switch key {
	case "model":
		cfg.Model = ""
	case "agent":
		cfg.Agent = ""
	default:
		return cfg, fmt.Errorf("unknown config key: %s (valid keys: model, agent)", key)
	}
	return cfg, nil
}

// Keys returns the list of all valid config keys.
var Keys = []string{"model", "agent"}
