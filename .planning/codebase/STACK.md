# Technology Stack

**Analysis Date:** 2026-02-01

## Languages

**Primary:**
- Go 1.24.2 - Entire application codebase, CLI tool, parsers, categorization engine

## Runtime

**Environment:**
- Go 1.24.2 runtime
- No runtime dependencies beyond Go standard library and dependencies

**Package Manager:**
- Go modules (go.mod / go.sum)
- Lockfile: Present (`go.sum`)

## Frameworks

**Core:**
- Cobra 1.10.2 - CLI framework for command structure and flag parsing (`cmd/` directory)
- Viper 1.21.0 - Configuration management with hierarchical loading (environment vars, config files, defaults)

**Data Processing:**
- antchfx/xmlquery 1.5.0 - XML parsing and querying for CAMT.053 ISO 20022 XML files
- antchfx/xpath 1.3.5 (indirect) - XPath support for XML traversal
- gocarina/gocsv 0.0.0-20240520201108-78e41c74b4b1 - CSV parsing and writing

**Logging & Monitoring:**
- sirupsen/logrus 1.9.4 - Structured logging with text/JSON formatters

**Testing:**
- stretchr/testify 1.11.1 - Assertion and mocking utilities for test suites
- Built-in Go testing framework (`testing` package)

## Key Dependencies

**Critical:**
- `github.com/antchfx/xmlquery` - Essential for parsing CAMT.053 XML bank statements
- `github.com/spf13/viper` - Configuration hierarchy (files, env vars, defaults)
- `github.com/spf13/cobra` - CLI command routing and argument parsing
- `github.com/sirupsen/logrus` - Application-wide logging

**Infrastructure:**
- `github.com/shopspring/decimal` 1.4.0 - Precise monetary amount handling (no float rounding errors)
- `github.com/google/uuid` 1.6.0 - UUID generation
- `github.com/joho/godotenv` 1.5.1 - Load `.env` files for local development
- `gopkg.in/yaml.v3` 3.0.1 - YAML parsing for configuration files and category databases
- `golang.org/x/net` 0.49.0 - Network utilities, XML namespace handling

**Development Tools:**
- `golangci-lint` - Linting (executes: errcheck, ineffassign, unused, gosec, gofmt)
- `gosec` - Security scanning
- `cyclonedx-gomod` - Software Bill of Materials generation

## Configuration

**Environment:**
- Viper-based hierarchical loading (in order of precedence):
  1. Environment variables (prefix `CAMT_`)
  2. Configuration file at `~/.camt-csv/config.yaml` or `.camt-csv/config.yaml`
  3. Hardcoded defaults in code

**Key Configuration Options:**
- `CAMT_LOG_LEVEL` - Log level (trace, debug, info, warn, error, fatal, panic)
- `CAMT_LOG_FORMAT` - Log format (text, json)
- `CAMT_AI_ENABLED` - Enable/disable Gemini API categorization
- `CAMT_AI_MODEL` - Gemini model (default: gemini-2.0-flash)
- `GEMINI_API_KEY` - Google Gemini API key (special case, not prefixed)
- `CAMT_CSV_DELIMITER` - CSV output delimiter (default: comma)

**Build:**
- `Makefile` with targets: build, test, coverage, lint, security, sbom, clean
- Build variables inject version and timestamp via ldflags
- Go module verification enabled (go mod verify)

## Platform Requirements

**Development:**
- Go 1.24.2 or later
- `pdftotext` command-line tool (for PDF statement extraction, system dependency)
- Git (for version information in build)
- make (for Makefile targets)

**Optional Development Tools:**
- golangci-lint - Code linting
- gosec - Security scanning
- cyclonedx-gomod - SBOM generation

**Production:**
- Go runtime (statically compilable with CGO_ENABLED=0)
- `pdftotext` system utility (from Poppler utils, required for PDF parsing)
- No database server required (file-based YAML storage)

## Security & Linting Configuration

**golangci-lint** (`.golangci.yml`):
- Enabled linters: errcheck, ineffassign, unused, gosec
- Disabled: staticcheck, govet, misspell
- Formatter: gofmt

**gosec** (`.gosec.json`):
- Excluded rules:
  - G101: Hardcoded credentials (false positive for XPath constants)
  - G204: Subprocess execution (expected for pdftotext)
  - G304: File inclusion via variable (expected for file parsers)

---

*Stack analysis: 2026-02-01*
