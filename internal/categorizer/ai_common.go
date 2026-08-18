package categorizer

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"

	"golang.org/x/time/rate"
)

// completeFn sends a prompt to a provider and returns its raw text response.
// It is the only part of chat-based categorization that differs between
// providers: everything around it (credential gating, rate limiting, retry and
// backoff, prompt construction, response cleaning) is shared.
type completeFn func(ctx context.Context, prompt string) (string, error)

// baseAIClient holds the provider-independent half of an AIClient.
// GeminiClient and OpenRouterClient embed it and supply only their own
// completeFn, so the two clients cannot drift apart in behaviour again.
//
// SECURITY: apiKey MUST remain private and MUST NEVER be logged at any level.
// Provider URLs may embed the key, so URLs MUST NOT be logged or included in
// error messages either. Response bodies carry no credentials and may be logged.
type baseAIClient struct {
	apiKey   string // SECURITY: never log this field
	log      logging.Logger
	limiter  *rate.Limiter
	provider string // human-readable provider name, used only in log messages
}

// newBaseAIClient builds the shared client half, applying the defaults every
// provider agrees on.
func newBaseAIClient(provider string, logger logging.Logger, apiKey string, requestsPerMinute int) baseAIClient {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 10
	}

	return baseAIClient{
		apiKey:   apiKey,
		log:      logger,
		provider: provider,
		// requestsPerMinute / 60 = requests per second, bursting up to the
		// full per-minute allowance.
		limiter: rate.NewLimiter(rate.Limit(float64(requestsPerMinute)/60.0), requestsPerMinute),
	}
}

// categorize runs the full categorization sequence for one transaction:
// skip when no credential is configured, wait for a rate-limiter token, call
// the provider with retry and backoff, then clean the answer.
//
// A missing API key is not an error: the transaction comes back uncategorized
// so the caller can fall back to the earlier strategy tiers.
func (b *baseAIClient) categorize(ctx context.Context, transaction models.Transaction, complete completeFn) (models.Transaction, error) {
	if b.apiKey == "" {
		b.log.Debug("No API key available, skipping AI categorization")
		transaction.Category = models.CategoryUncategorized
		return transaction, nil
	}

	prompt := buildCategorizationPrompt(transaction)

	b.log.WithFields(
		logging.Field{Key: "operation", Value: b.provider + "_categorization"},
		logging.Field{Key: "party_name", Value: transaction.PartyName},
		logging.Field{Key: "description", Value: transaction.Description},
	).Debug("Attempting to categorize transaction using " + b.provider + " API")

	// Blocks until a token is available, honouring ctx cancellation.
	if err := b.limiter.Wait(ctx); err != nil {
		b.log.WithError(err).Warn("Rate limiter wait cancelled")
		return transaction, fmt.Errorf("rate limiter wait cancelled: %w", err)
	}

	category, err := b.completeWithRetry(ctx, prompt, complete)
	if err != nil {
		b.log.WithError(err).WithFields(
			logging.Field{Key: "party_name", Value: transaction.PartyName},
		).Warn("Failed to categorize transaction using " + b.provider + " API")
		transaction.Category = models.CategoryUncategorized
		return transaction, err
	}

	category = cleanCategory(category)
	if category == "" || category == models.CategoryUncategorized {
		b.log.WithFields(
			logging.Field{Key: "party_name", Value: transaction.PartyName},
			logging.Field{Key: "raw_category", Value: category},
		).Debug(b.provider + " returned empty or uncategorized result")
		transaction.Category = models.CategoryUncategorized
		return transaction, nil
	}

	transaction.Category = category
	b.log.WithFields(
		logging.Field{Key: "party_name", Value: transaction.PartyName},
		logging.Field{Key: "category", Value: category},
	).Info("Transaction successfully categorized by " + b.provider + " API")

	return transaction, nil
}

