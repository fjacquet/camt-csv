package categorizer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withGeminiServer points the Gemini client at a stub server for the duration
// of a test and restores the real endpoint afterwards.
func withGeminiServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	original := geminiAPIBaseURL
	geminiAPIBaseURL = server.URL
	t.Cleanup(func() {
		geminiAPIBaseURL = original
		server.Close()
	})
}

func TestGeminiClient_Defaults(t *testing.T) {
	client := NewGeminiClient(testLogger(), 0, "", 0, "test-key")

	assert.Equal(t, "gemini-2.0-flash", client.model)
	assert.Equal(t, "gemini", client.provider)
	assert.NotNil(t, client.limiter)
	assert.NotNil(t, client.httpClient)
}

func TestGeminiClient_CategorizeWithMockServer(t *testing.T) {
	var gotPrompt string

	withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-api-key", r.URL.Query().Get("key"))
		assert.Contains(t, r.URL.Path, ":generateContent")

		var req GeminiRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Len(t, req.Contents, 1)
		require.Len(t, req.Contents[0].Parts, 1)
		gotPrompt = req.Contents[0].Parts[0].Text

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(GeminiResponse{
			Candidates: []GeminiCandidate{
				{Content: GeminiContent{Parts: []GeminiPart{{Text: "**Courses**"}}}},
			},
		}))
	})

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "test-api-key")

	tx, err := client.Categorize(context.Background(), models.Transaction{
		PartyName:   "Coop Pronto",
		Description: "Card payment",
		Amount:      decimal.NewFromFloat(15.50),
	})

	require.NoError(t, err)
	assert.Equal(t, "Courses", tx.Category, "markdown bold must be stripped")
	assert.Contains(t, gotPrompt, "Party: Coop Pronto")
}

// An empty key must short-circuit before any HTTP call is attempted.
func TestGeminiClient_CategorizeWithEmptyAPIKey(t *testing.T) {
	called := false
	withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) { called = true })

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "")

	tx, err := client.Categorize(context.Background(), models.Transaction{PartyName: "Coop"})

	require.NoError(t, err)
	assert.False(t, called)
	assert.Equal(t, models.CategoryUncategorized, tx.Category)
}

// A permanent API error must surface, leave the transaction uncategorized, and
// never leak the URL — which carries the API key as a query parameter.
func TestGeminiClient_APIErrorIsReportedWithoutLeakingKey(t *testing.T) {
	withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	})

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "super-secret-key")

	tx, err := client.Categorize(context.Background(), models.Transaction{PartyName: "Coop"})

	require.Error(t, err)
	assert.Equal(t, models.CategoryUncategorized, tx.Category)
	assert.NotContains(t, err.Error(), "super-secret-key")
	assert.NotContains(t, err.Error(), "http://")
}

func TestGeminiClient_EmptyCandidatesIsAnError(t *testing.T) {
	withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(GeminiResponse{}))
	})

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "k")

	_, err := client.Categorize(context.Background(), models.Transaction{PartyName: "Coop"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no content in API response")
}

func TestGeminiClient_GetEmbedding(t *testing.T) {
	withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "gemini-embedding-001:embedContent")

		var req GeminiEmbeddingRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "Coop Pronto", req.Content.Parts[0].Text)

		require.NoError(t, json.NewEncoder(w).Encode(GeminiEmbeddingResponse{
			Embedding: GeminiEmbeddingValues{Values: []float32{0.1, 0.2, 0.3}},
		}))
	})

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "k")

	vec, err := client.GetEmbedding(context.Background(), "Coop Pronto")

	require.NoError(t, err)
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, vec)
}

func TestGeminiClient_GetEmbeddingWithoutAPIKey(t *testing.T) {
	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "")

	_, err := client.GetEmbedding(context.Background(), "Coop")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key not set")
}

func TestGeminiClient_EmptyEmbeddingIsAnError(t *testing.T) {
	withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(GeminiEmbeddingResponse{}))
	})

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "k")

	_, err := client.GetEmbedding(context.Background(), "Coop")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty embedding")
}

// A 503 must be retried; the stub fails twice then succeeds.
func TestGeminiClient_RetriesServerErrors(t *testing.T) {
	calls := 0
	withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(GeminiResponse{
			Candidates: []GeminiCandidate{
				{Content: GeminiContent{Parts: []GeminiPart{{Text: "Restaurants"}}}},
			},
		}))
	})

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "k")

	tx, err := client.Categorize(context.Background(), models.Transaction{PartyName: "McDonalds"})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
	assert.Equal(t, "Restaurants", tx.Category)
}

// The security contract: neither the key nor the URL may appear in any log line.
func TestGeminiClient_NeverLogsCredentials(t *testing.T) {
	withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	})

	logger := logging.NewMockLogger()
	client := NewGeminiClient(logger, 600, "gemini-2.0-flash", 30, "super-secret-key")

	_, _ = client.Categorize(context.Background(), models.Transaction{PartyName: "Coop"})

	entries := logger.GetEntries()
	require.NotEmpty(t, entries, "the failing call must have logged something to check")

	for _, entry := range entries {
		line := entry.Message
		for _, f := range entry.Fields {
			line += fmt.Sprintf(" %s=%v", f.Key, f.Value)
		}
		if entry.Error != nil {
			line += " err=" + entry.Error.Error()
		}

		assert.NotContains(t, line, "super-secret-key", "API key must never be logged")
		assert.False(t, strings.Contains(line, "http://") || strings.Contains(line, "https://"),
			"URLs may embed the API key and must never be logged: %s", line)
	}
}
