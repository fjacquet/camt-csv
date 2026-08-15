package categorizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
)

// OpenRouterClient implements AIClient for OpenAI-compatible chat APIs
// (OpenRouter and anything speaking the same /chat/completions contract).
//
// Everything except the HTTP call itself lives in baseAIClient; see the
// security notes there, which apply to this client too.
type OpenRouterClient struct {
	baseAIClient
	model      string
	baseURL    string
	httpClient *http.Client
}

// OpenRouterRequest represents the request structure for OpenRouter (OpenAI-compatible) API
type OpenRouterRequest struct {
	Model    string              `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
}

// OpenRouterMessage represents a single chat message
type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse represents the response structure from OpenRouter API
type OpenRouterResponse struct {
	Choices []OpenRouterChoice `json:"choices"`
}

// OpenRouterChoice represents a single choice in the OpenRouter response
type OpenRouterChoice struct {
	Message OpenRouterMessage `json:"message"`
}

// NewOpenRouterClient creates a new instance of OpenRouterClient.
// apiKey is passed directly (not read from env) to allow flexible key management.
// baseURL defaults to "https://openrouter.ai/api/v1" when empty string passed.
// model defaults to "mistralai/mistral-small-2603" when empty string passed.
func NewOpenRouterClient(logger logging.Logger, requestsPerMinute int, model string, timeoutSeconds int, apiKey string, baseURL string) *OpenRouterClient {
	if logger == nil {
		logger = logging.NewLogrusAdapterFromLogger(logrus.New())
	}

	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	if model == "" {
		model = "mistralai/mistral-small-2603"
	}

	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	logger.WithField("model", model).Debug("Using OpenRouter model")

	return &OpenRouterClient{
		baseAIClient: newBaseAIClient("openrouter", logger, apiKey, requestsPerMinute),
		model:        model,
		baseURL:      baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

// Categorize takes a context and a Transaction model, and returns the categorized Transaction
// or an error if categorization fails.
func (c *OpenRouterClient) Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
	return c.categorize(ctx, transaction, c.complete)
}

// GetEmbedding returns an error since OpenRouter does not support embeddings.
// Use a dedicated embedding provider (e.g., Gemini) for semantic search.
func (c *OpenRouterClient) GetEmbedding(_ context.Context, _ string) ([]float32, error) {
	return nil, fmt.Errorf("OpenRouter does not support embeddings; use a dedicated embedding provider")
}

// complete posts the prompt to the provider's chat-completions endpoint and
// returns the assistant's raw reply.
func (c *OpenRouterClient) complete(ctx context.Context, prompt string) (string, error) {
	// SECURITY: URL carries no credentials; the key travels in the Authorization header.
	url := c.baseURL + "/chat/completions"

	request := OpenRouterRequest{
		Model: c.model,
		Messages: []OpenRouterMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// SECURITY: API key is in the Authorization header, never in the URL.
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req) // #nosec G107 -- URL is built from config, not user input
	if err != nil {
		return "", fmt.Errorf("failed to make API request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.log.WithError(closeErr).Warn("Failed to close response body")
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.log.WithFields(
			logging.Field{Key: "status_code", Value: resp.StatusCode},
			logging.Field{Key: "response_body", Value: string(body)},
		).Error("OpenRouter API returned error")
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var openRouterResp OpenRouterResponse
	if err := json.Unmarshal(body, &openRouterResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in API response")
	}

	return strings.TrimSpace(openRouterResp.Choices[0].Message.Content), nil
}
