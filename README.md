# CAMT-CSV

> Convert financial statements to CSV with intelligent transaction categorization

CAMT-CSV is a powerful command-line tool that converts various financial statement formats (CAMT.053 XML, PDF, Revolut CSV, Selma CSV, and more) into standardized CSV files with AI-powered transaction categorization.

[![Go CI](https://github.com/fjacquet/camt-csv/actions/workflows/go.yml/badge.svg)](https://github.com/fjacquet/camt-csv/actions/workflows/go.yml)



## ✨ Key Features

- **Multi-format Support**: CAMT.053 XML, PDF bank statements, Revolut, Selma, and generic CSV
- **Smart Categorization**: Hybrid approach using local rules + AI fallback
- **Batch Processing**: Handle multiple files at once
- **Configurable**: Custom delimiters, logging, and AI settings
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

# Batch process multiple files
./camt-csv batch -i input_dir/ -o output_dir/
```

### Configuration (Optional)

```bash
# Copy sample config and customize
cp .env.sample .env
nano .env  # Add your Gemini API key for AI categorization
```

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
