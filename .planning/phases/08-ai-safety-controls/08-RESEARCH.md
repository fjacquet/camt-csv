# Phase 8: AI Safety Controls - Research

**Researched:** 2026-02-16
**Domain:** AI safety gates, rate limiting, retry logic, confidence scoring
**Confidence:** HIGH

## Summary

Phase 8 implements three critical safety mechanisms to prevent silent AI miscategorization and API quota exhaustion. The codebase already has:

1. **Auto-learn infrastructure** (partially implemented) — categorizations auto-save via `Categorizer.CategorizeTransaction()`, but lacks a control flag
2. **Configuration infrastructure** — `categorization.auto_learn` exists in config with default `true`, but is not wired to any logic
3. **HTTP client foundation** — GeminiClient has 30s timeout, but no rate limiting or retry logic
4. **Confidence scoring gap** — Gemini API responses not parsed for confidence; categorizations lack confidence metadata

**Primary recommendation:** Wire `--auto-learn` flag to gate categorizer's auto-save behavior; add rate limiter and retry-with-backoff to GeminiClient; extend Category struct to carry confidence scores and log them before persistence.

---

## User Constraints (from Phase Context)

### Locked Decisions
- **D-11 (v1.2):** AI auto-learn defaults to OFF — `--auto-learn` flag controls whether AI categorizations are saved to YAML; defaults to disabled
- **Current decision:** Phase 7 creates the AI categorization pipeline; Phase 8 gates it safely

### Claude's Discretion
- How to implement rate limiting (token bucket, semaphore, sliding window?)
- How to extract/estimate confidence from Gemini responses (currently returns category string only)
- Retry backoff strategy (linear vs. exponential, max retries, jitter?)
- Logging format for confidence scores before persistence

### Deferred Ideas (OUT OF SCOPE)
- User-facing confidence UI in iCompta (Phase 8 logs confidence; presentation is future work)
- Dynamic rate adjustment based on quota remaining (collect metrics first; adapt later)
- Per-category confidence thresholds (use global threshold from config)

---

## Current Implementation State

### ✅ What Already Exists

#### Configuration Layer (HIGH confidence)
- `internal/config/viper.go`:
  - `Categorization.AutoLearn bool` — field exists, defaults to `true`
  - `Categorization.ConfidenceThreshold float64` — field exists, defaults to `0.8`
  - `AI.RequestsPerMinute int` — field exists, defaults to `10`
  - `AI.TimeoutSeconds int` — field exists, defaults to `30`
- Environment variable binding: `CAMT_CATEGORIZATION_AUTO_LEARN`, `CAMT_AI_REQUESTS_PER_MINUTE`
- Viper config file support: `~/.camt-csv/camt-csv.yaml` or `.camt-csv/config.yaml`

#### Auto-Learning (MEDIUM confidence)
- `internal/categorizer/categorizer.go`:
  - `Categorize()` method unconditionally saves mappings to YAML (lines ~180-210)
  - `CategorizeTransaction()` method calls internal `categorizeTransaction()` then auto-saves (lines ~133-175)
  - `updateDebitorCategory()` / `updateCreditorCategory()` → save via `SaveDebitorsToYAML()` / `SaveCreditorsToYAML()`
  - **Problem:** No control flag — auto-learning always happens if categorization succeeds

#### HTTP & Timeout (HIGH confidence)
- `internal/categorizer/gemini_client.go`:
  - `httpClient: &http.Client{Timeout: 30 * time.Second}` (line ~72)
  - Basic error handling in `callGeminiAPI()` (line ~108+) for HTTP status codes
  - **Problem:** No rate limiting; no retry logic; no backoff on 429/503

#### Confidence Metadata (MEDIUM confidence)
- `internal/models/categorizer.go`:
  - `Category struct` has Name, Description only — **no Confidence field**
  - **Problem:** Gemini response is parsed as plain string category name; confidence is lost

