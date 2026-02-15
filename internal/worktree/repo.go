package worktree

import (
	"errors"
	"path/filepath"
	"strings"
)

type GitRunner interface {
	Run(dir string, args ...string) (string, error)
}

func ParseRemoteURL(rawURL string) (owner, repo string, err error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", "", errors.New("empty URL")
	}

	rawURL = strings.TrimSuffix(rawURL, ".git")

	if strings.HasPrefix(rawURL, "https://github.com/") {
		parts := strings.TrimPrefix(rawURL, "https://github.com/")
		segments := strings.Split(parts, "/")
		if len(segments) >= 2 {
			return segments[0], segments[1], nil
		}
	} else if strings.HasPrefix(rawURL, "git@github.com:") {
		parts := strings.TrimPrefix(rawURL, "git@github.com:")
		segments := strings.Split(parts, "/")
		if len(segments) >= 2 {
			return segments[0], segments[1], nil
		}
	}

	return "", "", errors.New("invalid remote URL format")
}

func RepoID(git GitRunner, gitDir string) (owner, repo string, err error) {
	remoteURL, err := git.Run(gitDir, "remote", "get-url", "origin")
	if err != nil {
		return "", filepath.Base(gitDir), nil
	}

	owner, repo, err = ParseRemoteURL(remoteURL)
	if err != nil {
		return "", filepath.Base(gitDir), nil
	}

	return owner, repo, nil
}

func GitRoot(git GitRunner, dir string) (string, error) {
	output, err := git.Run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func IsWorktree(git GitRunner, dir string) (bool, error) {
	commonDir, err := git.Run(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return false, err
	}
	commonDir = strings.TrimSpace(commonDir)

	gitDir, err := git.Run(dir, "rev-parse", "--git-dir")
	if err != nil {
		return false, err
	}
	gitDir = strings.TrimSpace(gitDir)

	return commonDir != gitDir, nil
}
