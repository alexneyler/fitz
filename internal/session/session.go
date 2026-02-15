package session

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FindLatestSession scans configDir/session-state/*/workspace.yaml for the
// most recently updated session whose cwd matches worktreePath. Returns the
// session ID, or "" if none found.
func FindLatestSession(configDir, worktreePath string) (string, error) {
	stateDir := filepath.Join(configDir, "session-state")
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	var bestID string
	var bestTime time.Time

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		wsPath := filepath.Join(stateDir, e.Name(), "workspace.yaml")
		id, cwd, updatedAt := parseWorkspace(wsPath)
		if id == "" || cwd != worktreePath {
			continue
		}
		if updatedAt.After(bestTime) {
			bestTime = updatedAt
			bestID = id
		}
	}

	return bestID, nil
}

// parseWorkspace reads a workspace.yaml and extracts id, cwd, and updated_at.
// Returns zero values for fields that can't be parsed.
func parseWorkspace(path string) (id, cwd string, updatedAt time.Time) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", time.Time{}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if k, v, ok := splitYAMLLine(line); ok {
			switch k {
			case "id":
				id = v
			case "cwd":
				cwd = v
			case "updated_at":
				if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
					updatedAt = t
				}
			}
		}
	}

	// Fall back to file modification time if updated_at was not present.
	if id != "" && updatedAt.IsZero() {
		if info, err := os.Stat(path); err == nil {
			updatedAt = info.ModTime()
		}
	}

	return id, cwd, updatedAt
}

func splitYAMLLine(line string) (key, value string, ok bool) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}