#### CLI Wiring (LOW confidence)
- `cmd/root/root.go`:
  - `--ai-enabled` flag exists (line ~66)
  - **Missing:** `--auto-learn` flag not defined as CLI flag (only config-based)
- `cmd/categorize/categorize.go`:
  - Uses `container.GetCategorizer()` to call `CategorizeTransaction()`
  - **Missing:** No control over auto-learning at command level

---

## Architecture Patterns

### Pattern 1: Configuration Override Hierarchy (EXISTING)
**What:** Viper-based config with 4-level override: defaults → config file → environment → CLI flags
**When to use:** All new configuration (e.g., `--auto-learn` flag)
**Example:**
```go
// internal/config/viper.go
v.SetDefault("categorization.auto_learn", false)  // Default: OFF (per D-11)

// cmd/root/root.go
Cmd.PersistentFlags().Bool("auto-learn", false, "Enable AI auto-learning")
viper.BindPFlag("categorization.auto_learn", Cmd.PersistentFlags().Lookup("auto-learn"))

// At runtime:
isAutoLearn := AppConfig.Categorization.AutoLearn  // Respects all 4 levels
```

### Pattern 2: Dependency Injection via Container (EXISTING)
**What:** All components receive dependencies through constructors; global container initialized in `root.PersistentPreRun`
**When to use:** Pass config/flags to categorizer
**Example:**
```go
// Categorizer needs to know whether to auto-learn:
cat := categorizer.NewCategorizer(aiClient, store, logger, config.Categorization.AutoLearn)
```

### Pattern 3: Rate Limiting with Token Bucket (RECOMMENDED)
**What:** Track requests per minute; use time.Ticker to replenish allowance
**When to use:** Prevent quota exhaustion on Gemini API
**Go standard library:** `golang.org/x/time/rate` (Token Bucket Limiter)
**Example:**
```go
import "golang.org/x/time/rate"

type GeminiClient struct {
    limiter *rate.Limiter  // rate.NewLimiter(rate.Limit(float64(requestsPerMinute)/60), 1)
}

func (c *GeminiClient) Categorize(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
    if !c.limiter.Allow() {
        return tx, fmt.Errorf("rate limit exceeded")
    }
    // Proceed with API call
}
```
**Source:** Go stdlib `golang.org/x/time/rate` (stable, widely used)

### Pattern 4: Retry with Exponential Backoff (RECOMMENDED)
**What:** On transient errors (429, 503, timeouts), retry with increasing delays + jitter
**When to use:** Handle temporary API unavailability
**Strategy:** Max 3 retries, base delay 1s, exponential multiplier 2, jitter ±20%
**Example:**
```go
func (c *GeminiClient) callGeminiAPIWithRetry(ctx context.Context, prompt string) (string, error) {
    maxRetries := 3
    baseDelay := 1 * time.Second

    for attempt := 0; attempt <= maxRetries; attempt++ {
        category, err := c.callGeminiAPI(ctx, prompt)

        if err == nil {
            return category, nil
        }

        // Check if error is retryable (429, 503, timeout)
        if !isRetryableError(err) || attempt == maxRetries {
            return "", err
        }

        // Exponential backoff with jitter
        delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
        jitter := time.Duration(rand.Int63n(int64(delay / 5)))  // ±20% jitter
        time.Sleep(delay + jitter)
    }
}
```

### Pattern 5: Confidence Metadata Propagation (RECOMMENDED)
**What:** Parse Gemini response for confidence scores; attach to Category; log before persistence
**When to use:** Track which categorizations are high-confidence vs. low-confidence
**Limitation:** Gemini `generateContent` API doesn't return explicit confidence in response; must estimate from:
- Response presence/completeness
- Category match against known list
- Prompt structure confidence indicators

**Example structure:**
```go
type Category struct {
    Name        string
    Description string
    Confidence  float64  // 0.0-1.0; added for Phase 8
    Source      string   // "direct_mapping" | "keyword" | "semantic" | "ai"
}
```