// completeWithRetry calls complete, retrying transient failures with
// exponential backoff and jitter. Non-retryable errors return immediately.
func (b *baseAIClient) completeWithRetry(ctx context.Context, prompt string, complete completeFn) (string, error) {
	const (
		maxRetries        = 3
		baseDelay         = 1 * time.Second
		backoffMultiplier = 2.0
		jitterFraction    = 0.2 // ±20% jitter
	)

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		category, err := complete(ctx, prompt)
		if err == nil {
			return category, nil
		}
		lastErr = err

		if !isRetryableError(err) {
			b.log.WithError(err).Warn("Non-retryable error from " + b.provider + " API")
			return "", err
		}

		if attempt == maxRetries {
			b.log.WithError(err).WithField("attempts", attempt+1).Warn("All retry attempts exhausted")
			return "", fmt.Errorf("API request failed after %d attempts: %w", maxRetries+1, err)
		}

		delayMs := int64(math.Pow(backoffMultiplier, float64(attempt)) * float64(baseDelay.Milliseconds()))
		// Nanosecond timestamp drives the jitter sign. This paces retries; it is
		// not security-sensitive, so math/rand is neither needed nor wanted here.
		jitterSign := float64((time.Now().UnixNano()%2)*2 - 1) // -1 or +1
		jitterMs := int64(float64(delayMs) * jitterFraction * jitterSign)
		totalDelay := time.Duration(delayMs+jitterMs) * time.Millisecond

		b.log.WithFields(
			logging.Field{Key: "attempt", Value: attempt + 1},
			logging.Field{Key: "max_attempts", Value: maxRetries + 1},
			logging.Field{Key: "retry_delay_ms", Value: totalDelay.Milliseconds()},
			logging.Field{Key: "error", Value: err.Error()},
		).Info("Retrying API request due to transient error")

		select {
		case <-time.After(totalDelay):
		case <-ctx.Done():
			b.log.WithError(ctx.Err()).Warn("Context cancelled during retry wait")
			return "", fmt.Errorf("context cancelled: %w", ctx.Err())
		}
	}

	return "", lastErr
}

// isRetryableError reports whether err is a transient failure worth retrying:
// a timeout, a server-side 5xx, a 429, or a connection-level network error.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	if os.IsTimeout(err) {
		return true
	}

	errStr := err.Error()

	// Status codes surfaced by the providers' error strings.
	for _, status := range []string{"status 429", "status 500", "status 503"} {
		if strings.Contains(errStr, status) {
			return true
		}
	}

	for _, netErr := range []string{"connection refused", "connection reset", "temporary failure"} {
		if strings.Contains(errStr, netErr) {
			return true
		}
	}

	return false
}

// categorySynonyms maps lower-case answers a model may produce to the canonical
// category names used throughout the application.
var categorySynonyms = map[string]string{
	"food":              "Alimentation", // specialised food shops, not supermarkets
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
	"mobilier & maison": "Mobilier", // old consolidated name, now split
	"rentes & pensions": "Pension",
	"uncategorized":     models.CategoryUncategorized,
	"unknown":           models.CategoryUncategorized,
	"other":             models.CategoryUncategorized,
}

// cleanCategory reduces a model's answer to a bare category name.
//
// Models do not reliably answer with just the category: some prepend an
// explanation and put the answer on the last line, some wrap it in markdown
// bold, some add a "Category:" label or quotes, and some answer in English
// where the application's categories are French. Each step below strips one of
// those observed shapes; the synonym table handles the last case.
func cleanCategory(category string) string {
	category = strings.TrimSpace(category)

	// Verbose multi-line answers put the category on the last non-empty line.
	if strings.Contains(category, "\n") {
		lines := strings.Split(category, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			if line := strings.TrimSpace(lines[i]); line != "" {
				category = line
				break
			}
		}
	}

	// Pull **text** out of anywhere in a single-line answer; otherwise just
	// strip any stray asterisks at the edges.
	if _, after, found := strings.Cut(category, "**"); found {
		if inner, _, ok := strings.Cut(after, "**"); ok {
			category = inner
		}
	} else {
		category = strings.Trim(category, "*")
	}

	category = strings.TrimSpace(category)
	category = strings.TrimPrefix(category, "Category:")
	category = strings.TrimPrefix(category, "category:")
	category = strings.TrimSpace(category)
	category = strings.Trim(category, `"'`)

	// Some models echo the parenthetical hint from the prompt's category list
	// (e.g. "Alimentation (boucherie, boulangerie, traiteur - NOT supermarkets)")
	// instead of just the bare name.
	if idx := strings.Index(category, " ("); idx != -1 && strings.HasSuffix(category, ")") {
		category = strings.TrimSpace(category[:idx])
	}

	// Some providers redact perceived PII in the answer and return only the
	// redaction placeholder (e.g. "[ADDRESS]"). No real category is ever
	// bracket-wrapped, so treat this as no answer rather than a category.
	if strings.HasPrefix(category, "[") && strings.HasSuffix(category, "]") {
		return models.CategoryUncategorized
	}

	if canonical, ok := categorySynonyms[strings.ToLower(category)]; ok {
		return canonical
	}

	return category
}

// buildCategorizationPrompt renders the categorization prompt for a transaction.
// Both providers use the identical prompt so their answers stay comparable.
func buildCategorizationPrompt(transaction models.Transaction) string {
	return fmt.Sprintf(`You are a financial transaction categorizer for a personal finance application.

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
}
