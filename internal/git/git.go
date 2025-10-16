package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GetDiff returns the diff between the current working directory and a specified Git reference.
func GetDiff(gitRef string) (string, error) {
	cmd := exec.Command("git", "diff", gitRef)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %s - %w", stderr.String(), err)
	}

	return out.String(), nil
}

// GetCurrentBranch returns the name of the current Git branch.
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to get current git branch: %s - %w", stderr.String(), err)
	}

	return strings.TrimSpace(out.String()), nil
}

// IsGitRepo checks if the current directory is a Git repository.
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}
