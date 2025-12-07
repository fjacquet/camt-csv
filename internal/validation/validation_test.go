package validation_test

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/validation"

	"github.com/stretchr/testify/assert"
)

func TestIsValidPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0600)
	assert.NoError(t, err)

	// Get absolute paths
	absFile, err := filepath.Abs(testFile)
	assert.NoError(t, err)
	absDir, err := filepath.Abs(tmpDir)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		path        string
		expectError bool
		errContains string
	}{
		{
			name:        "Valid absolute file path",
			path:        absFile,
			expectError: false,
		},
		{
			name:        "Valid absolute directory path",
			path:        absDir,
			expectError: false,
		},
		{
			name:        "Non-existent path",
			path:        "/nonexistent/path/to/file.txt",
			expectError: true,
			errContains: "path does not exist",
		},
		{
			name:        "Relative path",
			path:        "relative/path",
			expectError: true,
			errContains: "path does not exist", // Check happens before absolute path check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.IsValidPath(tt.path)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidOutputFormat(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		expectError bool
		errContains string
	}{
		{
			name:        "Valid JSON format",
			format:      "json",
			expectError: false,
		},
		{
			name:        "Valid XML format",
			format:      "xml",
			expectError: false,
		},
		{
			name:        "Invalid format - csv",
			format:      "csv",
			expectError: true,
			errContains: "unsupported output format",
		},
		{
			name:        "Invalid format - empty",
			format:      "",
			expectError: true,
			errContains: "unsupported output format",
		},
		{
			name:        "Invalid format - uppercase",
			format:      "JSON",
			expectError: true,
			errContains: "unsupported output format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.IsValidOutputFormat(tt.format)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidFilePermissions(t *testing.T) {
	tests := []struct {
		name        string
		mode        os.FileMode
		expectError bool
		errContains string
	}{
		{
			name:        "Valid 0600 permissions",
			mode:        0600,
			expectError: false,
		},
		{
			name:        "Invalid 0644 permissions (others can read)",
			mode:        0644,
			expectError: true,
			errContains: "too permissive",
		},
		{
			name:        "Valid 0640 permissions",
			mode:        0640,
			expectError: false,
		},
		{
			name:        "Valid 0750 permissions",
			mode:        0750,
			expectError: false,
		},
		{
			name:        "Invalid 0777 permissions",
			mode:        0777,
			expectError: true,
			errContains: "too permissive",
		},
		{
			name:        "Invalid 0666 permissions",
			mode:        0666,
			expectError: true,
			errContains: "too permissive",
		},
		{
			name:        "Invalid 0755 permissions",
			mode:        0755,
			expectError: true,
			errContains: "too permissive",
		},
		{
			name:        "Invalid 0701 permissions",
			mode:        0701,
			expectError: true,
			errContains: "too permissive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.IsValidFilePermissions(tt.mode)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
