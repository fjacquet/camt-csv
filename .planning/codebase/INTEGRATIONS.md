# External Integrations

**Analysis Date:** 2026-02-01

## APIs & External Services

**AI Categorization:**
- **Google Gemini API** - LLM-based transaction categorization fallback
  - SDK/Client: Native HTTP client via `net/http`
  - Implementation: `internal/categorizer/gemini_client.go`
  - Auth: Environment variable `GEMINI_API_KEY`
  - Models:
    - `gemini-2.0-flash` (default, for categorization via generativelanguage.googleapis.com/v1beta/models/*/generateContent)
    - `text-embedding-004` (for semantic embeddings via generativelanguage.googleapis.com/v1beta/models/*/embedContent)
  - Timeout: 30 seconds (hardcoded)
  - Configuration: Controllable via `CAMT_AI_ENABLED`, `CAMT_AI_MODEL` env vars

## Data Storage

**Databases:**
- **None** - No relational or NoSQL database required

**File Storage:**
- Local filesystem only - YAML configuration files serve as persistent storage

**YAML Configuration Database:**
- Location: `database/` directory (relative) or from Viper config path resolution
- Files:
  - `categories.yaml` - Category definitions with keyword rules
  - `creditors.yaml` - Direct party-to-category mappings (payment recipients)
  - `debtors.yaml` - Direct party-to-category mappings (payment senders)
- Auto-learned mappings written back to YAML files when AI categorizes successfully
- Client: `gopkg.in/yaml.v3` for parsing and marshaling
- Path Resolution Order (via `internal/store/store.go`):
  1. Current directory
  2. `./config/` subdirectory
  3. `./database/` subdirectory
  4. `$HOME/.config/camt-csv/` (home directory fallback)

**Caching:**
- None - All data loaded from YAML files on demand

## Authentication & Identity

**Auth Provider:**
- None - No user authentication system

**API Authentication:**
- **Gemini API**: API Key via `GEMINI_API_KEY` environment variable
  - Requirement: Must be set in environment when `CAMT_AI_ENABLED=true`
  - Configuration validation enforces this in `internal/config/viper.go`
  - Special handling: Bound directly to GEMINI_API_KEY env var (not prefixed with CAMT_)

## Monitoring & Observability

**Error Tracking:**
- None - Application logs errors to console/file via logrus

**Logs:**
- Structured logging via **sirupsen/logrus**
- Format: Text (default) or JSON
- Levels: trace, debug, info, warn, error, fatal, panic
- Output: stdout/stderr
- No central log aggregation (logs stay local)

## CI/CD & Deployment

**Hosting:**
- Deployed as standalone binary (no container/serverless requirement)
- Statically compilable with `CGO_ENABLED=0 go build`

**CI Pipeline:**
- SLSA GoReleaser configured (`.slsa-goreleaser.yml`) for release automation
- Version injection via git tags in build process
- GitHub Actions capability (`.github/` directory present)

## Environment Configuration

**Required env vars (when AI enabled):**
- `GEMINI_API_KEY` - Google Gemini API key (required if `CAMT_AI_ENABLED=true`)

**Optional env vars:**
- `CAMT_LOG_LEVEL` - Log level (default: info)
- `CAMT_LOG_FORMAT` - Log format (default: text)
- `CAMT_AI_ENABLED` - Enable AI categorization (default: false)
- `CAMT_AI_MODEL` - Gemini model name (default: gemini-2.0-flash)
- `CAMT_CSV_DELIMITER` - CSV output delimiter (default: comma)
- `CAMT_DATA_DIRECTORY` - Data directory path (optional)

**Secrets location:**
- Environment variables only
- `.env` file auto-loaded from current directory via `github.com/joho/godotenv`
- Environment file NOT committed (`.env` in `.gitignore`)

## Webhooks & Callbacks

**Incoming:**
- None - CLI application, no webhook endpoints

**Outgoing:**
- None - No outbound callbacks or notifications

## External System Dependencies

**OS-Level Tool:**
- **pdftotext** (from Poppler utils) - Required for PDF statement parsing
  - Invoked via `os/exec` in `internal/pdfparser/pdfparser_helpers.go`
  - Command: `pdftotext -layout -raw <input> <output>`
  - Failure mode: PDF parsing fails gracefully if not installed

## Data Format Support (Input)

**File Formats Parsed:**
- CAMT.053 XML (ISO 20022) - Bank statements from XML exports
- PDF - Bank statements (requires pdftotext extraction + parsing)
- CSV - Revolut export format
- CSV - Selma export format
- CSV - Generic debit card export format
- JSON/YAML - Investment transaction exports (Revolut investment accounts)

**Output Format:**
- CSV (standardized format with configurable delimiter)

## Rate Limiting & Quotas

**Gemini API:**
- Default: 10 requests per minute (configurable via `CAMT_AI_MODEL`)
- Timeout per request: 30 seconds (hardcoded)
- No automatic retry logic (caller must handle)

---

*Integration audit: 2026-02-01*
