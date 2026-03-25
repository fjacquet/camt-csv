package categorizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// OpenRouterClient implements AIClient for OpenAI-compatible APIs (OpenRouter, etc.)
//
// SECURITY: This client handles sensitive API credentials. The following policies MUST be maintained:
//   - apiKey field MUST remain private and NEVER be logged at any log level
//   - API URLs MUST NOT be logged (URLs may be built but never logged)
//   - Error messages MUST NOT include URLs or credentials
//   - Only response bodies (which don't contain credentials) may be logged for debugging
type OpenRouterClient struct {
	apiKey         string // SECURITY: Never log this field
	model          string
	baseURL        string
	httpClient     *http.Client
	log            logging.Logger
	limiter        *rate.Limiter

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

	if requestsPerMinute <= 0 {
		requestsPerMinute = 10
	}

	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	// Create rate limiter: requestsPerMinute / 60 = requests per second
	limiter := rate.NewLimiter(
		rate.Limit(float64(requestsPerMinute)/60.0),
		requestsPerMinute, // Allow bursts up to the per-minute limit
	)

	logger.WithField("model", model).Debug("Using OpenRouter model")

	return &OpenRouterClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
		log:     logger,
		limiter: limiter,
	}
}

// Categorize takes a context and a Transaction model, and returns the categorized Transaction
// or an error if categorization fails.
func (c *OpenRouterClient) Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
	if c.apiKey == "" {
		c.log.Debug("No API key available, skipping AI categorization")
		transaction.Category = models.CategoryUncategorized
		return transaction, nil
	}

	// Build the prompt for categorization
	prompt := c.buildCategorizationPrompt(transaction)

	c.log.WithFields(
		logging.Field{Key: "operation", Value: "openrouter_categorization"},
		logging.Field{Key: "party_name", Value: transaction.PartyName},
		logging.Field{Key: "description", Value: transaction.Description},
	).Debug("Attempting to categorize transaction using OpenRouter API")

	// Wait for rate limiter token (blocks until available, respecting ctx cancellation)
	if err := c.limiter.Wait(ctx); err != nil {
		c.log.WithError(err).Warn("Rate limiter wait cancelled")
		return transaction, fmt.Errorf("rate limiter wait cancelled: %w", err)
	}

	// Make the API call with retry logic
	category, err := c.callAPIWithRetry(ctx, prompt)
	if err != nil {
		c.log.WithError(err).WithFields(
			logging.Field{Key: "party_name", Value: transaction.PartyName},
		).Warn("Failed to categorize transaction using OpenRouter API")
		transaction.Category = models.CategoryUncategorized
		return transaction, err
	}

	// Clean and validate the category
	category = c.cleanCategory(category)
	if category == "" || category == models.CategoryUncategorized {
		c.log.WithFields(
			logging.Field{Key: "party_name", Value: transaction.PartyName},
			logging.Field{Key: "raw_category", Value: category},
		).Debug("OpenRouter returned empty or uncategorized result")
		transaction.Category = models.CategoryUncategorized
	} else {
		transaction.Category = category
		c.log.WithFields(
			logging.Field{Key: "party_name", Value: transaction.PartyName},
			logging.Field{Key: "category", Value: category},
		).Info("Transaction successfully categorized by OpenRouter API")
	}

	return transaction, nil
}

// GetEmbedding returns an error since OpenRouter does not support embeddings.
// Use a dedicated embedding provider (e.g., Gemini) for semantic search.
func (c *OpenRouterClient) GetEmbedding(_ context.Context, _ string) ([]float32, error) {
	return nil, fmt.Errorf("OpenRouter does not support embeddings; use a dedicated embedding provider")
}

// isRetryableError checks if an error is worth retrying
func (c *OpenRouterClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	if strings.Contains(errStr, "status 429") || // Too Many Requests
		strings.Contains(errStr, "status 503") || // Service Unavailable
		strings.Contains(errStr, "status 500") { // Internal Server Error (sometimes retryable)
		return true
	}

	// Network errors are retryable
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "temporary failure") {
		return true
	}

	return false
}

