package categorizer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
)

// embeddingCacheData is the on-disk format for persisted category embeddings.
type embeddingCacheData struct {
	Hash       string               `json:"hash"`
	Embeddings map[string][]float32 `json:"embeddings"`
}

// EmbeddingCache persists category embeddings to disk to avoid recomputing
// them on every startup (~50 API calls).
type EmbeddingCache struct {
	path   string
	logger logging.Logger
}

// NewEmbeddingCache creates a new EmbeddingCache at the given directory.
// Pass "" to use the default ~/.camt-csv/ directory.
func NewEmbeddingCache(dir string, logger logging.Logger) *EmbeddingCache {
	if dir == "" || dir == "~/.camt-csv" {
		home, err := os.UserHomeDir()
		if err != nil {
			logger.WithError(err).Warn("Cannot determine home directory for embedding cache")
			return &EmbeddingCache{path: "", logger: logger}
		}
		dir = filepath.Join(home, ".camt-csv")
	}
	return &EmbeddingCache{
		path:   filepath.Join(dir, "embedding_cache.json"),
		logger: logger,
	}
}

// ComputeHash returns a deterministic SHA-256 hash of category names and keywords.
// Any change to categories.yaml busts the cache.
func ComputeHash(categories []models.CategoryConfig) string {
	// Sort categories by name for determinism
	sorted := make([]models.CategoryConfig, len(categories))
	copy(sorted, categories)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	var sb strings.Builder
	for _, cat := range sorted {
		kw := make([]string, len(cat.Keywords))
		copy(kw, cat.Keywords)
		sort.Strings(kw)
		sb.WriteString(cat.Name)
		sb.WriteString(":")
		sb.WriteString(strings.Join(kw, ","))
		sb.WriteString("\n")
	}

	hash := sha256.Sum256([]byte(sb.String()))
	return fmt.Sprintf("%x", hash)
}

// Load attempts to read cached embeddings from disk.
// Returns nil, false if the cache file doesn't exist, is corrupted, or the hash doesn't match.
func (c *EmbeddingCache) Load(hash string) (map[string][]float32, bool) {
	if c.path == "" {
		return nil, false
	}

	data, err := os.ReadFile(c.path)
	if err != nil {
		if !os.IsNotExist(err) {
			c.logger.WithError(err).Debug("Could not read embedding cache file")
		}
		return nil, false
	}

	var cache embeddingCacheData
	if err := json.Unmarshal(data, &cache); err != nil {
		c.logger.WithError(err).Warn("Embedding cache file corrupted, will regenerate")
		return nil, false
	}

	if cache.Hash != hash {
		c.logger.Debug("Embedding cache hash mismatch, categories changed")
		return nil, false
	}

	if len(cache.Embeddings) == 0 {
		return nil, false
	}

	c.logger.WithFields(
		logging.Field{Key: "count", Value: len(cache.Embeddings)},
	).Info("Loaded category embeddings from cache")
	return cache.Embeddings, true
}

// Save writes embeddings to disk atomically (write temp file, then rename).
func (c *EmbeddingCache) Save(hash string, embeddings map[string][]float32) error {
	if c.path == "" {
		return nil
	}

	cache := embeddingCacheData{
		Hash:       hash,
		Embeddings: embeddings,
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("marshal embedding cache: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, models.PermissionDirectory); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	// Atomic write: temp file in same directory + rename (safe under concurrent processes)
	tmpFile, err := os.CreateTemp(dir, "embedding_cache_*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp cache file: %w", err)
	}
	tmp := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("write temp cache file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close temp cache file: %w", err)
	}

	if err := os.Rename(tmp, c.path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename cache file: %w", err)
	}

	c.logger.WithFields(
		logging.Field{Key: "count", Value: len(embeddings)},
		logging.Field{Key: "path", Value: c.path},
	).Info("Saved category embeddings to cache")
	return nil
}
