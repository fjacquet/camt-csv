package scanner

import (
	"fjacquet/camt-csv/internal/models"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodebaseScanner_ScanPaths_File(t *testing.T) {
	scanner := NewCodebaseScanner()

	// Create a temporary file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_file.go")
	content := []byte("package main\nfunc main() {}\n")
	err := os.WriteFile(filePath, content, 0600)
	assert.NoError(t, err)

	sections, err := scanner.ScanPaths([]string{filePath})
	assert.NoError(t, err)
	assert.Len(t, sections, 1)
	assert.Equal(t, filePath, sections[0].Path)
	assert.Equal(t, models.CodebaseSectionTypeFile, sections[0].Type)
	assert.Equal(t, string(content), sections[0].Content)
}

func TestCodebaseScanner_ScanPaths_Directory(t *testing.T) {
	scanner := NewCodebaseScanner()

	// Create a temporary directory with multiple files
	tempDir := t.TempDir()
	dirPath := filepath.Join(tempDir, "test_dir")
	err := os.Mkdir(dirPath, 0750)
	assert.NoError(t, err)

	file1Path := filepath.Join(dirPath, "file1.go")
	file1Content := []byte("package main\n// file1")
	err = os.WriteFile(file1Path, file1Content, 0600)
	assert.NoError(t, err)

	file2Path := filepath.Join(dirPath, "file2.go")
	file2Content := []byte("package main\n// file2")
	err = os.WriteFile(file2Path, file2Content, 0600)
	assert.NoError(t, err)

	sections, err := scanner.ScanPaths([]string{dirPath})
	assert.NoError(t, err)
	assert.Len(t, sections, 2)

	// Check if both files are present (order might vary)
	foundFile1 := false
	foundFile2 := false
	for _, s := range sections {
		assert.Equal(t, models.CodebaseSectionTypeFile, s.Type)
		if s.Path == file1Path {
			assert.Equal(t, string(file1Content), s.Content)
			foundFile1 = true
		} else if s.Path == file2Path {
			assert.Equal(t, string(file2Content), s.Content)
			foundFile2 = true
		}
	}
	assert.True(t, foundFile1)
	assert.True(t, foundFile2)
}

func TestCodebaseScanner_ScanPaths_NonExistentPath(t *testing.T) {
	scanner := NewCodebaseScanner()

	_, err := scanner.ScanPaths([]string{"/non/existent/path"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stat path")
}

func TestCodebaseScanner_ScanPaths_MixedPaths(t *testing.T) {
	scanner := NewCodebaseScanner()

	// Create a temporary file and a directory with a file
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "single_file.go")
	fileContent := []byte("package main\n// single")
	err := os.WriteFile(filePath, fileContent, 0600)
	assert.NoError(t, err)

	dirPath := filepath.Join(tempDir, "mixed_dir")
	err = os.Mkdir(dirPath, 0750)
	assert.NoError(t, err)

	dirFilePath := filepath.Join(dirPath, "dir_file.go")
	dirFileContent := []byte("package main\n// dir_file")
	err = os.WriteFile(dirFilePath, dirFileContent, 0600)
	assert.NoError(t, err)

	sections, err := scanner.ScanPaths([]string{filePath, dirPath})
	assert.NoError(t, err)
	assert.Len(t, sections, 2)

	foundSingleFile := false
	foundDirFile := false
	for _, s := range sections {
		assert.Equal(t, models.CodebaseSectionTypeFile, s.Type)
		if s.Path == filePath {
			assert.Equal(t, string(fileContent), s.Content)
			foundSingleFile = true
		} else if s.Path == dirFilePath {
			assert.Equal(t, string(dirFileContent), s.Content)
			foundDirFile = true
		}
	}
	assert.True(t, foundSingleFile)
	assert.True(t, foundDirFile)
}

func TestCodebaseScanner_ScanPaths_EmptyPaths(t *testing.T) {
	scanner := NewCodebaseScanner()

	sections, err := scanner.ScanPaths([]string{})
	assert.NoError(t, err)
	assert.Len(t, sections, 0)
}
