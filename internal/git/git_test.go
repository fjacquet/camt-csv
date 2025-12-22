package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// These are integration tests that require a real git repository.
// The camt-csv project is a git repo, so these tests will work.

func TestIsGitRepo(t *testing.T) {
	t.Run("returns true in git repository", func(t *testing.T) {
		// We're running from within the camt-csv repo
		result := IsGitRepo()
		assert.True(t, result, "should detect that we're in a git repository")
	})
}

func TestGetCurrentBranch(t *testing.T) {
	t.Run("returns branch name", func(t *testing.T) {
		branch, err := GetCurrentBranch()
		assert.NoError(t, err)
		assert.NotEmpty(t, branch, "branch name should not be empty")
		// Branch name should be a valid identifier (no newlines, etc.)
		assert.NotContains(t, branch, "\n")
	})
}

func TestGetDiff(t *testing.T) {
	t.Run("returns diff against HEAD", func(t *testing.T) {
		// Diff against HEAD should work, even if empty
		diff, err := GetDiff("HEAD")
		assert.NoError(t, err)
		// Result might be empty if no changes, but shouldn't error
		_ = diff
	})

	t.Run("returns diff against HEAD~0", func(t *testing.T) {
		// HEAD~0 is the same as HEAD
		diff, err := GetDiff("HEAD~0")
		assert.NoError(t, err)
		_ = diff
	})

	t.Run("returns error for invalid ref", func(t *testing.T) {
		_, err := GetDiff("nonexistent-ref-12345")
		assert.Error(t, err, "should error on invalid git ref")
		assert.Contains(t, err.Error(), "failed to get git diff")
	})
}
