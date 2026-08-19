package categorizer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
)

// GeminiClient implements the AIClient interface for Google's Gemini API.
//
// Everything except the HTTP calls lives in baseAIClient; see the security
// notes there. One extra rule applies here: Gemini takes its credential as a
// URL query parameter, so Gemini URLs contain the API key and MUST NOT be
// logged or embedded in error messages.
type GeminiClient struct {
	baseAIClient
	model      string
	baseURL    string
	httpClient *http.Client
}

// GeminiRequest represents the request structure for Gemini API
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiResponse represents the response structure from Gemini API
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

// GeminiEmbeddingRequest represents the request structure for Gemini Embedding API
type GeminiEmbeddingRequest struct {
	Content GeminiContent `json:"content"`
}

// GeminiEmbeddingResponse represents the response structure from Gemini Embedding API
type GeminiEmbeddingResponse struct {
	Embedding GeminiEmbeddingValues `json:"embedding"`
}

type GeminiEmbeddingValues struct {
	Values []float32 `json:"values"`
}

// defaultGeminiAPIBaseURL is the Gemini generative-language endpoint root.
// Callers may override it through NewGeminiClient, which is how tests point the
// client at a stub server without any shared mutable state.
const defaultGeminiAPIBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"

// geminiEmbeddingModel is the embedding model used by GetEmbedding.
// text-embedding-004 was deprecated in November 2025.
const geminiEmbeddingModel = "gemini-embedding-001"

// NewGeminiClient creates a new instance of GeminiClient.
// model, timeoutSeconds and baseURL are wired from config; empty/zero values use
// sensible defaults. apiKey is injected directly by the container (no os.Getenv
// inside this constructor).
func NewGeminiClient(logger logging.Logger, requestsPerMinute int, model string, timeoutSeconds int, apiKey string, baseURL string) *GeminiClient {
	if logger == nil {
		logger = logging.NewLogrusAdapterFromLogger(logrus.New())
	}

	if apiKey == "" {
		logger.Warn("API key not set, AI categorization will fail")
	}

	if model == "" {
		model = "gemini-2.0-flash" // Match config default in viper.go
	}
	logger.WithField("model", model).Debug("Using Gemini model")

	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	if baseURL == "" {
		baseURL = defaultGeminiAPIBaseURL
	}

	return &GeminiClient{
		baseAIClient: newBaseAIClient("gemini", logger, apiKey, requestsPerMinute),
		model:        model,
		baseURL:      baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

// Categorize takes a context and a Transaction model, and returns the categorized Transaction
// or an error if categorization fails.
func (c *GeminiClient) Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
	return c.categorize(ctx, transaction, c.complete)
}

// complete posts the prompt to Gemini's generateContent endpoint and returns
// the model's raw reply.
func (c *GeminiClient) complete(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("%s/%s:generateContent", c.baseURL, c.model)

	request := GeminiRequest{
		Contents: []GeminiContent{
			{Parts: []GeminiPart{{Text: prompt}}},
		},
	}

	body, err := c.postJSON(ctx, url, request, "Gemini API")
	if err != nil {
		return "", err
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in API response")
	}

	return strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text), nil
}

// GetEmbedding returns the vector embedding for the given text using Gemini's embedding model.
func (c *GeminiClient) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("API key not set")
	}

	url := fmt.Sprintf("%s/%s:embedContent", c.baseURL, geminiEmbeddingModel)

	request := GeminiEmbeddingRequest{
		Content: GeminiContent{Parts: []GeminiPart{{Text: text}}},
	}

	body, err := c.postJSON(ctx, url, request, "Gemini Embedding API")
	if err != nil {
		return nil, err
	}

	var geminiResp GeminiEmbeddingResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(geminiResp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return geminiResp.Embedding.Values, nil
}

// postJSON marshals payload, POSTs it to url, and returns the response body.
// The API key travels in the x-goog-api-key header, the authentication method
// Google documents for these endpoints. It is deliberately not a query
// parameter: net/http reports transport failures as a *url.Error whose message
// embeds the whole request URL, so a key in the query string is printed in
// cleartext by every timeout, refused connection, or cancelled request.
// apiLabel names the endpoint in log messages.
func (c *GeminiClient) postJSON(ctx context.Context, url string, payload any, apiLabel string) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req) // #nosec G107 -- URL is built from config, not user input
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", withoutURL(err))
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.log.WithError(closeErr).Warn("Failed to close response body")
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.log.WithFields(
			logging.Field{Key: "status_code", Value: resp.StatusCode},
			logging.Field{Key: "response_body", Value: string(body)},
		).Error(apiLabel + " returned error")
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// withoutURL strips the request URL from a transport error.
//
// net/http wraps every transport failure in a *url.Error, whose Error() prints
// the full URL. Keeping the key out of the query string (see postJSON) is the
// first line of defence; this is the second, so that a URL cannot reach a log
// through a redirect, a proxy setting, or a future endpoint that does take a
// credential in its query. The operation and the underlying cause — the parts
// that make the error diagnosable — are preserved.
func withoutURL(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s request failed: %w", urlErr.Op, urlErr.Err)
	}
	return err
}
