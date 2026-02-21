package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// validGitRef matches safe git references (branches, tags, SHAs, HEAD, etc.)
var validGitRef = regexp.MustCompile(`^[a-zA-Z0-9_./~^@{}-]+$`)

// sanitizeGitRef validates that a git reference contains only safe characters.
func sanitizeGitRef(ref string) error {
	if !validGitRef.MatchString(ref) {
		return fmt.Errorf("invalid git reference: %q", ref)
	}
	return nil
}

// IsGitRepo checks if the current directory is a Git repository.
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}

// GetChangedFiles returns a list of files changed between the working tree and a git reference.
// This uses `git diff --name-only` for simple, reliable path extraction.
func GetChangedFiles(gitRef string) ([]string, error) {
	if err := sanitizeGitRef(gitRef); err != nil {
		return nil, err
	}
	cmd := exec.Command("git", "diff", "--name-only", gitRef) // #nosec G204 -- gitRef is sanitized above
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %s - %w", stderr.String(), err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	files := make([]string, 0) // Initialize to empty slice, not nil
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}