**Logging before save:**
```go
logger.WithFields(
    Field{Key: "party", Value: tx.PartyName},
    Field{Key: "category", Value: category.Name},
    Field{Key: "confidence", Value: category.Confidence},
    Field{Key: "source", Value: category.Source},
).Info("AI categorization (auto-learning)")
```

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Rate limiting with minute boundaries | Custom tracking with time.Now() comparisons | `golang.org/x/time/rate.Limiter` | Token bucket is standard, proven, handles edge cases (clock skew, burst semantics) |
| Retry logic with backoff | Manual for-loop with sleep() | Proven pattern with jitter calculation | Custom code often misses jitter (thundering herd), exponential overflow, context cancellation |
| Confidence scoring from unstructured API | Regex parsing of response text | Parse JSON response fields + heuristics | Cleaner, maintainable, avoids fragile string parsing |
| Config flag validation | Manual type conversion in commands | Viper + CLI flag binding | Viper handles type coercion, precedence, validation; CLI flags integrate seamlessly |

**Key insight:** Gemini API doesn't provide explicit confidence scores in its public API. Estimate confidence heuristically:
- If categorization matches known category list → confidence 0.9
- If empty/uncategorized response → confidence 0.0
- Use `confidence_threshold` from config as decision gate (current default 0.8, use for filtering)

---

## Common Pitfalls

### Pitfall 1: Auto-Learn Saves Before User Confirmation (Currently happening)
**What goes wrong:** Categorizer auto-saves mappings immediately on success; user has no way to review/reject bad AI decisions; once saved, mapping persists
**Why it happens:** Phase 6/7 implemented auto-learning as convenience feature; no gate existed
**How to avoid:**
- Add `isAutoLearnEnabled bool` parameter to Categorizer constructor
- Gate the save operations: `if c.isAutoLearnEnabled { c.updateDebitorCategory(...) }`
- Default to `false` (per D-11) — user must opt-in via `--auto-learn`
- Log intent at INFO level before saving: "Auto-learning enabled: saving creditor mapping"

**Warning signs:**
- Categorizer calls `SaveCreditorsToYAML()` unconditionally after any successful categorization
- No config check before save
- No CLI flag to disable

