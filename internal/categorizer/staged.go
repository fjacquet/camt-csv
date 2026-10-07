package categorizer

import (
	"context"
	"strings"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// StagedStrategy answers from AI suggestions staged by earlier runs. They are
// unreviewed, so they rank below the user's own mappings and keywords, but
// above the semantic and AI tiers that would otherwise pay to find the same
// answer again. It never writes: promotion to the mapping files stays manual.
type StagedStrategy struct {
	creditors map[string]string
	debtors   map[string]string
	logger    logging.Logger
}

// NewStagedStrategy builds the tier from lowercased party-name maps.
func NewStagedStrategy(creditors, debtors map[string]string, logger logging.Logger) *StagedStrategy {
	return &StagedStrategy{creditors: creditors, debtors: debtors, logger: logger}
}

// Name returns the strategy name for logs.
func (s *StagedStrategy) Name() string { return "Staged" }

// Categorize looks the party up in the staged suggestions for its direction.
func (s *StagedStrategy) Categorize(_ context.Context, tx Transaction) (models.Category, bool, error) {
	key := strings.ToLower(strings.TrimSpace(tx.PartyName))
	if key == "" {
		return models.Category{}, false, nil
	}
	mappings := s.creditors
	if tx.IsDebtor {
		mappings = s.debtors
	}
	name, ok := mappings[key]
	if !ok || name == "" {
		return models.Category{}, false, nil
	}
	return models.Category{
		Name:        name,
		Description: categoryDescriptionFromName(name),
		Confidence:  0.85,
		Source:      "staged",
	}, true, nil
}
