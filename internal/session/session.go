package session

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SessionInfo holds metadata about a Copilot session for a worktree.
type SessionInfo struct {
	SessionID string
	Summary   string    // first line of the session summary, empty if none
	UpdatedAt time.Time // zero if unknown
}

// FindLatestSession scans configDir/session-state/*/workspace.yaml for the
// most recently updated session whose cwd matches worktreePath. Returns the
// session ID, or "" if none found.
func FindLatestSession(configDir, worktreePath string) (string, error) {
	info, err := FindSessionInfo(configDir, worktreePath)
	if err != nil {
		return "", err
	}
	return info.SessionID, nil
}

// FindSessionInfo returns metadata about the most recently updated session
// whose cwd matches worktreePath. Returns a zero-value SessionInfo if none found.
func FindSessionInfo(configDir, worktreePath string) (SessionInfo, error) {
	infos, err := FindAllSessionInfos(configDir, []string{worktreePath})
	if err != nil {
		return SessionInfo{}, err
	}
	return infos[worktreePath], nil
}

// FindAllSessionInfos scans configDir/session-state/ once and returns the most
// recently updated SessionInfo for each cwd in cwds. Cwds with no matching
// session are omitted from the returned map.
func FindAllSessionInfos(configDir string, cwds []string) (map[string]SessionInfo, error) {
	if len(cwds) == 0 {
		return map[string]SessionInfo{}, nil
	}

	cwdSet := make(map[string]bool, len(cwds))
	for _, cwd := range cwds {
		cwdSet[cwd] = true
	}

	stateDir := filepath.Join(configDir, "session-state")
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]SessionInfo{}, nil
		}
		return nil, err
	}

	type candidate struct {
		info SessionInfo
		t    time.Time
	}
	best := make(map[string]candidate)

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		wsPath := filepath.Join(stateDir, e.Name(), "workspace.yaml")
		id, cwd, summary, updatedAt := parseWorkspace(wsPath)
		if id == "" || !cwdSet[cwd] {
			continue
		}
		if c, ok := best[cwd]; !ok || updatedAt.After(c.t) {
			best[cwd] = candidate{
				info: SessionInfo{SessionID: id, Summary: summary, UpdatedAt: updatedAt},
				t:    updatedAt,
			}
		}
	}

	result := make(map[string]SessionInfo, len(best))
	for cwd, c := range best {
		result[cwd] = c.info
	}
	return result, nil
}

// parseWorkspace reads a workspace.yaml and extracts id, cwd, summary, and updated_at.
// Returns zero values for fields that can't be parsed.
func parseWorkspace(path string) (id, cwd, summary string, updatedAt time.Time) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", "", time.Time{}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inSummaryBlock := false
	for scanner.Scan() {
		line := scanner.Text()

		// If we're inside a YAML block scalar for summary, collect the first non-empty line.
		if inSummaryBlock {
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" && summary == "" {
					summary = trimmed
				}
				continue
			}
			// Non-indented line ends the block.
			inSummaryBlock = false
		}

		k, v, ok := splitYAMLLine(line)
		if !ok {
			continue
		}
		switch k {
		case "id":
			id = v
		case "cwd":
			cwd = v
		case "updated_at":
			if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
				updatedAt = t
			}
		case "summary":
			if v == "" || v == "|-" || v == "|" || v == ">-" || v == ">" {
				inSummaryBlock = true
			} else {
				summary = v
			}
		}
	}

	// Fall back to file modification time if updated_at was not present.
	if id != "" && updatedAt.IsZero() {
		if info, err := os.Stat(path); err == nil {
			updatedAt = info.ModTime()
		}
	}

	return id, cwd, summary, updatedAt
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
