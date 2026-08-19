# CAMT-CSV

> Convert financial statements to CSV with intelligent transaction categorization

CAMT-CSV is a command-line tool that converts various financial statement formats (CAMT.053 XML, PDF, Revolut CSV, Revolut Crypto CSV, Revolut Investment CSV, Selma CSV, Viseca CSV, and generic debit CSV) into standardized CSV files with AI-powered transaction categorization.

[![Go CI](https://github.com/fjacquet/camt-csv/actions/workflows/go.yml/badge.svg)](https://github.com/fjacquet/camt-csv/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/fjacquet/camt-csv/graph/badge.svg?token=ST9KKUV81N)](https://codecov.io/gh/fjacquet/camt-csv)

## Key Features

- **Multi-format Support**: CAMT.053 XML, PDF, Revolut CSV, Revolut Crypto CSV, Revolut Investment CSV, Selma CSV, Viseca CSV, and generic debit CSV
- **Smart Categorization**: Four-tier hybrid approach using direct mapping, keyword matching, semantic vector search, and AI fallback
- **Dependency Injection Architecture**: Clean architecture with explicit dependencies through `Container` pattern
- **Format Auto-Detection**: `convert` offers each file to every parser in turn and uses the first that recognizes it — no per-format command to remember
- **Hierarchical Configuration**: Viper-based config supporting files, environment variables, and CLI flags

## Quick Start

```bash
# Clone and build
git clone https://github.com/fjacquet/camt-csv.git
cd camt-csv
go build

# Convert a statement — the format is auto-detected
./camt-csv convert -i statement.xml -o processed.csv
./camt-csv convert -i revolut-export.csv -o output.csv

# Convert a directory into one date-sorted CSV
./camt-csv convert -i input_dir/ -o output.csv
```

## Supported Formats

| Format | `--from` value | Description |
|--------|-----------------|-------------|
| CAMT.053 XML | `camt` | ISO 20022 bank statements |
| PDF | `pdf` | PDF bank statements (Viseca, generic) |
| Revolut CSV | `revolut` | Revolut app CSV exports |
| Revolut Crypto CSV | `revolut-crypto` | Revolut crypto account exports |
| Revolut Investment CSV | `revolut-investment` | Revolut investment transactions |
| Selma CSV | `selma` | Selma investment platform data |
| Viseca CSV | `viseca` | Viseca One portal card export |
| Generic debit CSV | `debit` | Generic debit transactions |

`convert` detects the format automatically; pass `--from <value>` to pin one when detection guesses wrong.

## Learn More

- [User Guide](user-guide.md) - Complete usage guide with examples
- [Developer Guide](developer-guide.md) - Contributing and development setup
- [Architecture](architecture.md) - Technical architecture documentation
