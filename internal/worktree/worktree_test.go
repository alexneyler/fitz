package worktree

import (
	"testing"
)

func TestManagerCreate(t *testing.T) {
	git := &mockGit{
		outputs: map[string]string{
			"/test/repo:remote get-url origin ":                                            "https://github.com/owner/repo.git",
			"/test/repo:worktree add /home/user/.fitz/owner/repo/feature -b feature main ": "",
		},
		errs: make(map[string]error),
	}

	m := &Manager{
		Git:     git,
		HomeDir: "/home/user",
	}

	path, err := m.Create("/test/repo", "feature", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantPath := "/home/user/.fitz/owner/repo/feature"
	if path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}

	found := false
	for _, call := range git.calls {
		if len(call) > 1 && call[1] == "worktree" && call[2] == "add" {
			found = true
			if call[3] != wantPath {
				t.Errorf("worktree add path = %q, want %q", call[3], wantPath)
			}
			if call[4] != "-b" {
				t.Errorf("expected -b flag")
			}
			if call[5] != "feature" {
				t.Errorf("branch name = %q, want feature", call[5])
			}
			if call[6] != "main" {
				t.Errorf("base = %q, want main", call[6])
			}
		}
	}
	if !found {
		t.Error("expected worktree add call")
	}
}

func TestManagerCreateDefaultBase(t *testing.T) {
	git := &mockGit{
		outputs: map[string]string{
			"/test/repo:remote get-url origin ":                                       "https://github.com/owner/repo.git",
			"/test/repo:worktree add /home/user/.fitz/owner/repo/feature -b feature ": "",
		},
		errs: make(map[string]error),
	}

	m := &Manager{
		Git:     git,
		HomeDir: "/home/user",
	}

	_, err := m.Create("/test/repo", "feature", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, call := range git.calls {
		if len(call) > 1 && call[1] == "worktree" && call[2] == "add" {
			found = true
			if len(call) != 6 {
				t.Errorf("expected 6 args, got %d", len(call))
			}
		}
	}
	if !found {
		t.Error("expected worktree add call")
	}
}

func TestManagerRemove(t *testing.T) {
	git := &mockGit{
		outputs: map[string]string{
			"/test/repo:remote get-url origin ":                               "https://github.com/owner/repo.git",
			"/test/repo:worktree remove /home/user/.fitz/owner/repo/feature ": "",
			"/test/repo:worktree prune ":                                      "",
		},
		errs: make(map[string]error),
	}

	m := &Manager{
		Git:     git,
		HomeDir: "/home/user",
	}

	err := m.Remove("/test/repo", "feature", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundRemove := false
	foundPrune := false
	for _, call := range git.calls {
		if len(call) > 2 && call[1] == "worktree" && call[2] == "remove" {
			foundRemove = true
			wantPath := "/home/user/.fitz/owner/repo/feature"
			if call[3] != wantPath {
				t.Errorf("worktree remove path = %q, want %q", call[3], wantPath)
			}
		}
		if len(call) > 2 && call[1] == "worktree" && call[2] == "prune" {
			foundPrune = true
		}
	}
	if !foundRemove {
		t.Error("expected worktree remove call")
	}
	if !foundPrune {
		t.Error("expected worktree prune call")
	}
}

func TestManagerRemoveForce(t *testing.T) {
	git := &mockGit{
		outputs: map[string]string{
			"/test/repo:remote get-url origin ":                                       "https://github.com/owner/repo.git",
			"/test/repo:worktree remove /home/user/.fitz/owner/repo/feature --force ": "",
			"/test/repo:worktree prune ":                                              "",
		},
		errs: make(map[string]error),
	}

	m := &Manager{
		Git:     git,
		HomeDir: "/home/user",
	}

	err := m.Remove("/test/repo", "feature", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, call := range git.calls {
		if len(call) > 2 && call[1] == "worktree" && call[2] == "remove" {
			found = true
			if call[4] != "--force" {
				t.Errorf("expected --force flag")
			}
		}
	}
	if !found {
		t.Error("expected worktree remove call")
	}
}

func TestManagerList(t *testing.T) {
	porcelain := `worktree /repo/main
HEAD abc123
branch refs/heads/main

worktree /repo/feature
HEAD def456
branch refs/heads/feature

worktree /repo/detached
HEAD ghi789
detached
`

	git := &mockGit{
		outputs: map[string]string{
			"/test/repo:worktree list --porcelain ": porcelain,
		},
		errs: make(map[string]error),
	}

	m := &Manager{
		Git: git,
	}

	list, err := m.List("/test/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(list))
	}

	if list[0].Path != "/repo/main" {
		t.Errorf("list[0].Path = %q, want /repo/main", list[0].Path)
	}
	if list[0].Branch != "main" {
		t.Errorf("list[0].Branch = %q, want main", list[0].Branch)
	}
	if list[0].Name != "main" {
		t.Errorf("list[0].Name = %q, want main", list[0].Name)
	}

	if list[1].Path != "/repo/feature" {
		t.Errorf("list[1].Path = %q, want /repo/feature", list[1].Path)
	}
	if list[1].Branch != "feature" {
		t.Errorf("list[1].Branch = %q, want feature", list[1].Branch)
	}

	if list[2].Path != "/repo/detached" {
		t.Errorf("list[2].Path = %q, want /repo/detached", list[2].Path)
	}
	if list[2].Branch != "" {
		t.Errorf("list[2].Branch = %q, want empty", list[2].Branch)
	}
}

func TestManagerPath(t *testing.T) {
	git := &mockGit{
		outputs: map[string]string{
			"/test/repo:remote get-url origin ": "https://github.com/owner/repo.git",
		},
		errs: make(map[string]error),
	}

	m := &Manager{
		Git:     git,
		HomeDir: "/home/user",
	}

	path, err := m.Path("/test/repo", "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "/home/user/.fitz/owner/repo/feature"
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestManagerCurrent(t *testing.T) {
	tests := []struct {
		name      string
		dir       string
		isWtree   bool
		wtreeList string
		want      string
		wantErr   bool
	}{
		{
			name:    "not in worktree",
			dir:     "/repo/main",
			isWtree: false,
			want:    "root",
		},
		{
			name:    "in worktree",
			dir:     "/home/user/.fitz/owner/repo/feature",
			isWtree: true,
			wtreeList: `worktree /repo/main
HEAD abc123
branch refs/heads/main

worktree /home/user/.fitz/owner/repo/feature
HEAD def456
branch refs/heads/feature
`,
			want: "feature",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			git := &mockGit{
				outputs: map[string]string{
					tc.dir + ":rev-parse --show-toplevel ":  tc.dir,
					tc.dir + ":worktree list --porcelain ":  tc.wtreeList,
					tc.dir + ":rev-parse --git-common-dir ": "/repo/.git",
				},
				errs: make(map[string]error),
			}

			if tc.isWtree {
				git.outputs[tc.dir+":rev-parse --git-dir "] = "/repo/.git/worktrees/feature"
			} else {
				git.outputs[tc.dir+":rev-parse --git-dir "] = "/repo/.git"
			}

			m := &Manager{
				Git: git,
			}

			current, err := m.Current(tc.dir)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if current != tc.want {
				t.Errorf("current = %q, want %q", current, tc.want)
			}
		})
	}
}

func TestParseWorktreeList(t *testing.T) {
	porcelain := `worktree /repo/main
HEAD abc123
branch refs/heads/main

worktree /repo/feature
HEAD def456
branch refs/heads/feature
bare

worktree /repo/detached
HEAD ghi789
detached
`

	list := parseWorktreeList(porcelain)
	if len(list) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(list))
	}

	if list[0].Path != "/repo/main" {
		t.Errorf("list[0].Path = %q", list[0].Path)
	}
	if list[0].Branch != "main" {
		t.Errorf("list[0].Branch = %q", list[0].Branch)
	}
	if list[0].Bare {
		t.Error("list[0].Bare should be false")
	}

	if !list[1].Bare {
		t.Error("list[1].Bare should be true")
	}

	if list[2].Branch != "" {
		t.Errorf("list[2].Branch = %q, want empty", list[2].Branch)
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names
		{name: "simple name", input: "my-feature", wantErr: false},
		{name: "with numbers", input: "fix-123", wantErr: false},
		{name: "uppercase", input: "CAPS", wantErr: false},
		{name: "mixed case", input: "MyFeature", wantErr: false},
		{name: "underscore", input: "my_feature", wantErr: false},

		// Invalid names
		{name: "path traversal", input: "../evil", wantErr: true},
		{name: "forward slash", input: "foo/bar", wantErr: true},
		{name: "backslash", input: "foo\\bar", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "dash prefix", input: "--flag", wantErr: true},
		{name: "single dash prefix", input: "-flag", wantErr: true},
		{name: "only whitespace", input: "   ", wantErr: true},
		{name: "double dot alone", input: "..", wantErr: true},
		{name: "double dot in middle", input: "foo..bar", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateName(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("ValidateName(%q) expected error, got nil", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ValidateName(%q) unexpected error: %v", tc.input, err)
			}
		})
	}
}

func TestManagerCreateRejectsInvalidName(t *testing.T) {
	git := &mockGit{
		outputs: map[string]string{
			"/test/repo:remote get-url origin ": "https://github.com/owner/repo.git",
		},
		errs: make(map[string]error),
	}

	m := &Manager{
		Git:     git,
		HomeDir: "/home/user",
	}

	_, err := m.Create("/test/repo", "../evil", "main")
	if err == nil {
		t.Fatal("expected error for path traversal name")
	}
}

func TestManagerPathRejectsInvalidName(t *testing.T) {
	git := &mockGit{
		outputs: map[string]string{
			"/test/repo:remote get-url origin ": "https://github.com/owner/repo.git",
		},
		errs: make(map[string]error),
	}

	m := &Manager{
		Git:     git,
		HomeDir: "/home/user",
	}

	_, err := m.Path("/test/repo", "foo/bar")
	if err == nil {
		t.Fatal("expected error for name with slash")
	}
}

func TestManagerRemoveRejectsInvalidName(t *testing.T) {
	git := &mockGit{
		outputs: map[string]string{
			"/test/repo:remote get-url origin ": "https://github.com/owner/repo.git",
		},
		errs: make(map[string]error),
	}

	m := &Manager{
		Git:     git,
		HomeDir: "/home/user",
	}

	err := m.Remove("/test/repo", "--flag", false)
	if err == nil {
		t.Fatal("expected error for name starting with dash")
	}
}
