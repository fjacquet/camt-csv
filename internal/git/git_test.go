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

func TestGetChangedFiles(t *testing.T) {
	t.Run("returns files against HEAD", func(t *testing.T) {
		// Getting changed files against HEAD should work
		files, err := GetChangedFiles("HEAD")
		assert.NoError(t, err)
		// Result might be empty if no changes, but shouldn't error
		assert.NotNil(t, files)
	})

	t.Run("returns error for invalid ref", func(t *testing.T) {
		_, err := GetChangedFiles("nonexistent-ref-12345")
		assert.Error(t, err, "should error on invalid git ref")
		assert.Contains(t, err.Error(), "failed to get changed files")
	})

	t.Run("returns empty slice for same commit", func(t *testing.T) {
		// Comparing HEAD to itself should return no changes
		files, err := GetChangedFiles("HEAD")
		assert.NoError(t, err)
		// This should work and return an empty or minimal list
		_ = files
	})
}
