package parser

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestConstitutionLoader_LoadConstitutionFiles_Valid(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	loader := NewConstitutionLoader(logger)

	// Create temporary valid constitution files
	tempDir := t.TempDir()

	file1Path := filepath.Join(tempDir, "constitution1.yaml")
	file1Content := []byte(`
principles:
  - id: GO-001
    name: "Error Handling"
    description: "Errors must be handled."
    category: "Quality"
    evaluation_method: "Automated"
    pattern: "_ = err"
`)
	err := os.WriteFile(file1Path, file1Content, 0600)
	assert.NoError(t, err)

	file2Path := filepath.Join(tempDir, "constitution2.yaml")
	file2Content := []byte(`
principles:
  - id: GO-002
    name: "Naming Conventions"
    description: "Follow Go naming conventions."
    category: "Style"
    evaluation_method: "Manual"
`)
	err = os.WriteFile(file2Path, file2Content, 0600)
	assert.NoError(t, err)

	principles, err := loader.LoadConstitutionFiles([]string{file1Path, file2Path})
	assert.NoError(t, err)
	assert.Len(t, principles, 2)

	assert.Equal(t, "GO-001", principles[0].ID)
	assert.Equal(t, models.EvaluationMethodAutomated, principles[0].EvaluationMethod)
	assert.Equal(t, "_ = err", principles[0].Pattern)

	assert.Equal(t, "GO-002", principles[1].ID)
	assert.Equal(t, models.EvaluationMethodManual, principles[1].EvaluationMethod)
	assert.Empty(t, principles[1].Pattern)
}

func TestConstitutionLoader_LoadConstitutionFiles_InvalidYAML(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	loader := NewConstitutionLoader(logger)

	// Create a temporary invalid YAML file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.yaml")
	content := []byte(`
principles:
  - id: GO-001
    name: "Error Handling"
    description: "Errors must be handled."
    category: "Quality"
    evaluation_method: "Automated"
    pattern: "_ = err"
  - id: GO-001 # Duplicate ID
    name: "Another Error Handling"
    description: "Another error handling rule."
    category: "Quality"
    evaluation_method: "Automated"
    pattern: "_ = err"
`)
	err := os.WriteFile(filePath, content, 0600)
	assert.NoError(t, err)

	_, err = loader.LoadConstitutionFiles([]string{filePath})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate constitution principle ID found: GO-001")
}

func TestConstitutionLoader_LoadConstitutionFiles_NonExistentFile(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	loader := NewConstitutionLoader(logger)

	_, err := loader.LoadConstitutionFiles([]string{"/non/existent/constitution.yaml"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestConstitutionLoader_LoadConstitutionFiles_EmptyFiles(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	loader := NewConstitutionLoader(logger)

	principles, err := loader.LoadConstitutionFiles([]string{})
	assert.NoError(t, err)
	assert.Len(t, principles, 0)
}
