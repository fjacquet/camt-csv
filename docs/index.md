# CAMT-CSV

> Convert financial statements to CSV with intelligent transaction categorization

CAMT-CSV is a command-line tool that converts various financial statement formats (CAMT.053 XML, PDF, Revolut CSV, Selma CSV, and more) into standardized CSV files with AI-powered transaction categorization.

[![Go CI](https://github.com/fjacquet/camt-csv/actions/workflows/go.yml/badge.svg)](https://github.com/fjacquet/camt-csv/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/fjacquet/camt-csv/graph/badge.svg?token=ST9KKUV81N)](https://codecov.io/gh/fjacquet/camt-csv)

## Key Features

- **Multi-format Support**: CAMT.053 XML, PDF, Revolut CSV, Selma CSV, debit CSV, and Revolut investment CSV
- **Smart Categorization**: Four-tier hybrid approach using direct mapping, keyword matching, semantic vector search, and AI fallback
- **Dependency Injection Architecture**: Clean architecture with explicit dependencies through `Container` pattern
- **Batch Processing**: Handle multiple files at once with automatic format detection
- **Hierarchical Configuration**: Viper-based config supporting files, environment variables, and CLI flags

## Quick Start

```bash
# Clone and build
git clone https://github.com/fjacquet/camt-csv.git
cd camt-csv
go build

# Convert a CAMT.053 file
./camt-csv camt -i statement.xml -o processed.csv

# Convert Revolut transactions
./camt-csv revolut -i revolut-export.csv -o output.csv

# Batch process multiple files
./camt-csv batch -i input_dir/ -o output_dir/
```

## Supported Formats

| Parser | Format | Description |
|--------|--------|-------------|
| `camt` | CAMT.053 XML | ISO 20022 bank statements |
| `pdf` | PDF | PDF bank statements (Viseca, generic) |
| `revolut` | CSV | Revolut app CSV exports |
| `revolut-investment` | CSV | Revolut investment transactions |
| `selma` | CSV | Selma investment platform data |
| `debit` | CSV | Generic debit transactions |

## Learn More

- [User Guide](user-guide.md) - Complete usage guide with examples
- [Developer Guide](developer-guide.md) - Contributing and development setup
- [Architecture](architecture.md) - Technical architecture documentation
