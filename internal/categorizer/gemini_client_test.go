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

// withGeminiServer starts a stub Gemini endpoint and returns its URL, to be
// passed to NewGeminiClient. Nothing global is touched, so these tests stay
// independent of each other.
func withGeminiServer(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server.URL
}

func TestGeminiClient_Defaults(t *testing.T) {
	client := NewGeminiClient(testLogger(), 0, "", 0, "test-key", "")

	assert.Equal(t, "gemini-2.0-flash", client.model)
	assert.Equal(t, "gemini", client.provider)
	assert.Equal(t, defaultGeminiAPIBaseURL, client.baseURL, "an empty baseURL must fall back to the real endpoint")
	assert.NotNil(t, client.limiter)
	assert.NotNil(t, client.httpClient)
}

func TestGeminiClient_CategorizeWithMockServer(t *testing.T) {
	var gotPrompt string

	baseURL := withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-api-key", r.Header.Get("x-goog-api-key"))
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

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "test-api-key", baseURL)

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
	baseURL := withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) { called = true })

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "", baseURL)

	tx, err := client.Categorize(context.Background(), models.Transaction{PartyName: "Coop"})

	require.NoError(t, err)
	assert.False(t, called)
	assert.Equal(t, models.CategoryUncategorized, tx.Category)
}

// A permanent API error must surface, leave the transaction uncategorized, and
// never leak the URL — which carries the API key as a query parameter.
func TestGeminiClient_APIErrorIsReportedWithoutLeakingKey(t *testing.T) {
	baseURL := withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	})

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "super-secret-key", baseURL)

	tx, err := client.Categorize(context.Background(), models.Transaction{PartyName: "Coop"})

	require.Error(t, err)
	assert.Equal(t, models.CategoryUncategorized, tx.Category)
	assert.NotContains(t, err.Error(), "super-secret-key")
	assert.NotContains(t, err.Error(), "http://")
}

func TestGeminiClient_EmptyCandidatesIsAnError(t *testing.T) {
	baseURL := withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(GeminiResponse{}))
	})

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "k", baseURL)

	_, err := client.Categorize(context.Background(), models.Transaction{PartyName: "Coop"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no content in API response")
}

func TestGeminiClient_GetEmbedding(t *testing.T) {
	baseURL := withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "gemini-embedding-001:embedContent")

		var req GeminiEmbeddingRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "Coop Pronto", req.Content.Parts[0].Text)

		require.NoError(t, json.NewEncoder(w).Encode(GeminiEmbeddingResponse{
			Embedding: GeminiEmbeddingValues{Values: []float32{0.1, 0.2, 0.3}},
		}))
	})

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "k", baseURL)

	vec, err := client.GetEmbedding(context.Background(), "Coop Pronto")

	require.NoError(t, err)
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, vec)
}

func TestGeminiClient_GetEmbeddingWithoutAPIKey(t *testing.T) {
	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "", "")

	_, err := client.GetEmbedding(context.Background(), "Coop")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key not set")
}

func TestGeminiClient_EmptyEmbeddingIsAnError(t *testing.T) {
	baseURL := withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(GeminiEmbeddingResponse{}))
	})

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "k", baseURL)

	_, err := client.GetEmbedding(context.Background(), "Coop")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty embedding")
}

// A 503 must be retried; the stub fails twice then succeeds.
func TestGeminiClient_RetriesServerErrors(t *testing.T) {
	calls := 0
	baseURL := withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
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

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "k", baseURL)

	tx, err := client.Categorize(context.Background(), models.Transaction{PartyName: "McDonalds"})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
	assert.Equal(t, "Restaurants", tx.Category)
}

// The security contract: neither the key nor the URL may appear in any log line.
func TestGeminiClient_NeverLogsCredentials(t *testing.T) {
	baseURL := withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	})

	logger := logging.NewMockLogger()
	client := NewGeminiClient(logger, 600, "gemini-2.0-flash", 30, "super-secret-key", baseURL)

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

// A transport-level failure — connection refused, timeout, cancelled context —
// returns a *url.Error whose message embeds the whole request URL. When the key
// travels as a query parameter, that error prints the credential in cleartext
// wherever it is logged, which is exactly what happened in production. The
// status-code tests above never caught it: they only exercise responses the
// server actually sent.
func TestGeminiClient_TransportErrorDoesNotLeakKey(t *testing.T) {
	// A server that is closed immediately: the next request to it is refused
	// at the TCP level, so the error comes from the transport, not the API.
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	baseURL := server.URL
	server.Close()

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "super-secret-key", baseURL)

	_, err := client.GetEmbedding(context.Background(), "Coop: groceries")

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "super-secret-key",
		"a transport error must not carry the API key")
	assert.NotContains(t, err.Error(), "http://",
		"a transport error must not carry the request URL, which may embed the key")
}

// The key belongs in the x-goog-api-key header, not the query string: a URL
// carrying it ends up in error messages, proxy logs, and crash reports. Google
// documents the header as the authentication method for these endpoints.
func TestGeminiClient_SendsKeyInHeaderNotURL(t *testing.T) {
	var gotHeader, gotQuery string
	baseURL := withGeminiServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("x-goog-api-key")
		gotQuery = r.URL.RawQuery
		require.NoError(t, json.NewEncoder(w).Encode(GeminiEmbeddingResponse{
			Embedding: struct {
				Values []float32 `json:"values"`
			}{Values: []float32{0.1, 0.2}},
		}))
	})

	client := NewGeminiClient(testLogger(), 600, "gemini-2.0-flash", 30, "super-secret-key", baseURL)

	_, err := client.GetEmbedding(context.Background(), "Coop: groceries")
	require.NoError(t, err)

	assert.Equal(t, "super-secret-key", gotHeader, "the key must be sent as a header")
	assert.NotContains(t, gotQuery, "super-secret-key", "the key must not appear in the URL")
}
