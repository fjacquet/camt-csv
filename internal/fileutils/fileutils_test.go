package fileutils_test

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/fileutils"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSetLogger(t *testing.T) {
	// SetLogger is a no-op, just verify it doesn't panic
	logger := logrus.New()
	fileutils.SetLogger(logger)
	fileutils.SetLogger(nil)
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0600)
	assert.NoError(t, err)

	// Test existing file
	assert.True(t, fileutils.FileExists(testFile))

	// Test non-existent file
	assert.False(t, fileutils.FileExists(filepath.Join(tmpDir, "nonexistent.txt")))

	// Test directory (should return false)
	assert.False(t, fileutils.FileExists(tmpDir))
}

func TestDirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Test existing directory
	assert.True(t, fileutils.DirectoryExists(tmpDir))

	// Test non-existent directory
	assert.False(t, fileutils.DirectoryExists(filepath.Join(tmpDir, "nonexistent")))

	// Create a file and test (should return false)
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0600)
	assert.NoError(t, err)
	assert.False(t, fileutils.DirectoryExists(testFile))
}

func TestEnsureDirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Test creating a new directory
	newDir := filepath.Join(tmpDir, "new", "nested", "dir")
	err := fileutils.EnsureDirectoryExists(newDir)
	assert.NoError(t, err)
	assert.True(t, fileutils.DirectoryExists(newDir))

	// Test with existing directory (should not error)
	err = fileutils.EnsureDirectoryExists(tmpDir)
	assert.NoError(t, err)
}

func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file with content
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	err := os.WriteFile(testFile, content, 0600)
	assert.NoError(t, err)

	// Test reading existing file
	data, err := fileutils.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, content, data)

	// Test reading non-existent file
	_, err = fileutils.ReadFile(filepath.Join(tmpDir, "nonexistent.txt"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file does not exist")
}

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Test writing to a new file
	testFile := filepath.Join(tmpDir, "output.txt")
	content := []byte("test content")
	err := fileutils.WriteFile(testFile, content, 0600)
	assert.NoError(t, err)

	// Verify file was written
	data, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, content, data)

	// Test writing with nested directories
	nestedFile := filepath.Join(tmpDir, "a", "b", "c", "output.txt")
	err = fileutils.WriteFile(nestedFile, content, 0600)
	assert.NoError(t, err)
	assert.True(t, fileutils.FileExists(nestedFile))
}

func TestOpenFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0600)
	assert.NoError(t, err)

	// Test opening existing file
	file, err := fileutils.OpenFile(testFile)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	_ = file.Close()

	// Test opening non-existent file
	_, err = fileutils.OpenFile(filepath.Join(tmpDir, "nonexistent.txt"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file does not exist")
}

func TestCreateFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Test creating a new file
	testFile := filepath.Join(tmpDir, "new.txt")
	file, err := fileutils.CreateFile(testFile)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	_ = file.Close()
	assert.True(t, fileutils.FileExists(testFile))

	// Test creating file with nested directories
	nestedFile := filepath.Join(tmpDir, "x", "y", "z", "new.txt")
	file, err = fileutils.CreateFile(nestedFile)
	assert.NoError(t, err)
	assert.NotNil(t, file)
	_ = file.Close()
	assert.True(t, fileutils.FileExists(nestedFile))
}

func TestListFilesWithExtension(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files with different extensions
	xmlFile1 := filepath.Join(tmpDir, "file1.xml")
	xmlFile2 := filepath.Join(tmpDir, "file2.xml")
	txtFile := filepath.Join(tmpDir, "file.txt")

	for _, f := range []string{xmlFile1, xmlFile2, txtFile} {
		err := os.WriteFile(f, []byte("test"), 0600)
		assert.NoError(t, err)
	}

	// Test listing XML files
	files, err := fileutils.ListFilesWithExtension(tmpDir, ".xml")
	assert.NoError(t, err)
	assert.Len(t, files, 2)

	// Test listing TXT files
	files, err = fileutils.ListFilesWithExtension(tmpDir, ".txt")
	assert.NoError(t, err)
	assert.Len(t, files, 1)

	// Test listing with no matches
	files, err = fileutils.ListFilesWithExtension(tmpDir, ".csv")
	assert.NoError(t, err)
	assert.Len(t, files, 0)

	// Test with non-existent directory
	_, err = fileutils.ListFilesWithExtension(filepath.Join(tmpDir, "nonexistent"), ".xml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "directory does not exist")
}

func TestListFilesWithExtension_Nested(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directory structure with files
	nestedDir := filepath.Join(tmpDir, "nested")
	err := os.MkdirAll(nestedDir, 0750)
	assert.NoError(t, err)

	// Create files in root and nested
	rootFile := filepath.Join(tmpDir, "root.xml")
	nestedFile := filepath.Join(nestedDir, "nested.xml")

	for _, f := range []string{rootFile, nestedFile} {
		err := os.WriteFile(f, []byte("test"), 0600)
		assert.NoError(t, err)
	}

	// Should find both files
	files, err := fileutils.ListFilesWithExtension(tmpDir, ".xml")
	assert.NoError(t, err)
	assert.Len(t, files, 2)
}
