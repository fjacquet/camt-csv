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
	// client is set once at construction and never reassigned. Warm-up runs in
	// a background goroutine that reads it concurrently with Categorize, so a
	// later swap would be a data race; wire the right client up front instead.
	client             AIClient
	log                logging.Logger
	categoryEmbeddings map[string][]float32
	threshold          float32
	mu                 sync.RWMutex
	initialized        bool

	// Disk cache for category embeddings (nil = no caching)
	diskCache  *EmbeddingCache
	categories []models.CategoryConfig

	// Per-transaction embedding cache (in-memory, per-run)
	txEmbCache sync.Map // key: string → value: []float32

	// Warm-up lifecycle. cancelWarmup stops the background embedding warm-up;
	// warmupDone closes once it has returned. Both are nil when no warm-up ran.
	// The warm-up starts on the first Categorize call, so a run that never
	// reaches tier 3 spends no embedding calls.
	warmupOnce   sync.Once
	cancelWarmup context.CancelFunc
	warmupDone   chan struct{}
}

// NewSemanticStrategy creates a new SemanticStrategy instance.
func NewSemanticStrategy(client AIClient, logger logging.Logger, categories []models.CategoryConfig, threshold float32) *SemanticStrategy {
	return NewSemanticStrategyWithCache(client, logger, categories, threshold, nil)
}

// NewSemanticStrategyWithCache creates a SemanticStrategy with an optional disk-based embedding cache.
func NewSemanticStrategyWithCache(client AIClient, logger logging.Logger, categories []models.CategoryConfig, threshold float32, diskCache *EmbeddingCache) *SemanticStrategy {
	if threshold <= 0 {
		threshold = 0.70
	}
	s := &SemanticStrategy{
		client:             client,
		log:                logger,
		categoryEmbeddings: make(map[string][]float32),
		threshold:          threshold,
		diskCache:          diskCache,
		categories:         categories,
	}

	return s
}

// startWarmup embeds the categories in the background, once. The context is
// owned by this strategy rather than being Background(), so Shutdown can stop
// an in-flight warm-up instead of leaving it hammering the embedding API while
// the process is trying to exit.
func (s *SemanticStrategy) startWarmup() {
	s.warmupOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		s.mu.Lock()
		s.cancelWarmup = cancel
		s.warmupDone = done
		s.mu.Unlock()
		go func() {
			defer close(done)
			s.initializeEmbeddings(ctx, s.categories)
		}()
	})
}

// Shutdown cancels any in-flight embedding warm-up and waits for it to stop.
// It is safe to call on a strategy that never started one, and safe to call
// more than once.
func (s *SemanticStrategy) Shutdown() {
	// Burn the Once so a lookup after Shutdown cannot start a warm-up nobody
	// would cancel.
	s.warmupOnce.Do(func() {})
	s.mu.RLock()
	cancel, done := s.cancelWarmup, s.warmupDone
	s.mu.RUnlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
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

	s.startWarmup()
	s.mu.RLock()
	done := s.warmupDone
	s.mu.RUnlock()
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return models.Category{}, false, ctx.Err()
		}
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

	// Check per-transaction embedding cache
	var txEmbedding []float32
	if cached, ok := s.txEmbCache.Load(textToEmbed); ok {
		txEmbedding = cached.([]float32)
	} else {
		// Get embedding for transaction
		var err error
		txEmbedding, err = s.client.GetEmbedding(ctx, textToEmbed)
		if err != nil {
			s.log.WithError(err).Warn("Failed to get embedding for transaction")
			return models.Category{}, false, nil // Fail gracefully
		}
		s.txEmbCache.Store(textToEmbed, txEmbedding)
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

	// Log near-miss scores for tuning
	if maxScore >= s.threshold-0.10 && maxScore < s.threshold {
		s.log.WithFields(
			logging.Field{Key: "party", Value: tx.PartyName},
			logging.Field{Key: "best_category", Value: bestCategory},
			logging.Field{Key: "score", Value: maxScore},
			logging.Field{Key: "threshold", Value: s.threshold},
		).Debug("Semantic near-miss: score below threshold")
	}

	if maxScore >= s.threshold {
		s.log.WithFields(
			logging.Field{Key: "category", Value: bestCategory},
			logging.Field{Key: "score", Value: maxScore},
			logging.Field{Key: "party", Value: tx.PartyName},
		).Debug("Semantic match found")
		return models.Category{
			Name:        bestCategory,
			Description: "Semantic match",
			Confidence:  0.90, // High confidence for semantic matches above threshold
			Source:      "semantic",
		}, true, nil
	}

	return models.Category{}, false, nil
}

// initializeEmbeddings generates embeddings for all categories.
func (s *SemanticStrategy) initializeEmbeddings(ctx context.Context, categories []models.CategoryConfig) {
	s.log.Info("Initializing semantic embeddings...")

	// Resume from whatever the disk cache holds. It may be partial: an earlier run
	// can have been cut off, so only the categories it lacks are embedded.
	hash := ComputeHash(categories)
	tempEmbeddings := make(map[string][]float32)
	if s.diskCache != nil {
		if cached, ok := s.diskCache.Load(hash); ok {
			tempEmbeddings = cached
		}
	}

	// Keep what was computed even when interrupted, so a short run still makes
	// progress and the next one finishes the job.
	computed := 0
	save := func() {
		if s.diskCache != nil && computed > 0 {
			if err := s.diskCache.Save(hash, tempEmbeddings); err != nil {
				s.log.WithError(err).Warn("Failed to save embedding cache")
			}
		}
	}

	for _, cat := range categories {
		if _, done := tempEmbeddings[cat.Name]; done {
			continue
		}

		// Stop promptly on shutdown rather than working through every remaining
		// category; each one is a network round trip to the embedding provider.
		if ctx.Err() != nil {
			s.log.Debug("Embedding warm-up cancelled")
			save()
			return
		}

		// Construct representative text: "Name: keyword1, keyword2, ..."
		keywords := strings.Join(cat.Keywords, ", ")
		text := fmt.Sprintf("%s: %s", cat.Name, keywords)

		embedding, err := s.client.GetEmbedding(ctx, text)
		if err != nil {
			// Shutdown aborted this call: an orderly stop, not a provider failure.
			if ctx.Err() != nil {
				s.log.Debug("Embedding warm-up cancelled")
				save()
				return
			}
			s.log.WithError(err).WithFields(
				logging.Field{Key: "category", Value: cat.Name},
			).Warn("Failed to generate embedding for category")
			continue
		}
		tempEmbeddings[cat.Name] = embedding
		computed++
	}

	save()

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