// callAPIWithRetry wraps callAPI with retry-backoff logic matching GeminiClient pattern
func (c *OpenRouterClient) callAPIWithRetry(ctx context.Context, prompt string) (string, error) {
	const (
		maxRetries        = 3
		baseDelay         = 1 * time.Second
		backoffMultiplier = 2.0
		jitterFraction    = 0.2 // ±20% jitter
	)

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		category, err := c.callAPI(ctx, prompt)

		if err == nil {
			return category, nil
		}

		lastErr = err

		// If error is not retryable, return immediately
		if !c.isRetryableError(err) {
			c.log.WithError(err).Warn("Non-retryable error from OpenRouter API")
			return "", err
		}

		// If this was the last retry, return error
		if attempt == maxRetries {
			c.log.WithError(err).WithField("attempts", attempt+1).Warn("All retry attempts exhausted")
			return "", fmt.Errorf("API request failed after %d attempts: %w", maxRetries+1, err)
		}

		// Calculate backoff delay with time-based jitter (not security-sensitive)
		delayMs := int64(math.Pow(backoffMultiplier, float64(attempt)) * float64(baseDelay.Milliseconds()))
		// Use nanosecond timestamp for jitter — varies per retry, not security-sensitive
		jitterSign := float64((time.Now().UnixNano()%2)*2 - 1) // -1 or +1
		jitterMs := int64(float64(delayMs) * jitterFraction * jitterSign)
		totalDelay := time.Duration(delayMs+jitterMs) * time.Millisecond

		c.log.WithFields(
			logging.Field{Key: "attempt", Value: attempt + 1},
			logging.Field{Key: "max_attempts", Value: maxRetries + 1},
			logging.Field{Key: "retry_delay_ms", Value: totalDelay.Milliseconds()},
			logging.Field{Key: "error", Value: err.Error()},
		).Info("Retrying API request due to transient error")

		// Wait before retry (or until context cancelled)
		select {
		case <-time.After(totalDelay):
			// Continue to next attempt
		case <-ctx.Done():
			c.log.WithError(ctx.Err()).Warn("Context cancelled during retry wait")
			return "", fmt.Errorf("context cancelled: %w", ctx.Err())
		}
	}

	return "", lastErr
}

// callAPI makes the actual API call to OpenRouter
func (c *OpenRouterClient) callAPI(ctx context.Context, prompt string) (string, error) {
	// SECURITY: URL does not contain credentials (key is in Authorization header, not URL)
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

	// SECURITY: API key is in Authorization header, never in URL
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

	content := openRouterResp.Choices[0].Message.Content
	return strings.TrimSpace(content), nil
}

// buildCategorizationPrompt creates a prompt for the OpenRouter API to categorize the transaction.
// Uses the same prompt text as GeminiClient for consistency.
func (c *OpenRouterClient) buildCategorizationPrompt(transaction models.Transaction) string {
	prompt := fmt.Sprintf(`You are a financial transaction categorizer for a personal finance application.

Your goal is to categorize the given transaction into ONE of the specific categories listed below.



CATEGORIES (Strictly limit your answer to this list):

- Abonnements

- Activités

- Alimentation (boucherie, boulangerie, traiteur - NOT supermarkets)

- Allocations

- Animaux

- Assurance Maladie

- Assurances

- Autre

- Bien-être (spa, massage)

- Cadeaux

- Courses (supermarkets like Migros, Coop, Aldi, Lidl)

- Divers (cash withdrawals, pocket money)

- Divertissement (movies, games)

- Dons

- Éducation

- Enfants

- Épargne

- Équipement Maison (appliances, electronics for home)

- Famille

- Formation

- Frais Bancaires

- Hypothèques

- Impôts

- Investissements

- Logement (rent, charges)

- Loisirs (parks, museums, concerts)

- Mobilier (furniture, decoration, IKEA)

- Non Classé

- Pension (retirement, AVS/AI)

- Prêts

- Restaurants (dining out, fast food, cafes)

- Revenus Financiers

- Revenus Locatifs

- Revenus Professionnels

- Salaire

- Santé (doctors, pharmacy)

- Séjours (short stays, weekends)

- Services

- Shopping (clothes, electronics, online)

- Soins Personnels (hairdresser, cosmetics)

- Sport

- Taxes

- Transferts

- Transport Privé

- Transports Publics

- Utilités (electricity, phone, internet)

- Vacances (travel, flights, hotels)

- Virements

- Voiture (fuel, parking, repairs)

- Voyages (travel agency, cruises)



TRICKY CASES / RULES:

1. **Supermarkets**: "Migros", "Coop", "Denner", "Aldi" are **Courses**. They are NOT "Alimentation" (reserved for specialized food shops) or "Restaurants".

2. **Restaurants**: "McDonalds", "Starbucks", "Restaurant X" are **Restaurants**.

3. **AI & Tech**: "Claude.ai", "OpenAI", "ChatGPT", "Google One" are **Abonnements**.

4. **Transport**: "SNCF", "CFF", "SBB" are **Transports Publics**. "Shell", "BP", "Parking" are **Voiture**.

5. **Vacation**: "EasyJet", "Airbnb", "Booking.com" are **Vacances**.

6. **Furniture vs Appliances**: "IKEA", "Conforama" are **Mobilier**. "Dyson", "Fust" are **Équipement Maison**.

7. **Retirement**: "Pension" is ONLY for retirement funds.



FEW-SHOT EXAMPLES:

- Transaction: "OpenAI *ChatGPT", Amount: 20.00 -> Category: Abonnements

- Transaction: "Coop Pronto", Amount: 15.50 -> Category: Courses

- Transaction: "McDonalds", Amount: 24.90 -> Category: Restaurants

- Transaction: "SBB CFF FFS Mobile Ticket", Amount: 5.60 -> Category: Transports Publics

- Transaction: "Parking de la Gare", Amount: 3.00 -> Category: Voiture

- Transaction: "IKEA AG", Amount: 150.00 -> Category: Mobilier

- Transaction: "Zalando", Amount: 89.90 -> Category: Shopping

- Transaction: "Retrait Bancomat", Amount: 100.00 -> Category: Divers

- Transaction: "La Vaudoise Assurances", Amount: 450.00 -> Category: Assurances

- Transaction: "EasyJet", Amount: 120.00 -> Category: Vacances



TRANSACTION TO CATEGORIZE:

Party: %s

Description: %s

Amount: %s CHF



Category:`, transaction.PartyName, transaction.Description, transaction.Amount.String())

	return prompt
}

