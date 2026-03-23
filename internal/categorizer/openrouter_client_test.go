package categorizer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenRouterClient_DefaultBaseURL(t *testing.T) {
	logger := logging.NewLogrusAdapter("debug", "text")
	client := NewOpenRouterClient(logger, 10, "", 30, "test-key", "")
	assert.Equal(t, "https://openrouter.ai/api/v1", client.baseURL)
}

func TestOpenRouterClient_DefaultModel(t *testing.T) {
	logger := logging.NewLogrusAdapter("debug", "text")
	client := NewOpenRouterClient(logger, 10, "", 30, "test-key", "")
	assert.Equal(t, "mistralai/mistral-small-2603", client.model)
}

func TestOpenRouterClient_CustomBaseURL(t *testing.T) {
	logger := logging.NewLogrusAdapter("debug", "text")
	client := NewOpenRouterClient(logger, 10, "", 30, "test-key", "https://custom.example.com/v1")
	assert.Equal(t, "https://custom.example.com/v1", client.baseURL)
}

func TestOpenRouterClient_GetEmbeddingAlwaysReturnsError(t *testing.T) {
	logger := logging.NewLogrusAdapter("debug", "text")
	client := NewOpenRouterClient(logger, 10, "mistralai/mistral-small-2603", 30, "test-key", "")

	result, err := client.GetEmbedding(context.Background(), "some text")
	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support embeddings")
}

func TestOpenRouterClient_CategorizeWithMockServer(t *testing.T) {
	// Create a mock HTTP server that returns a valid OpenRouter response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Decode and verify request body
		var req OpenRouterRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)
		assert.Equal(t, "mistralai/mistral-small-2603", req.Model)
		assert.Len(t, req.Messages, 1)
		assert.Equal(t, "user", req.Messages[0].Role)

		// Return mock response
		resp := OpenRouterResponse{
			Choices: []OpenRouterChoice{
				{
					Message: OpenRouterMessage{
						Role:    "assistant",
						Content: "Courses",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	logger := logging.NewLogrusAdapter("debug", "text")
	client := NewOpenRouterClient(logger, 60, "mistralai/mistral-small-2603", 30, "test-api-key", server.URL)

	transaction := models.Transaction{
		PartyName:   "Coop",
		Description: "Grocery shopping",
	}

	result, err := client.Categorize(context.Background(), transaction)
	require.NoError(t, err)
	assert.Equal(t, "Courses", result.Category)
}

func TestOpenRouterClient_CategorizeWithEmptyAPIKey(t *testing.T) {
	logger := logging.NewLogrusAdapter("debug", "text")
	client := NewOpenRouterClient(logger, 10, "mistralai/mistral-small-2603", 30, "", "")

	transaction := models.Transaction{
		PartyName:   "Coop",
		Description: "Grocery shopping",
	}

	result, err := client.Categorize(context.Background(), transaction)
	require.NoError(t, err)
	assert.Equal(t, models.CategoryUncategorized, result.Category)
}

func TestOpenRouterClient_ImplementsAIClientInterface(t *testing.T) {
	logger := logging.NewLogrusAdapter("debug", "text")
	var _ AIClient = NewOpenRouterClient(logger, 10, "", 30, "test-key", "")
}

func TestOpenRouterClient_CategorizeWithServerError(t *testing.T) {
	// Simulate a 500 server error (will cause retry attempts but eventually fail)
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`)) //nolint:errcheck
	}))
	defer server.Close()

	logger := logging.NewLogrusAdapter("error", "text") // Use error level to suppress retry logs
	// Use high rate limit and very short timeout to speed up test
	client := NewOpenRouterClient(logger, 600, "mistralai/mistral-small-2603", 5, "test-api-key", server.URL)

	transaction := models.Transaction{
		PartyName:   "Test",
		Description: "Test transaction",
	}

	result, err := client.Categorize(context.Background(), transaction)
	require.Error(t, err)
	assert.Equal(t, models.CategoryUncategorized, result.Category)
}
