package reviewer

import (
	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAutomatedPrincipleEvaluator_Evaluate_Compliant(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	evaluator := NewAutomatedPrincipleEvaluator(logger)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main\nfunc main() { fmt.Println(\"Hello\") }\n",
	}
	principle := models.ConstitutionPrinciple{
		ID:               "GO-001",
		Name:             "Print Statement Check",
		Description:      "Code should contain fmt.Println.",
		Category:         "Style",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          "fmt.Println",
	}

	finding, err := evaluator.Evaluate(section, principle)
	assert.NoError(t, err)
	assert.Equal(t, models.FindingStatusCompliant, finding.Status)
	assert.Contains(t, finding.Details, "Pattern 'fmt.Println' found")
}

func TestAutomatedPrincipleEvaluator_Evaluate_NonCompliant(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	evaluator := NewAutomatedPrincipleEvaluator(logger)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main\nfunc main() {}\n",
	}
	principle := models.ConstitutionPrinciple{
		ID:               "GO-001",
		Name:             "Print Statement Check",
		Description:      "Code should contain fmt.Println.",
		Category:         "Style",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          "fmt.Println",
	}

	finding, err := evaluator.Evaluate(section, principle)
	assert.NoError(t, err)
	assert.Equal(t, models.FindingStatusNonCompliant, finding.Status)
	assert.Contains(t, finding.Details, "Pattern 'fmt.Println' not found")
}

func TestAutomatedPrincipleEvaluator_Evaluate_NotAutomated(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	evaluator := NewAutomatedPrincipleEvaluator(logger)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main\nfunc main() {}\n",
	}
	principle := models.ConstitutionPrinciple{
		ID:               "GO-002",
		Name:             "Manual Check",
		Description:      "This is a manual check.",
		Category:         "Style",
		EvaluationMethod: models.EvaluationMethodManual,
		Pattern:          "",
	}

	_, err := evaluator.Evaluate(section, principle)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "principle GO-002 is not an automated evaluation method")
}

func TestAutomatedPrincipleEvaluator_Evaluate_InvalidPattern(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	evaluator := NewAutomatedPrincipleEvaluator(logger)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main\nfunc main() {}\n",
	}
	principle := models.ConstitutionPrinciple{
		ID:               "GO-003",
		Name:             "Invalid Regex",
		Description:      "Uses an invalid regex.",
		Category:         "Error",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          "[", // Invalid regex
	}

	_, err := evaluator.Evaluate(section, principle)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex pattern for principle GO-003")
}

func TestAutomatedPrincipleEvaluator_Evaluate_NonFileSection(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	evaluator := NewAutomatedPrincipleEvaluator(logger)

	section := models.CodebaseSection{
		Path: "/test/dir",
		Type: models.CodebaseSectionTypeDirectory,
	}
	principle := models.ConstitutionPrinciple{
		ID:               "GO-001",
		Name:             "Print Statement Check",
		Description:      "Code should contain fmt.Println.",
		Category:         "Style",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          "fmt.Println",
	}

	_, err := evaluator.Evaluate(section, principle)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "automated evaluation only supports file sections")
}

func TestAutomatedPrincipleEvaluator_Evaluate_EmptyPattern(t *testing.T) {
	logger := logging.NewLogrusAdapter("info", "text")
	evaluator := NewAutomatedPrincipleEvaluator(logger)

	section := models.CodebaseSection{
		Path:    "/test/file.go",
		Type:    models.CodebaseSectionTypeFile,
		Content: "package main\nfunc main() {}\n",
	}
	principle := models.ConstitutionPrinciple{
		ID:               "GO-004",
		Name:             "Empty Pattern",
		Description:      "No pattern defined.",
		Category:         "Error",
		EvaluationMethod: models.EvaluationMethodAutomated,
		Pattern:          "", // Empty pattern
	}

	_, err := evaluator.Evaluate(section, principle)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "automated principle GO-004 has no pattern defined")
}
