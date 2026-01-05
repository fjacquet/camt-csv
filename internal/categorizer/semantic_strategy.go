package categorizer

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// SemanticStrategy implements CategorizationStrategy using vector embeddings.
// It matches transactions to categories by comparing the semantic similarity
// of the transaction description with the category's keywords.
type SemanticStrategy struct {
	client             AIClient
	log                logging.Logger
	categoryEmbeddings map[string][]float32
	threshold          float32
	mu                 sync.RWMutex
	initialized        bool
}

// NewSemanticStrategy creates a new SemanticStrategy instance.
func NewSemanticStrategy(client AIClient, logger logging.Logger, categories []models.CategoryConfig) *SemanticStrategy {
	s := &SemanticStrategy{
		client:             client,
		log:                logger,
		categoryEmbeddings: make(map[string][]float32),
		threshold:          0.70, // Default threshold
	}

	// Initialize embeddings in background to not block startup
	if client != nil {
		go s.initializeEmbeddings(context.Background(), categories)
	}

	return s
}

// Name returns the name of the strategy.
func (s *SemanticStrategy) Name() string {
	return "Semantic"
}

// Categorize attempts to categorize the transaction using semantic similarity.
func (s *SemanticStrategy) Categorize(ctx context.Context, tx Transaction) (models.Category, bool, error) {
	if s.client == nil {
		return models.Category{}, false, nil
	}

	s.mu.RLock()
	if !s.initialized {
		s.mu.RUnlock()
		return models.Category{}, false, nil // Skip if not ready
	}
	defer s.mu.RUnlock()

	// Construct text to embed: Party Name + Description
	textToEmbed := fmt.Sprintf("%s %s", tx.PartyName, tx.Description)
	textToEmbed = strings.TrimSpace(textToEmbed)
	if textToEmbed == "" {
		return models.Category{}, false, nil
	}

	// Get embedding for transaction
	txEmbedding, err := s.client.GetEmbedding(ctx, textToEmbed)
	if err != nil {
		s.log.WithError(err).Warn("Failed to get embedding for transaction")
		return models.Category{}, false, nil // Fail gracefully
	}

	var bestCategory string
	var maxScore float32 = -1.0

	// Find best matching category
	for catName, catEmbedding := range s.categoryEmbeddings {
		score := s.cosineSimilarity(txEmbedding, catEmbedding)
		if score > maxScore {
			maxScore = score
			bestCategory = catName
		}
	}

	if maxScore >= s.threshold {
		s.log.WithFields(
			logging.Field{Key: "category", Value: bestCategory},
			logging.Field{Key: "score", Value: maxScore},
			logging.Field{Key: "party", Value: tx.PartyName},
		).Debug("Semantic match found")
		return models.Category{Name: bestCategory, Description: "Semantic match"}, true, nil
	}

	return models.Category{}, false, nil
}

// initializeEmbeddings generates embeddings for all categories.
func (s *SemanticStrategy) initializeEmbeddings(ctx context.Context, categories []models.CategoryConfig) {
	s.log.Info("Initializing semantic embeddings...")

	tempEmbeddings := make(map[string][]float32)

	for _, cat := range categories {
		// Construct representative text: "Name: keyword1, keyword2, ..."
		keywords := strings.Join(cat.Keywords, ", ")
		text := fmt.Sprintf("%s: %s", cat.Name, keywords)

		embedding, err := s.client.GetEmbedding(ctx, text)
		if err != nil {
			s.log.WithError(err).WithFields(
				logging.Field{Key: "category", Value: cat.Name},
			).Warn("Failed to generate embedding for category")
			continue
		}
		tempEmbeddings[cat.Name] = embedding
	}

	s.mu.Lock()
	s.categoryEmbeddings = tempEmbeddings
	s.initialized = true
	s.mu.Unlock()

	s.log.WithFields(
		logging.Field{Key: "count", Value: len(tempEmbeddings)},
	).Info("Semantic embeddings initialized")
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func (s *SemanticStrategy) cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