### Pitfall 2: Rate Limit Quota Exhaustion (Silent failure mode)
**What goes wrong:** Rapid batch processing hits Gemini API quota; requests start failing with 429 (Too Many Requests); application continues trying, wasting budget
**Why it happens:** No rate limiter in GeminiClient; `http.Client.Timeout` only handles slow responses, not quota
**How to avoid:**
- Add `rate.Limiter` initialized with `ai.requests_per_minute` from config
- Check `limiter.Allow()` BEFORE making HTTP request
- Return error immediately if rate limit exceeded (don't waste HTTP round-trip)
- Log rate limit events at WARN level so user knows to slow down

**Warning signs:**
- Batch operations hit 429 errors on large file sets
- No visible throttling in logs
- API quota exhausted on day 1 of heavy testing

### Pitfall 3: Transient Failures Treated as Permanent (User-facing breakage)
**What goes wrong:** Gemini API has temporary blip (503, network timeout); request fails once; entire batch aborts; user has to re-run manually
**Why it happens:** No retry logic; single failed API call propagates to user as final error
**How to avoid:**
- Implement retry-with-backoff for HTTP 429, 503, timeout errors
- Max 3 retries; base delay 1s; exponential backoff (2x); jitter ±20%
- Log each retry: "Retrying API request (attempt 2/3, wait 1.5s...)"
- Return error only after all retries exhausted

**Warning signs:**
- Batch of 100 transactions fails because 1 API call had network hiccup
- No retry attempts visible in logs
- User reports "sometimes it works, sometimes it doesn't" (flaky)

### Pitfall 4: Confidence Data Lost (Audit trail gap)
**What goes wrong:** Categorizer auto-learns mapping with 0.6 confidence (below threshold); mapping saved; weeks later, user has bad category; no log showing confidence was low
**Why it happens:** Category model doesn't carry confidence; once saved to YAML, confidence info is gone; no pre-save audit log
**How to avoid:**
- Add `Confidence float64` and `Source string` to Category struct
- Log categorization WITH confidence before any save: `logger.Info("Saving categorization", Field{Key: "confidence", Value: cat.Confidence})`
- Include source strategy name: `Field{Key: "source", Value: "ai"}`
- Users can later audit logs to understand why certain mappings exist

**Warning signs:**
- No confidence score in Category struct
- Logging happens AFTER save (can't trace back why it was saved)
- User has no way to know which categorizations are AI-derived vs. direct mappings

### Pitfall 5: Configuration Not Passed Through Container (Wiring gap)
**What goes wrong:** Config has `auto_learn: false`, but Categorizer doesn't receive it; uses hardcoded default instead; auto-learning still happens despite user setting flag
**Why it happens:** Container constructor receives config but doesn't pass `AutoLearn` flag to Categorizer
**How to avoid:**
- Add parameter: `NewCategorizer(aiClient, store, logger, autoLearnEnabled bool)`
- Pass from container: `categorizer.NewCategorizer(aiClient, store, logger, cfg.Categorization.AutoLearn)`
- Verify in tests: config `AutoLearn=false` → categorizer doesn't save mappings

**Warning signs:**
- Setting `CAMT_CATEGORIZATION_AUTO_LEARN=false` has no effect
- Mappings still auto-save after categorization succeeds
- Container initialization doesn't read `cfg.Categorization.AutoLearn`

---

## Code Examples

### Example 1: Rate Limiting Setup
**Source:** `golang.org/x/time/rate` (Go standard library extension, stable since Go 1.8)

```go
// internal/categorizer/gemini_client.go

import "golang.org/x/time/rate"

type GeminiClient struct {
    apiKey        string
    model         string
    httpClient    *http.Client
    log           logging.Logger
    limiter       *rate.Limiter  // NEW: rate limiter
    requestsPerMin int           // Store for reference
}

// NewGeminiClient creates a new GeminiClient with rate limiting
func NewGeminiClient(logger logging.Logger, requestsPerMinute int) *GeminiClient {
    if logger == nil {
        logger = logging.NewLogrusAdapterFromLogger(logrus.New())
    }

    if requestsPerMinute <= 0 {
        requestsPerMinute = 10  // Default from config
    }

    apiKey := os.Getenv("GEMINI_API_KEY")
    if apiKey == "" {
        logger.Warn("GEMINI_API_KEY not set, AI categorization will fail")
    }

    model := os.Getenv("GEMINI_MODEL")
    if model == "" {
        model = "gemini-2.5-flash"
        logger.WithField("model", model).Debug("GEMINI_MODEL not set, using default")
    }

    // Create rate limiter: requestsPerMinute divided by 60 = requests per second
    limiter := rate.NewLimiter(
        rate.Limit(float64(requestsPerMinute) / 60.0),
        1,  // Burst size = 1 (no bursting, strict rate limiting)
    )

    return &GeminiClient{
        apiKey:        apiKey,
        model:         model,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        log:            logger,
        limiter:        limiter,
        requestsPerMin: requestsPerMinute,
    }
}

// Categorize takes a transaction and returns categorized transaction or error
func (c *GeminiClient) Categorize(ctx context.Context, transaction models.Transaction) (models.Transaction, error) {
    if c.apiKey == "" {
        c.log.Debug("No API key available, skipping AI categorization")
        transaction.Category = models.CategoryUncategorized
        return transaction, nil
    }

    // Check rate limit BEFORE making API call
    if !c.limiter.Allow() {
        c.log.Warn("Rate limit exceeded, skipping categorization request")
        return transaction, fmt.Errorf("rate limit exceeded: %d requests per minute limit reached", c.requestsPerMin)
    }

    // Rest of existing Categorize logic...
    prompt := c.buildCategorizationPrompt(transaction)
    category, err := c.callGeminiAPIWithRetry(ctx, prompt)
    // ... rest of implementation
}
```

### Example 2: Retry with Exponential Backoff
**Source:** Standard Go patterns (errors package, time package)

```go
// internal/categorizer/gemini_client.go

import (
    "math/rand"
    "time"
)

// isRetryableError checks if an error is worth retrying
func (c *GeminiClient) isRetryableError(err error) bool {
    if err == nil {
        return false
    }

    // Check for timeout
    if os.IsTimeout(err) {
        return true
    }

    // Check for HTTP status codes in error message
    errStr := err.Error()
    if strings.Contains(errStr, "status 429") ||  // Too Many Requests
        strings.Contains(errStr, "status 503") || // Service Unavailable
        strings.Contains(errStr, "status 500") {  // Internal Server Error (sometimes retryable)
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

// callGeminiAPIWithRetry wraps callGeminiAPI with retry-backoff logic
func (c *GeminiClient) callGeminiAPIWithRetry(ctx context.Context, prompt string) (string, error) {
    const (
        maxRetries   = 3
        baseDelay    = 1 * time.Second
        backoffMultiplier = 2.0
        jitterFraction = 0.2  // ±20% jitter
    )

    var lastErr error

    for attempt := 0; attempt <= maxRetries; attempt++ {
        // Call API
        category, err := c.callGeminiAPI(ctx, prompt)

        if err == nil {
            return category, nil
        }

        lastErr = err

        // If error is not retryable, return immediately
        if !c.isRetryableError(err) {
            c.log.WithError(err).Warn("Non-retryable error from Gemini API")
            return "", err
        }

        // If this was the last retry, return error
        if attempt == maxRetries {
            c.log.WithError(err).WithField("attempts", attempt+1).Warn("All retry attempts exhausted")
            return "", fmt.Errorf("API request failed after %d attempts: %w", maxRetries+1, err)
        }

        // Calculate backoff delay with jitter
        delayMs := int64(math.Pow(backoffMultiplier, float64(attempt)) * float64(baseDelay.Milliseconds()))
        jitterMs := int64(float64(delayMs) * jitterFraction * (2*rand.Float64() - 1))  // ±jitter
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
```

### Example 3: Auto-Learn Control
**Source:** Existing Categorizer + config pattern

```go
// internal/categorizer/categorizer.go (Modified)

type Categorizer struct {
    // ... existing fields ...
    isAutoLearnEnabled bool  // NEW: Control flag
}

// NewCategorizer creates a new Categorizer with auto-learn control
func NewCategorizer(
    aiClient AIClient,
    store CategoryStoreInterface,
    logger logging.Logger,
    autoLearnEnabled bool,  // NEW parameter
) *Categorizer {
    if logger == nil {
        logger = logging.NewLogrusAdapter("info", "text")
    }

    c := &Categorizer{
        categories:         make([]models.CategoryConfig, 0, 50),
        creditorMappings:   make(map[string]string, 100),
        debitorMappings:    make(map[string]string, 100),
        configMutex:        sync.RWMutex{},
        isDirtyCreditors:   false,
        isDirtyDebitors:    false,
        store:              store,
        logger:             logger,
        aiClient:           aiClient,
        isAutoLearnEnabled: autoLearnEnabled,  // NEW: Store flag
    }

    // ... rest of existing initialization ...
    return c
}

// Categorize with auto-learn control
func (c *Categorizer) Categorize(ctx context.Context, partyName string, isDebtor bool, amount, date, info string) (models.Category, error) {
    transaction := Transaction{
        PartyName: partyName,
        IsDebtor:  isDebtor,
        Amount:    amount,
        Date:      date,
        Info:      info,
    }

    category, err := c.categorizeTransaction(ctx, transaction)

    // NEW: Only auto-learn if enabled AND categorization succeeded
    if err == nil && c.isAutoLearnEnabled &&
        category.Name != "" && category.Name != models.CategoryUncategorized {

        // Log with confidence before saving
        if isDebtor {
            c.logger.WithFields(
                logging.Field{Key: "party", Value: partyName},
                logging.Field{Key: "category", Value: category.Name},
                logging.Field{Key: "confidence", Value: category.Confidence},  // Log confidence
                logging.Field{Key: "source", Value: category.Source},
                logging.Field{Key: "action", Value: "auto_learn"},
            ).Info("Auto-learning debitor mapping")

            c.updateDebitorCategory(partyName, category.Name)
            if saveErr := c.SaveDebitorsToYAML(); saveErr != nil {
                c.logger.WithError(saveErr).Warn("Failed to save debitor mapping")
            }
        } else {
            c.logger.WithFields(
                logging.Field{Key: "party", Value: partyName},
                logging.Field{Key: "category", Value: category.Name},
                logging.Field{Key: "confidence", Value: category.Confidence},
                logging.Field{Key: "source", Value: category.Source},
                logging.Field{Key: "action", Value: "auto_learn"},
            ).Info("Auto-learning creditor mapping")

            c.updateCreditorCategory(partyName, category.Name)
            if saveErr := c.SaveCreditorsToYAML(); saveErr != nil {
                c.logger.WithError(saveErr).Warn("Failed to save creditor mapping")
            }
        }
    } else if err == nil && !c.isAutoLearnEnabled {
        // Log that auto-learning is disabled
        c.logger.WithFields(
            logging.Field{Key: "party", Value: partyName},
            logging.Field{Key: "category", Value: category.Name},
            logging.Field{Key: "action", Value: "skip_auto_learn"},
            logging.Field{Key: "reason", Value: "auto_learn_disabled"},
        ).Debug("Categorization found but auto-learning disabled")
    }

    return category, err
}
```

### Example 4: CLI Flag Wiring
**Source:** Existing Cobra + Viper pattern

```go
// cmd/root/root.go (Modified)

func Init() {
    // ... existing flags ...

    // NEW: Add --auto-learn flag
    Cmd.PersistentFlags().Bool("auto-learn", false, "Enable AI auto-learning of categorizations (default: false)")

    // Bind to Viper
    if err := viper.BindPFlag("categorization.auto_learn", Cmd.PersistentFlags().Lookup("auto-learn")); err != nil {
        log.Printf("Warning: failed to bind auto-learn flag: %v", err)
    }

    // ... existing bindings ...
}
```

```yaml
# ~/.camt-csv/camt-csv.yaml (user config example)

categorization:
  auto_learn: true  # Override default (default is false)
  confidence_threshold: 0.8
```

```bash
# CLI usage examples:
camt-csv camt convert input.xml output.csv --ai-enabled              # Auto-learn OFF (default)
camt-csv camt convert input.xml output.csv --ai-enabled --auto-learn # Auto-learn ON
CAMT_CATEGORIZATION_AUTO_LEARN=true camt-csv camt convert ...         # Via env var
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Gemini model `text-embedding-004` | `gemini-embedding-001` | Nov 2025 (latest commit history) | Deprecation handled; new model in use |
| Manual sleep-based backoff | Exponential backoff with jitter | Phase 8 (new requirement) | Prevents thundering herd; reduces API load |
| Categorization auto-save always on | Gated by `--auto-learn` flag | Phase 8 (D-11 locked decision) | User controls AI learning; safer defaults |
| Confidence lost after categorization | Confidence stored in Category struct | Phase 8 (new requirement) | Enables audit trail; users know confidence level |

**Deprecated/outdated:**
- Direct model name hardcoding (now reads `GEMINI_MODEL` env var with fallback)

---

## Open Questions

1. **Gemini Confidence Score Extraction**
   - What we know: Gemini `generateContent` API returns `candidates[0].content.parts[0].text` — a string category name
   - What's unclear: Does Gemini API provide confidence scores in the response? (Checked: no explicit `confidence` field in public API)
   - Recommendation: Estimate confidence heuristically:
     - If response matches known category list → 0.9
     - If response is empty/uncategorized → 0.0
     - Use `confidence_threshold` from config (default 0.8) for filtering
     - Log heuristic source: "Confidence is estimated (no explicit score in API response)"

2. **Retry Jitter Implementation**
   - What we know: Standard pattern is uniform random jitter ±20%
   - What's unclear: Should jitter be applied per-attempt or per-retry? (Answer: per-retry, so each attempt has independent jitter)
   - Recommendation: Use `rand.Float64() * 2 - 1` for uniform ±20% of base delay

3. **Rate Limiter Initialization Timing**
   - What we know: `rate.Limiter` should be created once at client init time
   - What's unclear: Should limiter be configurable per GeminiClient instance? (Answer: yes, via constructor parameter)
   - Recommendation: Pass `requestsPerMinute` from config to `NewGeminiClient(logger, requestsPerMinute)`

---

## Sources

### Primary (HIGH confidence)
- **`internal/config/viper.go`** - Configuration struct and defaults; `AutoLearn` and `RequestsPerMinute` fields verified
- **`internal/categorizer/gemini_client.go`** - GeminiClient HTTP client; timeout and API structure verified
- **`internal/categorizer/categorizer.go`** - Auto-learning logic; unconditional save behavior verified
- **`cmd/root/root.go`** - CLI flag binding pattern; existing `--ai-enabled` flag verified
- **`internal/container/container.go`** - Dependency injection pattern; container wiring verified
- **Go stdlib `golang.org/x/time/rate`** - Token bucket rate limiter (standard, stable since Go 1.8)

### Secondary (MEDIUM confidence)
- **`.planning/reference/v1.2-decisions.md`** - D-11 decision: "AI auto-learn defaults to OFF"
- **`internal/integration/cross_parser_test.go`** - Auto-learning test setup; shows current behavior
- **`internal/categorizer/ai_strategy.go`** - AI strategy pattern; categorization flow verified

### Tertiary (LOW confidence)
- Gemini API confidence scoring — not found in public docs; estimated via heuristics

---

## Metadata

**Confidence breakdown:**
- **Standard stack:** HIGH - Config system, DI container, CLI flag binding all verified in codebase
- **Architecture:** HIGH - Auto-learning gates, rate limiting, retry patterns are standard Go patterns
- **Pitfalls:** HIGH - Current unconditional auto-save and missing rate limits are explicit code issues
- **Code examples:** HIGH - Drawn from actual codebase patterns and Go stdlib docs

**Research date:** 2026-02-16
**Valid until:** 2026-02-23 (stable domain, no breaking changes expected)

---

## Key Findings Summary

1. **Configuration infrastructure ready** — `categorization.auto_learn` exists in config but isn't wired to Categorizer
2. **Auto-learning always happens** — Current code saves mappings unconditionally; needs gate
3. **No rate limiting** — GeminiClient has no throttle; batch operations can exhaust quota
4. **No retry logic** — Single transient error fails entire batch; needs backoff + retry
5. **Confidence data lost** — Gemini response parsed as plain string; confidence metadata not captured
6. **CLI flag missing** — `--auto-learn` not defined as CLI flag; only available via config file
7. **Go stdlib ready** — `golang.org/x/time/rate` provides production-grade rate limiter

**Phase 8 tasks will:**
1. Wire `--auto-learn` flag through CLI → config → Categorizer
2. Add rate limiter to GeminiClient
3. Add retry-with-backoff to GeminiClient (max 3 retries, exponential backoff, jitter)
4. Extend Category struct with Confidence and Source fields
5. Log confidence and source before persistence
6. Update container to pass `AutoLearn` flag to Categorizer constructor
