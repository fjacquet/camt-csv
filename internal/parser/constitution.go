package parser

import (
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ConstitutionLoader provides functionality to load and parse constitution definition files.
type ConstitutionLoader struct {
	logger logging.Logger
}

// NewConstitutionLoader creates a new instance of ConstitutionLoader.
func NewConstitutionLoader(logger logging.Logger) *ConstitutionLoader {
	if logger == nil {
		logger = logging.NewLogrusAdapter("info", "text")
	}
	return &ConstitutionLoader{
		logger: logger.WithField("component", "ConstitutionLoader"),
	}
}

// LoadConstitutionFiles loads and parses multiple constitution definition files.
// It returns a slice of ConstitutionPrinciple and an error if any file cannot be loaded or parsed,
// or if duplicate principle IDs are found across files.
func (l *ConstitutionLoader) LoadConstitutionFiles(filePaths []string) ([]models.ConstitutionPrinciple, error) {
	var allPrinciples []models.ConstitutionPrinciple

	for _, filePath := range filePaths {
		l.logger.WithField("file", filePath).Debug("Loading constitution file")
		principles, err := l.loadSingleConstitutionFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load constitution file %s: %w", filePath, err)
		}
		allPrinciples = append(allPrinciples, principles...)
	}

	// Basic validation for unique principle IDs
	seenIDs := make(map[string]bool)
	for _, p := range allPrinciples {
		if seenIDs[p.ID] {
			return nil, fmt.Errorf("duplicate constitution principle ID found: %s", p.ID)
		}
		seenIDs[p.ID] = true
	}

	return allPrinciples, nil
}

// loadSingleConstitutionFile reads and parses a single YAML constitution file.
func (l *ConstitutionLoader) loadSingleConstitutionFile(filePath string) ([]models.ConstitutionPrinciple, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config struct {
		Principles []models.ConstitutionPrinciple `yaml:"principles"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return config.Principles, nil
}
