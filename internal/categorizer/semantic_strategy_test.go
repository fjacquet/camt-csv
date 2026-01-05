package categorizer

import (
	"context"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestSemanticStrategy_Name(t *testing.T) {
	strategy := &SemanticStrategy{}
	assert.Equal(t, "Semantic", strategy.Name())
}

func TestSemanticStrategy_Categorize(t *testing.T) {
	// Setup categories
	categories := []models.CategoryConfig{
		{Name: "Food", Keywords: []string{"restaurant", "burger"}},
		{Name: "Transport", Keywords: []string{"train", "bus"}},
	}

	// Mock embeddings
	// Food: [1, 0, 0]
	// Transport: [0, 1, 0]
	embeddings := map[string][]float32{
		"Food: restaurant, burger": {1.0, 0.0, 0.0},
		"Transport: train, bus":    {0.0, 1.0, 0.0},
		"McDonalds Burger":         {0.9, 0.1, 0.0}, // Close to Food
		"SBB Ticket":               {0.1, 0.9, 0.0}, // Close to Transport
		"Unknown Stuff":            {0.0, 0.0, 1.0}, // Orthogonal to both
	}

	mockClient := &TestMockAIClient{
		GetEmbeddingFunc: func(ctx context.Context, text string) ([]float32, error) {
			if e, ok := embeddings[text]; ok {
				return e, nil
			}
			return []float32{0, 0, 0}, nil
		},
	}

	logger := &logging.MockLogger{}
	strategy := NewSemanticStrategy(mockClient, logger, categories)

	// Wait for initialization (since it runs in a goroutine)
	// In a real test we might want to expose a way to wait or check initialized status
	// For now, simple sleep or checking the boolean via reflection/helper
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name          string
		party         string
		desc          string
		expectedCat   string
		expectedFound bool
	}{
		{
			name:          "Match Food",
			party:         "McDonalds",
			desc:          "Burger",
			expectedCat:   "Food",
			expectedFound: true,
		},
		{
			name:          "Match Transport",
			party:         "SBB",
			desc:          "Ticket",
			expectedCat:   "Transport",
			expectedFound: true,
		},
		{
			name:          "No Match (Low Score)",
			party:         "Unknown",
			desc:          "Stuff",
			expectedCat:   "",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := Transaction{
				PartyName:   tt.party,
				Description: tt.desc,
			}
			cat, found, err := strategy.Categorize(context.Background(), tx)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedFound, found)
			if tt.expectedFound {
				assert.Equal(t, tt.expectedCat, cat.Name)
			}
		})
	}
}

func TestSemanticStrategy_CosineSimilarity(t *testing.T) {
	strategy := &SemanticStrategy{}

	tests := []struct {
		name     string
		a, b     []float32
		expected float32
	}{
		{"Identical", []float32{1, 0}, []float32{1, 0}, 1.0},
		{"Orthogonal", []float32{1, 0}, []float32{0, 1}, 0.0},
		{"Opposite", []float32{1, 0}, []float32{-1, 0}, -1.0},
		{"Similar", []float32{0.707, 0.707}, []float32{0.707, 0.707}, 1.0}, // approx
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := strategy.cosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.expected, score, 0.001)
		})
	}
}
