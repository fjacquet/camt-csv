package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCodebaseScanner_NilLogger(t *testing.T) {
	scanner := NewCodebaseScanner(nil)
	require.NotNil(t, scanner)
	assert.NotNil(t, scanner.logger)
}

func TestCodebaseScanner_ScanPaths_UnreadableFile(t *testing.T) {
	scanner := NewCodebaseScanner(nil)

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "unreadable.go")
	err := os.WriteFile(filePath, []byte("content"), 0600)
	require.NoError(t, err)

	// Make file unreadable
	err = os.Chmod(filePath, 0000)
	require.NoError(t, err)

	_, err = scanner.ScanPaths([]string{filePath})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestCodebaseScanner_ScanDirectory_UnreadableFileInDir(t *testing.T) {
	scanner := NewCodebaseScanner(nil)

	tempDir := t.TempDir()
	goodFile := filepath.Join(tempDir, "good.go")
	err := os.WriteFile(goodFile, []byte("good"), 0600)
	require.NoError(t, err)

	badFile := filepath.Join(tempDir, "bad.go")
	err = os.WriteFile(badFile, []byte("bad"), 0600)
	require.NoError(t, err)

	// Make one file unreadable — scanFile will error, causing WalkDir to stop
	err = os.Chmod(badFile, 0000)
	require.NoError(t, err)

	_, err = scanner.ScanPaths([]string{tempDir})
	assert.Error(t, err)
}

func TestCodebaseScanner_ScanPaths_EmptyDirectory(t *testing.T) {
	scanner := NewCodebaseScanner(nil)

	tempDir := t.TempDir()
	emptyDir := filepath.Join(tempDir, "empty")
	err := os.Mkdir(emptyDir, 0750)
	require.NoError(t, err)

	sections, err := scanner.ScanPaths([]string{emptyDir})
	assert.NoError(t, err)
	assert.Empty(t, sections)
}

func TestCodebaseScanner_ScanPaths_NestedDirectories(t *testing.T) {
	scanner := NewCodebaseScanner(nil)

	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "sub")
	err := os.Mkdir(subDir, 0750)
	require.NoError(t, err)

	file1 := filepath.Join(tempDir, "root.go")
	err = os.WriteFile(file1, []byte("root"), 0600)
	require.NoError(t, err)

	file2 := filepath.Join(subDir, "nested.go")
	err = os.WriteFile(file2, []byte("nested"), 0600)
	require.NoError(t, err)

	sections, err := scanner.ScanPaths([]string{tempDir})
	assert.NoError(t, err)
	assert.Len(t, sections, 2)
}
