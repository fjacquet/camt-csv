package validation

import (
	"fmt"
	"os"
	"path/filepath"
)

// IsValidPath checks if a given path exists and is accessible.
func IsValidPath(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", path)
	}
	if err != nil {
		return fmt.Errorf("error checking path %s: %w", path, err)
	}

	// Ensure it's an absolute path
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	// Check if it's a file or directory
	if !info.IsDir() && !info.Mode().IsRegular() {
		return fmt.Errorf("path %s is neither a file nor a directory", path)
	}

	return nil
}

// IsValidOutputFormat checks if the given format is supported.
func IsValidOutputFormat(format string) error {
	switch format {
	case "json", "xml":
		return nil
	default:
		return fmt.Errorf("unsupported output format: %s. Supported formats are 'json', 'xml'", format)
	}
}

// IsValidFilePermissions checks if the given file mode is valid for sensitive files.
func IsValidFilePermissions(mode os.FileMode) error {
	// For sensitive files like config, 0600 is recommended.
	// For reports, 0755 might be acceptable, but 0644 is safer.
	// This is a basic check, more complex logic might be needed based on context.
	// For now, we'll just ensure it's not overly permissive (e.g., 0777).
	if mode&0007 != 0 { // Check if 'others' have any permissions
		return fmt.Errorf("file permissions are too permissive: %s. Recommended 0600 or 0644", mode.String())
	}
	return nil
}
