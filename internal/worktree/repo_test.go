package worktree

import (
	"errors"
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "https with .git",
			url:       "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "https without .git",
			url:       "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "ssh with .git",
			url:       "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "ssh without .git",
			url:       "git@github.com:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:    "empty url",
			url:     "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			url:     "not-a-url",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			owner, repo, err := ParseRemoteURL(tc.url)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tc.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tc.wantOwner)
			}
			if repo != tc.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tc.wantRepo)
			}
		})
	}
}

type mockGit struct {
	calls   [][]string
	outputs map[string]string
	errs    map[string]error
}

func (m *mockGit) Run(dir string, args ...string) (string, error) {
	key := dir + ":" + argsKey(args)
	m.calls = append(m.calls, append([]string{dir}, args...))
	if err, ok := m.errs[key]; ok {
		return "", err
	}
	if out, ok := m.outputs[key]; ok {
		return out, nil
	}
	return "", errors.New("mock: no output configured")
}

func argsKey(args []string) string {
	result := ""
	for _, arg := range args {
		result += arg + " "
	}
	return result
}

func TestRepoID(t *testing.T) {
	tests := []struct {
		name      string
		gitDir    string
		remoteURL string
		remoteErr error
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "https url",
			gitDir:    "/test/repo",
			remoteURL: "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "ssh url",
			gitDir:    "/test/repo",
			remoteURL: "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "no remote",
			gitDir:    "/test/myrepo",
			remoteErr: errors.New("no remote"),
			wantOwner: "",
			wantRepo:  "myrepo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			git := &mockGit{
				outputs: make(map[string]string),
				errs:    make(map[string]error),
			}

			key := tc.gitDir + ":remote get-url origin "
			if tc.remoteErr != nil {
				git.errs[key] = tc.remoteErr
			} else {
				git.outputs[key] = tc.remoteURL
			}

			owner, repo, err := RepoID(git, tc.gitDir)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tc.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tc.wantOwner)
			}
			if repo != tc.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tc.wantRepo)
			}
		})
	}
}

func TestGitRoot(t *testing.T) {
	git := &mockGit{
		outputs: map[string]string{
			"/some/path:rev-parse --show-toplevel ": "/repo/root\n",
		},
		errs: make(map[string]error),
	}

	root, err := GitRoot(git, "/some/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != "/repo/root" {
		t.Errorf("root = %q, want %q", root, "/repo/root")
	}
}

func TestIsWorktree(t *testing.T) {
	tests := []struct {
		name       string
		gitDir     string
		commonDir  string
		gitDirVal  string
		wantResult bool
		wantErr    bool
	}{
		{
			name:       "is worktree",
			gitDir:     "/worktree/path",
			commonDir:  "/repo/.git",
			gitDirVal:  "/repo/.git/worktrees/feature",
			wantResult: true,
		},
		{
			name:       "is main repo",
			gitDir:     "/repo/path",
			commonDir:  "/repo/.git",
			gitDirVal:  "/repo/.git",
			wantResult: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			git := &mockGit{
				outputs: map[string]string{
					tc.gitDir + ":rev-parse --git-common-dir ": tc.commonDir + "\n",
					tc.gitDir + ":rev-parse --git-dir ":        tc.gitDirVal + "\n",
				},
				errs: make(map[string]error),
			}

			result, err := IsWorktree(git, tc.gitDir)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tc.wantResult {
				t.Errorf("result = %v, want %v", result, tc.wantResult)
			}
		})
	}
}
