package categorizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"github.com/sirupsen/logrus"
)

// GeminiClient implements the AIClient interface for interacting with the Google Gemini API.
type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	log        logging.Logger
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

// NewGeminiClient creates a new instance of GeminiClient.
func NewGeminiClient(logger logging.Logger) *GeminiClient {
	if logger == nil {
		logger = logging.NewLogrusAdapterFromLogger(logrus.New())
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		logger.Warn("GEMINI_API_KEY not set, AI categorization will fail")
	}

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.5-flash" // Default fallback
		logger.WithField("model", model).Debug("GEMINI_MODEL not set, using default")
	} else {
		logger.WithField("model", model).Debug("Using GEMINI_MODEL from environment")
	}

	return &GeminiClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: logger,
	}
}

// Categorize takes a context and a Transaction model, and returns the categorized Transaction
// or an error if categorization fails.
func (c *GeminiClient) Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
	if c.apiKey == "" {
		c.log.Debug("No API key available, skipping AI categorization")
		transaction.Category = models.CategoryUncategorized
		return transaction, nil
	}

	// Build the prompt for categorization
	prompt := c.buildCategorizationPrompt(transaction)

	c.log.WithFields(
		logging.Field{Key: "operation", Value: "gemini_categorization"},
		logging.Field{Key: "party_name", Value: transaction.PartyName},
		logging.Field{Key: "description", Value: transaction.Description},
	).Debug("Attempting to categorize transaction using Gemini API")

	// Make the API call
	category, err := c.callGeminiAPI(ctx, prompt)
	if err != nil {
		c.log.WithError(err).WithFields(
			logging.Field{Key: "party_name", Value: transaction.PartyName},
		).Warn("Failed to categorize transaction using Gemini API")
		transaction.Category = models.CategoryUncategorized
		return transaction, err
	}

	// Clean and validate the category
	category = c.cleanCategory(category)
	if category == "" || category == models.CategoryUncategorized {
		c.log.WithFields(
			logging.Field{Key: "party_name", Value: transaction.PartyName},
			logging.Field{Key: "raw_category", Value: category},
		).Debug("Gemini returned empty or uncategorized result")
		transaction.Category = models.CategoryUncategorized
	} else {
		transaction.Category = category
		c.log.WithFields(
			logging.Field{Key: "party_name", Value: transaction.PartyName},
			logging.Field{Key: "category", Value: category},
		).Info("Transaction successfully categorized by Gemini API")
	}

	return transaction, nil
}

// buildCategorizationPrompt creates a prompt for the Gemini API to categorize the transaction

func (c *GeminiClient) buildCategorizationPrompt(transaction models.Transaction) string {

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



// callGeminiAPI makes the actual API call to Gemini

func (c *GeminiClient) callGeminiAPI(ctx context.Context, prompt string) (string, error) {

	// Construct the API URL using the configured model

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)



	// Create the request payload

	request := GeminiRequest{

		Contents: []GeminiContent{

			{

				Parts: []GeminiPart{

					{Text: prompt},

				},

			},

		},

	}



	// Marshal to JSON

	jsonData, err := json.Marshal(request)

	if err != nil {

		return "", fmt.Errorf("failed to marshal request: %w", err)

	}



	// Create HTTP request

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))

	if err != nil {

		return "", fmt.Errorf("failed to create request: %w", err)

	}



	req.Header.Set("Content-Type", "application/json")



	// Make the request

	resp, err := c.httpClient.Do(req)

	if err != nil {

		return "", fmt.Errorf("failed to make API request: %w", err)

	}

	defer func() {

		if closeErr := resp.Body.Close(); closeErr != nil {

			c.log.WithError(closeErr).Warn("Failed to close response body")

		}

	}()



	// Read response

	body, err := io.ReadAll(resp.Body)

	if err != nil {

		return "", fmt.Errorf("failed to read response: %w", err)

	}



	// Check for HTTP errors

	if resp.StatusCode != http.StatusOK {

		c.log.WithFields(

			logging.Field{Key: "status_code", Value: resp.StatusCode},

			logging.Field{Key: "response_body", Value: string(body)},

		).Error("Gemini API returned error")

		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))

	}



	// Parse response

	var geminiResp GeminiResponse

	if err := json.Unmarshal(body, &geminiResp); err != nil {

		return "", fmt.Errorf("failed to unmarshal response: %w", err)

	}



	// Extract the category from response

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {

		return "", fmt.Errorf("no content in API response")

	}



	category := geminiResp.Candidates[0].Content.Parts[0].Text

	return strings.TrimSpace(category), nil

}



