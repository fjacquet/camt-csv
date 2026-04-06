# CAMT-CSV

> Convert financial statements to CSV with intelligent transaction categorization

[![Go CI](https://github.com/fjacquet/camt-csv/actions/workflows/go.yml/badge.svg)](https://github.com/fjacquet/camt-csv/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/fjacquet/camt-csv/graph/badge.svg?token=ST9KKUV81N)](https://codecov.io/gh/fjacquet/camt-csv)
[![Go Report Card](https://goreportcard.com/badge/github.com/fjacquet/camt-csv)](https://goreportcard.com/report/github.com/fjacquet/camt-csv)
[![Go Reference](https://pkg.go.dev/badge/github.com/fjacquet/camt-csv.svg)](https://pkg.go.dev/github.com/fjacquet/camt-csv)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub release](https://img.shields.io/github/v/release/fjacquet/camt-csv)](https://github.com/fjacquet/camt-csv/releases/latest)
[![Docker Pulls](https://img.shields.io/badge/docker-ghcr.io-blue)](https://github.com/fjacquet/camt-csv/pkgs/container/camt-csv)

CAMT-CSV converts financial statement formats (CAMT.053 XML, PDF, Revolut CSV, Revolut Crypto CSV, Selma CSV) into standardized CSV files with AI-powered transaction categorization.

## Installation

```bash
# Homebrew (macOS/Linux)
brew tap fjacquet/homebrew-tap
brew install camt-csv

# Docker (multi-arch: amd64/arm64)
docker pull ghcr.io/fjacquet/camt-csv:latest

# Binary — download from GitHub Releases
# https://github.com/fjacquet/camt-csv/releases/latest

# Build from source
git clone https://github.com/fjacquet/camt-csv.git
cd camt-csv
make build
# For PDF support: brew install poppler (macOS) or apt-get install poppler-utils (Ubuntu)
```

## Usage

```bash
# Convert CAMT.053 XML
camt-csv camt -i statement.xml -o output.csv

# Convert Revolut CSV
camt-csv revolut -i revolut_export.csv -o output.csv

# Convert PDF bank statement
camt-csv pdf -i statement.pdf -o output.csv

# Revolut investment transactions
camt-csv revolut-investment -i investments.csv -o output.csv

# Revolut Crypto transactions
camt-csv revolut-crypto -i crypto.csv -o output.csv

# Selma investment CSV
camt-csv selma -i selma.csv -o output.csv

# Generic debit CSV
camt-csv debit -i debit.csv -o output.csv

# Batch process a directory
camt-csv batch -i input_dir/ -o output_dir/

# Use iCompta output format (semicolon-delimited, 10 columns)
camt-csv revolut -i export.csv -o output.csv --format icompta

# Enable AI categorization
camt-csv --ai-enabled --auto-learn camt -i statement.xml -o output.csv

# Check version
camt-csv --version
```

## Configuration

Settings can be provided via config file, environment variables, or CLI flags (highest precedence).

```bash
# Config file
mkdir -p ~/.camt-csv
cat > ~/.camt-csv/camt-csv.yaml << EOF
log:
  level: "info"
ai:
  enabled: true
categorization:
  auto_learn: false
EOF

# Environment variables
export GEMINI_API_KEY=your_api_key_here
export CAMT_AI_ENABLED=true
export CAMT_LOG_LEVEL=debug
```

See the [User Guide](https://fjacquet.github.io/camt-csv/user-guide/) for the complete configuration reference.

## Categorization

Four-tier strategy pattern for transaction categorization:

1. **Direct Mapping** - Exact match from `creditors.yaml`/`debtors.yaml`
2. **Keyword Matching** - Pattern rules from `categories.yaml`
3. **Semantic Search** - Vector embedding similarity matching
4. **AI Fallback** - Gemini API for unknown transactions (optional, requires `--ai-enabled`)

When `--auto-learn` is enabled, AI results are saved directly to the main YAML files. When disabled (the default), AI suggestions are saved to staging files (`database/staging_creditors.yaml`, `database/staging_debtors.yaml`) for manual review.

## Documentation

Full documentation: **https://fjacquet.github.io/camt-csv/**

- [User Guide](https://fjacquet.github.io/camt-csv/user-guide/) - Usage, configuration, troubleshooting
- [Developer Guide](https://fjacquet.github.io/camt-csv/developer-guide/) - Contributing, architecture, adding parsers
- [API Specifications](https://fjacquet.github.io/camt-csv/api-specifications/) - Interfaces and data models
- [Architecture Decision Records](https://fjacquet.github.io/camt-csv/adr/ADR-001-parser-interface-standardization/) - Design decisions

## Development

```bash
make all              # Lint, test, and build
make test             # Run tests
make lint             # Run golangci-lint
make coverage         # Generate coverage report
make install-tools    # Install dev tools
```

## License

MIT License - see [LICENSE](LICENSE) file.
