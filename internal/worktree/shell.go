package worktree

import (
	"bytes"
	"fmt"
	"os/exec"
)

type ShellGit struct{}

func (ShellGit) Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %v: %w: %s", args, err, stderr.String())
	}

	return stdout.String(), nil
}
