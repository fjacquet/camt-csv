package categorizer

import (
	"fmt"
	"strings"

	"fjacquet/camt-csv/internal/models"
)

// StrategyResult represents the result of a categorization strategy attempt
type StrategyResult struct {
	Strategy   string
	Category   models.Category
	Found      bool
	Error      error
	Confidence float64 // 0.0 to 1.0, for future use in weighted strategies
}

// StrategyResults aggregates results from multiple strategies
type StrategyResults struct {
	Results []StrategyResult
}

// GetBestResult returns the first successful result or the highest confidence result
func (sr StrategyResults) GetBestResult() (models.Category, bool) {
	var bestResult *StrategyResult
	
	// First, look for any successful result (found = true)
	for i := range sr.Results {
		if sr.Results[i].Found && sr.Results[i].Error == nil {
			return sr.Results[i].Category, true
		}
	}
	
	// If no successful result, find the highest confidence result
	for i := range sr.Results {
		if sr.Results[i].Error == nil {
			if bestResult == nil || sr.Results[i].Confidence > bestResult.Confidence {
				bestResult = &sr.Results[i]
			}
		}
	}
	
	if bestResult != nil {
		return bestResult.Category, bestResult.Found
	}
	
	return models.Category{}, false
}

// GetErrors returns all errors encountered during strategy execution
func (sr StrategyResults) GetErrors() []error {
	var errors []error
	for _, result := range sr.Results {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("%s strategy: %w", result.Strategy, result.Error))
		}
	}
	return errors
}

// Summary returns a human-readable summary of all strategy attempts
func (sr StrategyResults) Summary() string {
	var parts []string
	for _, result := range sr.Results {
		status := "failed"
		if result.Found {
			status = "success"
		} else if result.Error == nil {
			status = "no_match"
		}
		parts = append(parts, fmt.Sprintf("%s:%s", result.Strategy, status))
	}
	return strings.Join(parts, ", ")
}