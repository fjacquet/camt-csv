package reviewer

import (
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestNewAutomatedPrincipleEvaluator(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	evaluator := NewAutomatedPrincipleEvaluator(logger)

	assert.NotNil(t, evaluator)
	assert.NotNil(t, evaluator.logger)
}

func TestNewAutomatedPrincipleEvaluatorWithNilLogger(t *testing.T) {
	evaluator := NewAutomatedPrincipleEvaluator(nil)

	assert.NotNil(t, evaluator)
	assert.NotNil(t, evaluator.logger) // Should create default logger
}

func TestAutomatedPrincipleEvaluator_Evaluate_Compliant(t *testing.T) {
	evaluator := NewAutomatedPrincipleEvaluator(nil)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}",
	}

	principle := models.ConstitutionPrinciple{
		ID:               "GO-001",
		Name:             "Package Declaration",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          `package\s+\w+`,
	}

	finding, err := evaluator.Evaluate(section, principle)

	assert.NoError(t, err)
	assert.Equal(t, models.FindingStatusCompliant, finding.Status)
	assert.Equal(t, principle, finding.Principle)
	assert.Contains(t, finding.Details, "Pattern 'package\\s+\\w+' found in /test/file.go")
}

func TestAutomatedPrincipleEvaluator_Evaluate_NonCompliant(t *testing.T) {
	evaluator := NewAutomatedPrincipleEvaluator(nil)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "// This file has no declaration\n\nfunc test() {\n\tfmt.Println(\"Hello, World!\")\n}",
	}

	principle := models.ConstitutionPrinciple{
		ID:               "GO-001",
		Name:             "Package Declaration",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          `package\s+\w+`,
	}

	finding, err := evaluator.Evaluate(section, principle)

	assert.NoError(t, err)
	assert.Equal(t, models.FindingStatusNonCompliant, finding.Status)
	assert.Equal(t, principle, finding.Principle)
	assert.Contains(t, finding.Details, "Pattern 'package\\s+\\w+' not found in /test/file.go")
}

func TestAutomatedPrincipleEvaluator_Evaluate_NotAutomated(t *testing.T) {
	evaluator := NewAutomatedPrincipleEvaluator(nil)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main",
	}

	principle := models.ConstitutionPrinciple{
		ID:               "MANUAL-001",
		Name:             "Manual Review Required",
		EvaluationMethod: models.EvaluationMethodManual,
		Pattern:          `package\s+\w+`,
	}

	_, err := evaluator.Evaluate(section, principle)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "principle MANUAL-001 is not an automated evaluation method")
}

func TestAutomatedPrincipleEvaluator_Evaluate_InvalidPattern(t *testing.T) {
	evaluator := NewAutomatedPrincipleEvaluator(nil)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main",
	}

	principle := models.ConstitutionPrinciple{
		ID:               "GO-003",
		Name:             "Invalid Pattern",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          `[`, // Invalid regex pattern
	}

	_, err := evaluator.Evaluate(section, principle)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex pattern for principle GO-003")
}

func TestAutomatedPrincipleEvaluator_Evaluate_NonFileSection(t *testing.T) {
	evaluator := NewAutomatedPrincipleEvaluator(nil)

	section := models.CodebaseSection{
		Path: "/test/directory",
		Type: models.CodebaseSectionTypeDirectory,
	}

	principle := models.ConstitutionPrinciple{
		ID:               "GO-001",
		Name:             "Package Declaration",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          `package\s+\w+`,
	}

	_, err := evaluator.Evaluate(section, principle)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "automated evaluation only supports file sections")
	assert.Contains(t, err.Error(), "got Directory for /test/directory")
}

func TestAutomatedPrincipleEvaluator_Evaluate_EmptyPattern(t *testing.T) {
	evaluator := NewAutomatedPrincipleEvaluator(nil)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main",
	}

	principle := models.ConstitutionPrinciple{
		ID:               "GO-002",
		Name:             "Empty Pattern",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          "", // Empty pattern
	}

	_, err := evaluator.Evaluate(section, principle)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "automated principle GO-002 has no pattern defined")
}

func TestAutomatedPrincipleEvaluator_Evaluate_ComplexPattern(t *testing.T) {
	evaluator := NewAutomatedPrincipleEvaluator(nil)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}",
	}

	principle := models.ConstitutionPrinciple{
		ID:               "GO-004",
		Name:             "Import Statement",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          `import\s+"[^"]+"|import\s+\([^)]+\)`,
	}

	finding, err := evaluator.Evaluate(section, principle)

	assert.NoError(t, err)
	assert.Equal(t, models.FindingStatusCompliant, finding.Status)
	assert.Equal(t, principle, finding.Principle)
	assert.Contains(t, finding.Details, "Pattern")
	assert.Contains(t, finding.Details, "found in /test/file.go")
}

func TestAutomatedPrincipleEvaluator_Evaluate_CaseInsensitivePattern(t *testing.T) {
	evaluator := NewAutomatedPrincipleEvaluator(nil)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "PACKAGE main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}",
	}

	principle := models.ConstitutionPrinciple{
		ID:               "GO-005",
		Name:             "Case Insensitive Package",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          `(?i)package\s+\w+`, // Case insensitive pattern
	}

	finding, err := evaluator.Evaluate(section, principle)

	assert.NoError(t, err)
	assert.Equal(t, models.FindingStatusCompliant, finding.Status)
	assert.Equal(t, principle, finding.Principle)
}

func TestAutomatedPrincipleEvaluator_Evaluate_MultilinePattern(t *testing.T) {
	evaluator := NewAutomatedPrincipleEvaluator(nil)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main\n\n// This is a comment\n// spanning multiple lines\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}",
	}

	principle := models.ConstitutionPrinciple{
		ID:               "GO-006",
		Name:             "Multiline Comment",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          `(?s)//.*\n//.*`, // Multiline pattern
	}

	finding, err := evaluator.Evaluate(section, principle)

	assert.NoError(t, err)
	assert.Equal(t, models.FindingStatusCompliant, finding.Status)
	assert.Equal(t, principle, finding.Principle)
}
