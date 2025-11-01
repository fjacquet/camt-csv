package scanner

import (
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// CodebaseScanner provides functionality to scan files and directories.
type CodebaseScanner struct {
	logger logging.Logger
}

// NewCodebaseScanner creates a new instance of CodebaseScanner.
func NewCodebaseScanner() *CodebaseScanner {
	return &CodebaseScanner{
		logger: logging.GetLogger().WithField("component", "CodebaseScanner"),
	}
}

// ScanPaths scans the given paths (files or directories) and returns a slice of CodebaseSection.
// If a path is a directory, it recursively scans all files within it.
// It returns an error if any path is invalid or unreadable.
func (s *CodebaseScanner) ScanPaths(paths []string) ([]models.CodebaseSection, error) {
	var sections []models.CodebaseSection

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			s.logger.WithError(err).WithField("path", p).Error("Failed to get absolute path")
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", p, err)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			s.logger.WithError(err).WithField("path", absPath).Error("Failed to stat path")
			return nil, fmt.Errorf("failed to stat path %s: %w", absPath, err)
		}

		if info.IsDir() {
			dirSections, err := s.scanDirectory(absPath)
			if err != nil {
				return nil, err
			}
			sections = append(sections, dirSections...)
		} else {
			fileSection, err := s.scanFile(absPath)
			if err != nil {
				return nil, err
			}
			sections = append(sections, fileSection)
		}
	}

	return sections, nil
}

// scanDirectory recursively scans a directory for files.
// It returns a slice of CodebaseSection for each file found.
func (s *CodebaseScanner) scanDirectory(dirPath string) ([]models.CodebaseSection, error) {
	var sections []models.CodebaseSection

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			s.logger.WithError(err).WithField("path", path).Warn("Error walking path")
			return nil // Continue walking even if there's an error with one path
		}

		if !d.IsDir() {
			fileSection, err := s.scanFile(path)
			if err != nil {
				return err // Stop walking if a file cannot be scanned
			}
			sections = append(sections, fileSection)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %s: %w", dirPath, err)
	}

	return sections, nil
}

// scanFile reads the content of a single file.
// It returns a CodebaseSection for the file.
func (s *CodebaseScanner) scanFile(filePath string) (models.CodebaseSection, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		s.logger.WithError(err).WithField("file", filePath).Error("Failed to read file")
		return models.CodebaseSection{}, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return models.CodebaseSection{
		Path:    filePath,
		Type:    models.CodebaseSectionTypeFile,
		Content: string(content),
	}, nil
}