// cleanCategory cleans and validates the category returned by the API

func (c *GeminiClient) cleanCategory(category string) string {

	// Remove common prefixes/suffixes

	category = strings.TrimSpace(category)

	category = strings.TrimPrefix(category, "Category:")

	category = strings.TrimPrefix(category, "category:")

	category = strings.TrimSpace(category)



	// Remove quotes if present

	category = strings.Trim(category, `"'`)



	// Map of lower-case synonyms to canonical category names

	synonyms := map[string]string{

		"food":            "Alimentation", // Or Courses, context dependent, defaulting to generic

		"groceries":       "Courses",

		"supermarket":     "Courses",

		"restaurant":      "Restaurants",

		"transport":       "Transports Publics",

		"public transport": "Transports Publics",

		"train":           "Transports Publics",

		"bus":             "Transports Publics",

		"car":             "Voiture",

		"fuel":            "Voiture",

		"gas":             "Voiture",

		"parking":         "Voiture",

		"shopping":        "Shopping",

		"retail":          "Shopping",

		"clothes":         "Shopping",

		"clothing":        "Shopping",

		"electronics":     "Shopping", // Could be Equipement Maison too

		"health":          "Santé",

		"medical":         "Santé",

		"doctor":          "Santé",

		"pharmacy":        "Santé",

		"subscriptions":   "Abonnements",

		"subscription":    "Abonnements",

		"insurance":       "Assurances",

		"bank fees":       "Frais Bancaires",

		"fees":            "Frais Bancaires",

		"salary":          "Salaire",

		"income":          "Salaire",

		"rent":            "Logement",

		"housing":         "Logement",

		"utilities":       "Utilités",

		"phone":           "Utilités",

		"internet":        "Utilités",

		"electricity":     "Utilités",

		"entertainment":   "Divertissement",

		"movies":          "Divertissement",

		"leisure":         "Loisirs",

		"hobbies":         "Loisirs",

		"sports":          "Sport",

		"gym":             "Sport",

		"fitness":         "Sport",

		"travel":          "Vacances",

		"vacation":        "Vacances",

		"hotel":           "Vacances",

		"hotels":          "Vacances",

		"kids":            "Enfants",

		"children":        "Enfants",

		"education":       "Éducation",

		"school":          "Éducation",

		"gift":            "Cadeaux",

		"gifts":           "Cadeaux",

		"donation":        "Dons",

		"charity":         "Dons",

		"tax":             "Impôts",

		"taxes":           "Impôts",

		"investment":      "Investissements",

		"investments":     "Investissements",

		"furniture":       "Mobilier",

		"appliances":      "Équipement Maison",

		"withdrawal":      "Divers",

		"cash":            "Divers",

		"transfer":        "Virements",

		"transfers":       "Virements",

		"pension":         "Pension",

		"retirement":      "Pension",

		"mobilier & maison": "Mobilier", // Mapping old consolidated to new split (could be equiv too)

		"rentes & pensions": "Pension",

		"uncategorized":   models.CategoryUncategorized,

		"unknown":         models.CategoryUncategorized,

		"other":           models.CategoryUncategorized,

	}



	lowerCat := strings.ToLower(category)

	if canonical, ok := synonyms[lowerCat]; ok {

		return canonical

	}



	// If no synonym found, return the category as is (but trimmed)

	return category

}

// GetEmbedding returns the vector embedding for the given text using Gemini's embedding model
func (c *GeminiClient) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("API key not set")
	}

	// use text-embedding-004 for better performance/cost
	embeddingModel := "text-embedding-004"
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s", embeddingModel, c.apiKey)

	request := GeminiEmbeddingRequest{
		Content: GeminiContent{
			Parts: []GeminiPart{
				{Text: text},
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
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
		).Error("Gemini Embedding API returned error")
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
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
