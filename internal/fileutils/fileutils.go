// Package fileutils provides common file operations used throughout the application.
package fileutils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// SetLogger sets a custom logger for this package
func SetLogger(logger *logrus.Logger) {
	if logger != nil {
		log = logger
	}
}

// FileExists checks if a file exists and is not a directory
func FileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirectoryExists checks if a directory exists
func DirectoryExists(dirPath string) bool {
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// EnsureDirectoryExists creates a directory if it doesn't exist
func EnsureDirectoryExists(dirPath string) error {
	if !DirectoryExists(dirPath) {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	return nil
}

// ReadFile reads the entire contents of a file and returns it as a byte slice
func ReadFile(filePath string) ([]byte, error) {
	if !FileExists(filePath) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// WriteFile writes data to a file, creating the file if it doesn't exist
// and creating any parent directories if needed
func WriteFile(filePath string, data []byte, perm os.FileMode) error {
	// Create parent directories if they don't exist
	dir := filepath.Dir(filePath)
	if err := EnsureDirectoryExists(dir); err != nil {
		return err
	}

	// Write to file
	if err := os.WriteFile(filePath, data, perm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// OpenFile opens a file for reading, returning an error if the file doesn't exist
func OpenFile(filePath string) (*os.File, error) {
	if !FileExists(filePath) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// CreateFile creates or truncates a file for writing
func CreateFile(filePath string) (*os.File, error) {
	// Create parent directories if they don't exist
	dir := filepath.Dir(filePath)
	if err := EnsureDirectoryExists(dir); err != nil {
		return nil, err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return file, nil
}

// ListFilesWithExtension returns a list of files with the specified extension in a directory
func ListFilesWithExtension(dirPath, extension string) ([]string, error) {
	if !DirectoryExists(dirPath) {
		return nil, fmt.Errorf("directory does not exist: %s", dirPath)
	}

	var files []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == extension {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return files, nil
}
