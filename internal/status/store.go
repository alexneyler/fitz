package status

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type BranchStatus struct {
	Message   string    `json:"message,omitempty"`
	PRURL     string    `json:"pr_url,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func StorePath(homeDir, owner, repo string) (string, error) {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("get home dir: %w", err)
		}
	}
	return filepath.Join(homeDir, ".fitz", owner, repo, "status.json"), nil
}

func Load(path string) (map[string]BranchStatus, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]BranchStatus{}, nil
		}
		return nil, fmt.Errorf("read status: %w", err)
	}

	var entries map[string]BranchStatus
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse status: %w", err)
	}
	if entries == nil {
		entries = map[string]BranchStatus{}
	}
	return entries, nil
}

func Save(path string, entries map[string]BranchStatus) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encode status: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write status: %w", err)
	}
	return nil
}

func SetStatus(path, branch, message string) (BranchStatus, error) {
	entries, err := Load(path)
	if err != nil {
		return BranchStatus{}, err
	}

	entry := entries[branch]
	entry.Message = message
	entry.UpdatedAt = time.Now().UTC()
	entries[branch] = entry
	if err := Save(path, entries); err != nil {
		return BranchStatus{}, err
	}
	return entry, nil
}

func SetPR(path, branch, prURL string) (BranchStatus, error) {
	entries, err := Load(path)
	if err != nil {
		return BranchStatus{}, err
	}

	entry := entries[branch]
	entry.PRURL = prURL
	entry.UpdatedAt = time.Now().UTC()
	entries[branch] = entry
	if err := Save(path, entries); err != nil {
		return BranchStatus{}, err
	}
	return entry, nil
}