// cleanCategory cleans and validates the category returned by the API.
// Mirrors the GeminiClient implementation for consistency.
func (c *OpenRouterClient) cleanCategory(category string) string {
	category = strings.TrimSpace(category)

	// If multi-line verbose response, extract the last non-empty line
	if strings.Contains(category, "\n") {
		lines := strings.Split(category, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line != "" {
				category = line
				break
			}
		}
	}

	// Extract **text** from anywhere in the string (single-line verbose responses)
	if _, after, found := strings.Cut(category, "**"); found {
		if inner, _, ok := strings.Cut(after, "**"); ok {
			category = inner
		}
	} else {
		// Strip markdown bold formatting at edges (**Category**)
		category = strings.Trim(category, "*")
	}

	// Remove common prefixes/suffixes
	category = strings.TrimSpace(category)
	category = strings.TrimPrefix(category, "Category:")
	category = strings.TrimPrefix(category, "category:")
	category = strings.TrimSpace(category)

	// Remove quotes if present
	category = strings.Trim(category, `"'`)

	// Map of lower-case synonyms to canonical category names
	synonyms := map[string]string{
		"food":              "Alimentation",
		"groceries":         "Courses",
		"supermarket":       "Courses",
		"restaurant":        "Restaurants",
		"transport":         "Transports Publics",
		"public transport":  "Transports Publics",
		"train":             "Transports Publics",
		"bus":               "Transports Publics",
		"car":               "Voiture",
		"fuel":              "Voiture",
		"gas":               "Voiture",
		"parking":           "Voiture",
		"shopping":          "Shopping",
		"retail":            "Shopping",
		"clothes":           "Shopping",
		"clothing":          "Shopping",
		"electronics":       "Shopping",
		"health":            "Santé",
		"medical":           "Santé",
		"doctor":            "Santé",
		"pharmacy":          "Santé",
		"subscriptions":     "Abonnements",
		"subscription":      "Abonnements",
		"insurance":         "Assurances",
		"bank fees":         "Frais Bancaires",
		"fees":              "Frais Bancaires",
		"salary":            "Salaire",
		"income":            "Salaire",
		"rent":              "Logement",
		"housing":           "Logement",
		"utilities":         "Utilités",
		"phone":             "Utilités",
		"internet":          "Utilités",
		"electricity":       "Utilités",
		"entertainment":     "Divertissement",
		"movies":            "Divertissement",
		"leisure":           "Loisirs",
		"hobbies":           "Loisirs",
		"sports":            "Sport",
		"gym":               "Sport",
		"fitness":           "Sport",
		"travel":            "Vacances",
		"vacation":          "Vacances",
		"hotel":             "Vacances",
		"hotels":            "Vacances",
		"kids":              "Enfants",
		"children":          "Enfants",
		"education":         "Éducation",
		"school":            "Éducation",
		"gift":              "Cadeaux",
		"gifts":             "Cadeaux",
		"donation":          "Dons",
		"charity":           "Dons",
		"tax":               "Impôts",
		"taxes":             "Impôts",
		"investment":        "Investissements",
		"investments":       "Investissements",
		"furniture":         "Mobilier",
		"appliances":        "Équipement Maison",
		"withdrawal":        "Divers",
		"cash":              "Divers",
		"transfer":          "Virements",
		"transfers":         "Virements",
		"pension":           "Pension",
		"retirement":        "Pension",
		"mobilier & maison": "Mobilier",
		"rentes & pensions": "Pension",
		"uncategorized":     models.CategoryUncategorized,
		"unknown":           models.CategoryUncategorized,
		"other":             models.CategoryUncategorized,
	}

	lowerCat := strings.ToLower(category)
	if canonical, ok := synonyms[lowerCat]; ok {
		return canonical
	}

	// If no synonym found, return the category as is (but trimmed)
	return category
}
