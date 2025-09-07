# CAMT-CSV

> Convert financial statements to CSV with intelligent transaction categorization

CAMT-CSV is a powerful command-line tool that converts various financial statement formats (CAMT.053 XML, PDF, Revolut CSV, Selma CSV, and more) into standardized CSV files with AI-powered transaction categorization.

[![Go CI](https://github.com/fjacquet/camt-csv/actions/workflows/go.yml/badge.svg)](https://github.com/fjacquet/camt-csv/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/fjacquet/camt-csv/graph/badge.svg?token=ST9KKUV81N)](https://codecov.io/gh/fjacquet/camt-csv)

## ✨ Key Features

- **Multi-format Support**: CAMT.053 XML, PDF bank statements, Revolut, Revolut Investment, Selma, and generic CSV
- **Smart Categorization**: Hybrid approach using local rules + AI fallback
- **Hierarchical Configuration**: Viper-based config system with files, environment variables, and CLI flags
- **Batch Processing**: Handle multiple files at once
- **Investment Support**: Dedicated parser for Revolut investment transactions
- **Fast & Reliable**: Local processing with optional cloud AI

## 🚀 Quick Start

### Installation

```bash
# Clone and build
git clone https://github.com/fjacquet/camt-csv.git
cd camt-csv
go build

# For PDF support, install poppler:
# macOS: brew install poppler
# Ubuntu: apt-get install poppler-utils
```

### Basic Usage

```bash
# Convert CAMT.053 XML to CSV
./camt-csv camt -i statement.xml -o transactions.csv

# Process PDF bank statement
./camt-csv pdf -i statement.pdf -o transactions.csv

# Convert Revolut export
./camt-csv revolut -i revolut.csv -o processed.csv

# Convert Revolut investment transactions
./camt-csv revolut-investment -i investments.csv -o processed.csv

# Batch process multiple files
./camt-csv batch -i input_dir/ -o output_dir/
```

### Configuration

CAMT-CSV supports hierarchical configuration with multiple options:

```bash
# Option 1: Configuration file (recommended)
mkdir -p ~/.camt-csv
cat > ~/.camt-csv/config.yaml << EOF
log:
  level: "info"
  format: "text"
csv:
  delimiter: ","
  include_headers: true
ai:
  enabled: true
  model: "gemini-2.0-flash"
EOF

# Option 2: Environment variables (backward compatible)
export GEMINI_API_KEY=your_api_key_here
export CAMT_LOG_LEVEL=debug
export CAMT_AI_ENABLED=true

# Option 3: CLI flags (temporary overrides)
./camt-csv --log-level debug --ai-enabled camt -i file.xml -o output.csv
```

See [Configuration Migration Guide](docs/configuration-migration-guide.md) for complete details.

## 📚 Documentation

- **[User Guide](docs/user-guide.md)** - Complete usage guide with examples and troubleshooting
- **[Codebase Documentation](docs/codebase_documentation.md)** - Technical architecture and development details
- **[Design Principles](docs/design-principles.md)** - Core design philosophy and patterns

## 🏗️ Supported Formats

| Format | Description | Command |
|--------|-------------|----------|
| **CAMT.053 XML** | ISO 20022 bank statements | `camt` |
| **PDF** | Bank statements (including Viseca) | `pdf` |
| **Revolut CSV** | Revolut app exports | `revolut` |
| **Revolut Investment** | Revolut investment transactions | `revolut-investment` |
| **Selma CSV** | Investment platform data | `selma` |
| **Generic CSV** | Debit transaction files | `debit` |

## 🤖 Smart Categorization

CAMT-CSV uses a three-tier approach for transaction categorization:

1. **Direct Mapping** - Instant recognition of known payees
2. **Keyword Matching** - Local rules from `database/categories.yaml`
3. **AI Fallback** - Gemini AI for unknown transactions (optional)

## 🛠️ Development

```bash
# Run tests
go test ./...

# Build for production
go build -ldflags="-s -w"

# View help
./camt-csv --help
```

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

---

**Need help?** Check the [User Guide](docs/user-guide.md) for detailed instructions and examples.
