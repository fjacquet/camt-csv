package categorizer

import (
	"context"
	"strings"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// aiBatchSize is how many parties one AI request carries.
const aiBatchSize = 25

var _ models.BatchCategorizer = (*Categorizer)(nil)

// CategorizeBatch categorizes many transactions, asking the AI once per chunk
// of parties the local tiers could not place instead of once per party.
// Results are in request order and go through the same cache, validation,
// learning and staging as single requests.
func (c *Categorizer) CategorizeBatch(ctx context.Context, requests []models.CategorizeRequest) ([]models.Category, error) {
	results := make([]models.Category, len(requests))
	pending := map[string][]int{} // cache key -> request indexes
	var order []string            // first-seen order of pending keys

	for i, req := range requests {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if category, ok := c.categorizeWithoutAI(ctx, toTransaction(req)); ok {
			results[i] = category
			continue
		}
		key := cacheKey(req.PartyName, req.IsDebtor)
		if _, seen := pending[key]; !seen {
			order = append(order, key)
		}
		pending[key] = append(pending[key], i)
	}

	batcher, canBatch := c.aiClient.(BatchAIClient)
	for _, isDebtor := range []bool{true, false} {
		var keys []string
		for _, key := range order {
			if requests[pending[key][0]].IsDebtor == isDebtor {
				keys = append(keys, key)
			}
		}
		for start := 0; start < len(keys); start += aiBatchSize {
			chunk := keys[start:min(start+aiBatchSize, len(keys))]
			if err := c.answerChunk(ctx, batcher, canBatch, requests, pending, chunk, results); err != nil {
				return nil, err
			}
		}
	}

	return results, nil
}

func toTransaction(req models.CategorizeRequest) Transaction {
	return Transaction{PartyName: req.PartyName, IsDebtor: req.IsDebtor, Amount: req.Amount, Date: req.Date, Info: req.Info}
}

// categorizeWithoutAI answers from the cache or the local tiers, reporting
// false when only the AI is left. A blank party is answered as Uncategorized.
func (c *Categorizer) categorizeWithoutAI(ctx context.Context, tx Transaction) (models.Category, bool) {
	if strings.TrimSpace(tx.PartyName) == "" {
		return models.Category{Name: models.CategoryUncategorized, Description: "No party name provided"}, true
	}
	key := cacheKey(tx.PartyName, tx.IsDebtor)
	c.batchCacheMu.RLock()
	cached, ok := c.batchCache[key]
	c.batchCacheMu.RUnlock()
	if ok {
		return cached, true
	}
	for _, strategy := range c.strategies {
		if _, isAI := strategy.(*AIStrategy); isAI {
			continue
		}
		category, found, err := strategy.Categorize(ctx, tx)
		if err != nil || !found {
			continue
		}
		c.storeInCache(key, category)
		return category, true
	}
	return models.Category{}, false
}

// answerChunk asks the AI about one chunk of same-direction parties, then
// falls back to single requests for any party the answer left out.
func (c *Categorizer) answerChunk(ctx context.Context, batcher BatchAIClient, canBatch bool, requests []models.CategorizeRequest, pending map[string][]int, chunk []string, results []models.Category) error {
	answers := map[string]string{}
	if canBatch {
		txs := make([]models.Transaction, 0, len(chunk))
		for _, key := range chunk {
			req := requests[pending[key][0]]
			txs = append(txs, models.Transaction{PartyName: req.PartyName, Description: req.Info, Amount: models.ParseAmount(req.Amount)})
		}
		got, err := batcher.CategorizeBatch(ctx, txs)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			c.logger.WithError(err).Warn("Batch AI categorization failed, falling back to single requests",
				logging.Field{Key: "parties", Value: len(chunk)})
		} else {
			answers = got
			var missing int
			for _, key := range chunk {
				party := strings.ToLower(strings.TrimSpace(oneLine(requests[pending[key][0]].PartyName)))
				if _, ok := answers[party]; !ok {
					missing++
				}
			}
			if missing > 0 {
				c.logger.Info("Batch AI answer left out parties; asking them one by one",
					logging.Field{Key: "missing", Value: missing},
					logging.Field{Key: "chunk_size", Value: len(chunk)})
			}
		}
	}

	for _, key := range chunk {
		req := requests[pending[key][0]]
		var category models.Category
		if answer, ok := answers[strings.ToLower(strings.TrimSpace(oneLine(req.PartyName)))]; ok {
			category = c.aiResult(req.PartyName, models.Category{Name: answer, Confidence: 0.8})
		} else {
			category = c.singleAI(ctx, toTransaction(req))
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
		}
		// Write results directly: recordLearning may invalidate the cache entry.
		for _, i := range pending[key] {
			results[i] = category
		}
		c.storeInCache(key, category)
		c.recordLearning(req.PartyName, req.IsDebtor, category)
	}
	return nil
}

// aiResult validates a raw AI answer against categories.yaml, returning the
// canonical spelling (keeping the answer's confidence), or Uncategorized when
// the answer is empty or not a known category.
func (c *Categorizer) aiResult(partyName string, answer models.Category) models.Category {
	if answer.Name == "" || answer.Name == models.CategoryUncategorized {
		return models.Category{Name: models.CategoryUncategorized, Source: "ai"}
	}
	canonical, ok := c.canonicalAICategory(answer.Name)
	if !ok {
		c.logger.WithFields(
			logging.Field{Key: "party", Value: partyName},
			logging.Field{Key: "answer", Value: answer.Name},
		).Warn("Rejected AI category not in categories.yaml")
		return models.Category{Name: models.CategoryUncategorized, Source: "ai"}
	}
	answer.Name = canonical
	if answer.Description == "" {
		answer.Description = categoryDescriptionFromName(canonical)
	}
	answer.Source = "ai"
	return answer
}

// singleAI asks the AI tier alone about one party.
func (c *Categorizer) singleAI(ctx context.Context, tx Transaction) models.Category {
	for _, strategy := range c.strategies {
		ai, ok := strategy.(*AIStrategy)
		if !ok {
			continue
		}
		category, found, err := ai.Categorize(ctx, tx)
		if err != nil || !found {
			return models.Category{Name: models.CategoryUncategorized, Source: "ai"}
		}
		return c.aiResult(tx.PartyName, category)
	}
	return models.Category{Name: models.CategoryUncategorized}
}

func (c *Categorizer) storeInCache(key string, category models.Category) {
	c.batchCacheMu.Lock()
	c.batchCache[key] = category
	c.batchCacheMu.Unlock()
}
