# CAMT-CSV User Guide

## Table of Contents

1. [Introduction](#introduction)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [Basic Usage](#basic-usage)
5. [Advanced Features](#advanced-features)
6. [File Format Support](#file-format-support)
7. [Transaction Categorization](#transaction-categorization)
8. [Troubleshooting](#troubleshooting)
9. [Examples](#examples)

## Introduction

CAMT-CSV is a powerful command-line tool that converts various financial statement formats into standardized CSV files with intelligent transaction categorization. It supports multiple input formats and uses a hybrid approach combining local rules with AI-powered categorization.

### Key Features

- **Multi-format Support**: CAMT.053 XML, PDF bank statements, Revolut CSV (English and French locales), Revolut Crypto CSV, Revolut Investment CSV, Selma investment CSV, and generic debit CSV
- **Smart Categorization**: Four-tier strategy pattern using direct mapping, keyword matching, semantic search, and AI fallback with auto-learning
- **Dependency Injection Architecture**: Clean architecture with explicit dependencies, eliminating global state
- **Hierarchical Configuration**: Viper-based configuration system with config files, environment variables, and CLI flags
- **Batch Processing**: Process multiple files at once with automatic format detection
- **Investment Support**: Dedicated parser for Revolut investment transactions with specialized categorization
- **Extensible Architecture**: Standardized parser interfaces with BaseParser foundation and segregated interfaces
- **Comprehensive Error Handling**: Custom error types with detailed context and proper error wrapping
- **Framework-Agnostic Logging**: Structured logging abstraction with dependency injection and configurable backends
- **Performance Optimized**: String operations optimization, lazy initialization, and pre-allocation for efficient processing

## Installation

### Homebrew (macOS / Linux) — Recommended

```bash
brew tap fjacquet/homebrew-tap
brew install camt-csv
```

### Docker

Multi-arch images (amd64/arm64) are available on GitHub Container Registry:

```bash
docker pull ghcr.io/fjacquet/camt-csv:latest

# Run directly
docker run --rm -v $(pwd):/data ghcr.io/fjacquet/camt-csv:latest camt -i /data/statement.xml -o /data/output.csv
```

### Binary Download

Download pre-built binaries from [GitHub Releases](https://github.com/fjacquet/camt-csv/releases/latest) for linux/darwin/windows (amd64/arm64).

### Building from Source

Prerequisites:
- **Go 1.24.2 or higher**: [Download Go](https://golang.org/dl/)
- **pdftotext CLI tool** (for PDF processing):
  - **macOS**: `brew install poppler`
  - **Ubuntu/Debian**: `apt-get install poppler-utils`
  - **Windows**: [Download Poppler for Windows](http://blog.alivate.com.au/poppler-windows/)

```bash
git clone https://github.com/fjacquet/camt-csv.git
cd camt-csv
make build
```

### Verify Installation

```bash
camt-csv --version
camt-csv --help
```

## Configuration

CAMT-CSV uses a hierarchical configuration system, allowing you to manage settings flexibly. Settings are applied in the following order of precedence (highest to lowest):

1.  **CLI Flags**: Options passed directly on the command line (e.g., `--log-level debug`).
2.  **Environment Variables**: Variables prefixed with `CAMT_` (e.g., `CAMT_LOG_LEVEL=debug`).
3.  **Configuration File**: A `camt-csv.yaml` file located in `~/.camt-csv/` or `.camt-csv/config.yaml`.

### Setting Up Configuration

Create and edit the configuration file for persistent settings:

```bash
mkdir -p ~/.camt-csv
nano ~/.camt-csv/camt-csv.yaml  # or your preferred editor
```

### Global Configuration Options

All commands support these global flags and configuration options:

#### Core Options

| YAML Key | Environment Variable | CLI Flag | Default | Description |
|----------|---------------------|----------|---------|-------------|
| - | - | `--config` | `$HOME/.camt-csv/config.yaml` | Config file path |
| - | - | `-i, --input` | - | Input file or directory |
| - | - | `-o, --output` | - | Output file or directory |
| - | - | `-v, --validate` | `false` | Validate format before conversion |

#### Logging

| YAML Key | Environment Variable | CLI Flag | Default | Description |
|----------|---------------------|----------|---------|-------------|
| `log.level` | `CAMT_LOG_LEVEL` | `--log-level` | `info` | Log level (debug, info, warn, error) |
| `log.format` | `CAMT_LOG_FORMAT` | `--log-format` | `text` | Log format (text, json) |

#### CSV Output

| YAML Key | Environment Variable | CLI Flag | Default | Description |
|----------|---------------------|----------|---------|-------------|
| `csv.delimiter` | `CAMT_CSV_DELIMITER` | `--csv-delimiter` | `,` | CSV delimiter character |
| `csv.date_format` | `CAMT_CSV_DATE_FORMAT` | - | `DD.MM.YYYY` | Date format for CSV output |
| `csv.include_headers` | `CAMT_CSV_INCLUDE_HEADERS` | - | `true` | Include CSV header row |
| `csv.quote_all` | `CAMT_CSV_QUOTE_ALL` | - | `false` | Quote all CSV fields |

#### AI Categorization

| YAML Key | Environment Variable | CLI Flag | Default | Description |
|----------|---------------------|----------|---------|-------------|
| `ai.enabled` | `CAMT_AI_ENABLED` | `--ai-enabled` | `false` | Enable AI categorization |
| `ai.api_key` | `GEMINI_API_KEY` | - | - | Gemini API key |
| `ai.model` | `CAMT_AI_MODEL` | - | `gemini-2.0-flash` | AI model to use |
| `ai.requests_per_minute` | `CAMT_AI_REQUESTS_PER_MINUTE` | - | `10` | API rate limit |
| `ai.timeout_seconds` | `CAMT_AI_TIMEOUT_SECONDS` | - | `30` | API request timeout |
| `ai.fallback_category` | `CAMT_AI_FALLBACK_CATEGORY` | - | `Uncategorized` | Category when AI fails |

#### Categorization

| YAML Key | Environment Variable | CLI Flag | Default | Description |
|----------|---------------------|----------|---------|-------------|
| `categorization.auto_learn` | `CAMT_CATEGORIZATION_AUTO_LEARN` | `--auto-learn` | `false` | Auto-save AI categorizations to YAML |
| `categorization.confidence_threshold` | `CAMT_CATEGORIZATION_CONFIDENCE_THRESHOLD` | - | `0.8` | Minimum confidence threshold |
| `categorization.case_sensitive` | `CAMT_CATEGORIZATION_CASE_SENSITIVE` | - | `false` | Case-sensitive matching |

**Auto-Learn Behavior**:
- **`--auto-learn` enabled**: AI categorizations are saved directly to `creditors.yaml`/`debtors.yaml`. Backups are created automatically before each write.
- **`--auto-learn` disabled** (default): AI categorizations are saved to staging files (`staging_creditors.yaml`/`staging_debtors.yaml`) for manual review. You can copy approved entries to the main files.

#### Staging

| YAML Key | Environment Variable | CLI Flag | Default | Description |
|----------|---------------------|----------|---------|-------------|
| `staging.enabled` | `CAMT_STAGING_ENABLED` | - | `true` | Save AI suggestions to staging files when auto-learn is off |
| `staging.creditors_file` | `CAMT_STAGING_CREDITORS_FILE` | - | `staging_creditors.yaml` | Staging file for creditor suggestions |
| `staging.debtors_file` | `CAMT_STAGING_DEBTORS_FILE` | - | `staging_debtors.yaml` | Staging file for debtor suggestions |

#### Data and Backup

| YAML Key | Environment Variable | CLI Flag | Default | Description |
|----------|---------------------|----------|---------|-------------|
| `data.directory` | `CAMT_DATA_DIRECTORY` | - | - | Custom data directory |
| `data.backup_enabled` | `CAMT_DATA_BACKUP_ENABLED` | - | `true` | Enable backups |
| `backup.enabled` | `CAMT_BACKUP_ENABLED` | - | `true` | Enable backup system |
| `backup.directory` | `CAMT_BACKUP_DIRECTORY` | - | - | Backup directory |
| `categories.file` | `CAMT_CATEGORIES_FILE` | - | `categories.yaml` | Categories file |
| `categories.creditors_file` | `CAMT_CATEGORIES_CREDITORS_FILE` | - | `creditors.yaml` | Creditors mapping file |
| `categories.debtors_file` | `CAMT_CATEGORIES_DEBTORS_FILE` | - | `debtors.yaml` | Debtors mapping file |

#### Parser-Specific Settings

| YAML Key | Environment Variable | CLI Flag | Default | Description |
|----------|---------------------|----------|---------|-------------|
| `parsers.camt.strict_validation` | `CAMT_PARSERS_CAMT_STRICT_VALIDATION` | - | `true` | Strict CAMT validation |
| `parsers.pdf.ocr_enabled` | `CAMT_PARSERS_PDF_OCR_ENABLED` | - | `false` | Enable OCR for PDF |
| `parsers.revolut.date_format_detection` | `CAMT_PARSERS_REVOLUT_DATE_FORMAT_DETECTION` | - | `true` | Auto-detect date format |

### Command-Specific Flags

#### Parser Commands (camt, pdf, revolut, revolut-crypto, revolut-investment, selma, debit)

| CLI Flag | Default | Description |
|----------|---------|-------------|
| `-f, --format` | `standard` | Output format: `standard` (29-col, comma) or `icompta` (10-col, semicolon, dd.MM.yyyy) |
| `--date-format` | `DD.MM.YYYY` | Date format in output |

#### PDF Command Only

| CLI Flag | Default | Description |
|----------|---------|-------------|
| `--batch` | `false` | Batch mode: convert each PDF individually |

#### Categorize Command

| CLI Flag | Default | Description |
|----------|---------|-------------|
| `-p, --party` | - | Party name (required) |
| `-d, --debtor` | `false` | Whether party is debtor |
| `-a, --amount` | - | Transaction amount |
| `-t, --date` | - | Transaction date |
| `-n, --info` | - | Additional info |

### Example Configuration

Complete example of `~/.camt-csv/camt-csv.yaml`:

```yaml
# Logging configuration
log:
  level: "info"
  format: "text"

# CSV output settings
csv:
  delimiter: ";"
  date_format: "DD.MM.YYYY"
  include_headers: true
  quote_all: false

# AI categorization
ai:
  enabled: true
  model: "gemini-2.0-flash"
  requests_per_minute: 10
  timeout_seconds: 30
  fallback_category: "Uncategorized"

# Categorization behavior
categorization:
  auto_learn: false
  confidence_threshold: 0.8
  case_sensitive: false

# Data management
data:
  backup_enabled: true

# Category files
categories:
  file: "categories.yaml"
  creditors_file: "creditors.yaml"
  debtors_file: "debtors.yaml"

# Staging (AI suggestions when auto-learn is off)
staging:
  enabled: true
  creditors_file: "staging_creditors.yaml"
  debtors_file: "staging_debtors.yaml"

# Parser-specific settings
parsers:
  camt:
    strict_validation: true
  pdf:
    ocr_enabled: false
  revolut:
    date_format_detection: true
```

To set the API key, use the environment variable:

```bash
export GEMINI_API_KEY=your_api_key_here
```

## Basic Usage

### Command Structure

All CAMT-CSV commands follow this pattern:

```bash
./camt-csv [command] -i [input_file] -o [output_file]
```

### Supported Commands

| Command | Description | Input Format |
|---------|-------------|--------------|
| `camt` | Convert CAMT.053 XML files | XML bank statements |
| `pdf` | Convert PDF bank statements | PDF files |
| `revolut` | Process Revolut CSV exports | Revolut CSV format |
| `revolut-crypto` | Process Revolut Crypto account CSV exports | Revolut Crypto CSV (French locale) |
| `revolut-investment` | Process Revolut investment transactions | Revolut investment CSV format |
| `selma` | Process Selma investment files | Selma CSV format |
| `debit` | Process generic debit CSV files | Generic CSV format |
| `batch` | Process multiple files | Directory of files |
| `categorize` | Categorize existing transactions | CSV files |

### Quick Start Examples

1. **Convert a CAMT.053 XML file:**

   ```bash
   ./camt-csv camt -i bank_statement.xml -o transactions.csv
   ```

2. **Process a PDF bank statement:**

   ```bash
   ./camt-csv pdf -i statement.pdf -o transactions.csv
   ```

3. **Convert Revolut export:**

   ```bash
   ./camt-csv revolut -i revolut_export.csv -o processed.csv
   ```

4. **Process Revolut Crypto transactions:**

   ```bash
   ./camt-csv revolut-crypto -i crypto_export.csv -o processed.csv
   ```

5. **Process Revolut investment transactions:**

   ```bash
   ./camt-csv revolut-investment -i investment_export.csv -o processed.csv
   ```

## Advanced Features

### Batch Processing

Process multiple files in a directory:

```bash
./camt-csv batch -i input_directory -o output_directory
```

**Features:**

- Automatically detects file types
- Processes all supported formats
- Maintains original filenames with `.csv` extension
- Skips unsupported files with warnings

### Transaction Categorization

CAMT-CSV uses a sophisticated three-tier categorization system:

1. **Direct Mapping** (fastest): Exact matches from learned patterns
2. **Keyword Matching**: Local rules from `database/categories.yaml`
3. **AI Categorization** (fallback): Gemini AI for unknown transactions

#### Customizing Categories

Edit `database/categories.yaml` to add custom categories:

```yaml
categories:
  - name: "Groceries"
    keywords:
      - "supermarket"
      - "grocery"
      - "food store"
  
  - name: "Transportation"
    keywords:
      - "uber"
      - "taxi"
      - "bus"
      - "train"
```

#### AI Categorization Setup

1.  Get a Google AI API key from [Google AI Studio](https://makersuite.google.com/app/apikey)
2.  Set your API key as an environment variable:

    ```bash
    export GEMINI_API_KEY=your_api_key_here
    ```

3.  Enable AI categorization in `~/.camt-csv/camt-csv.yaml`:

    ```yaml
    ai:
      enabled: true
    ```

### Custom Output Formats

#### Change CSV Delimiter

For European Excel compatibility, set the delimiter in `~/.camt-csv/camt-csv.yaml`:

```yaml
csv:
  delimiter: ";"
```

#### Custom Data Directory

Store configuration files in a custom location by setting the `CAMT_DATA_DIRECTORY` environment variable:

```bash
export CAMT_DATA_DIRECTORY="/path/to/custom/data"
./camt-csv camt -i input.xml -o output.csv
```

## File Format Support

### CAMT.053 XML Files

**Description**: ISO 20022 standard bank statement format
**Features**:

- Complete transaction details
- Multi-currency support
- Reference numbers and codes
- Party information (payer/payee)

**Example Usage**:

```bash
./camt-csv camt -i bank_statement.xml -o transactions.csv
```

### PDF Bank Statements

**Description**: Extracts transactions from PDF bank statements
**Supported Types**:

- Viseca credit card statements (specialized parsing)
- Generic bank statement PDFs

**Requirements**: `pdftotext` must be installed

**Example Usage**:

```bash
./camt-csv pdf -i statement.pdf -o transactions.csv
```

### Revolut CSV Files

**Description**: Processes Revolut app CSV exports
**Features**:

- Transaction state handling
- Fee processing
- Currency conversion tracking
- Category mapping

**Example Usage**:

```bash
./camt-csv revolut -i revolut_export.csv -o processed.csv
```

### Revolut Investment CSV Files

**Description**: Processes Revolut investment transaction CSV exports
**Features**:

- Investment transaction categorization (BUY, DIVIDEND, CASH TOP-UP)
- Share quantity and price tracking
- Multi-currency support with FX rate handling
- Automatic debit/credit classification
- Investment-specific metadata (ticker, fund information)

**Supported Transaction Types**:
- **BUY**: Stock purchases with quantity and price per share
- **DIVIDEND**: Dividend payments from holdings
- **CASH TOP-UP**: Cash deposits to investment account

**Example Usage**:

```bash
./camt-csv revolut-investment -i investment_export.csv -o processed.csv
```

### Selma Investment CSV

**Description**: Processes Selma investment platform exports
**Features**:

- Investment transaction categorization
- Stamp duty association
- Dividend and income tracking
- Trade transaction processing

**Example Usage**:

```bash
./camt-csv selma -i selma_transactions.csv -o processed.csv
```

### Generic Debit CSV

**Description**: Processes generic CSV files with debit transactions
**Features**:

- Flexible column mapping
- Date format detection
- Amount standardization

**Example Usage**:

```bash
./camt-csv debit -i debit_transactions.csv -o processed.csv
```

## Transaction Categorization

### How Categorization Works

CAMT-CSV uses a sophisticated **Strategy Pattern** with four-tier categorization:

1. **Direct Mapping Strategy** (Fastest):
   - Checks `database/creditors.yaml` and `database/debtors.yaml`
   - Exact, case-insensitive matches for known payees/payers
   - Instant recognition for recurring transactions
   - No processing overhead

2. **Keyword Strategy** (Local Processing):
   - Uses pattern matching rules from `database/categories.yaml`
   - Matches against transaction descriptions and party names
   - Configurable keyword patterns and rules
   - No API calls required, fully local processing

3. **Semantic Strategy** (Advanced Matching):
   - Advanced pattern matching using semantic analysis
   - Handles variations in transaction descriptions
   - More intelligent than simple keyword matching
   - Still local processing, no external API calls

4. **AI Strategy** (Optional Fallback):
   - Fallback to Gemini AI when local methods fail
   - Context-aware analysis of transaction details
   - With `--auto-learn`: saves results directly to main YAML files
   - Without `--auto-learn`: saves results to staging files for review
   - Rate limiting to prevent API quota exceeded
   - Lazy initialization for optimal performance

### Strategy Pattern Benefits

- **Independent Testing**: Each strategy can be tested and optimized separately
- **Easy Extension**: New categorization algorithms can be added as strategies
- **Flexible Configuration**: Strategies can be enabled/disabled or reordered
- **Performance Optimization**: Strategies execute in order of efficiency

### Managing Categories

#### View Current Categories

```bash
cat database/categories.yaml
```

#### Add New Category

Edit `database/categories.yaml`:

```yaml
categories:
  - name: "New Category"
    keywords:
      - "keyword1"
      - "keyword2"
```

#### View Learned Mappings

```bash
cat database/creditors.yaml  # For money received
cat database/debtors.yaml    # For money spent (renamed from debitors.yaml)
```

**Migration Note**: The debtor mapping file has been renamed from `debitors.yaml` to `debtors.yaml` for standard English spelling. The application maintains backward compatibility with the old filename, but it's recommended to rename your existing file.

### Categorization Best Practices

1. **Start with Keywords**: Define common patterns in `categories.yaml`
2. **Use AI Sparingly**: Enable AI for unknown transactions only
3. **Review and Clean**: Periodically review learned mappings
4. **Case Sensitivity**: All matching is case-insensitive
5. **Rate Limiting**: Respect API limits with `GEMINI_REQUESTS_PER_MINUTE`

## Troubleshooting

### Common Issues

#### 1. "pdftotext not found"

**Problem**: PDF processing fails
**Solution**: Install Poppler Utils:

```bash
# macOS
brew install poppler

# Ubuntu/Debian
sudo apt-get install poppler-utils
```

#### 2. "Invalid file format"

**Problem**: File not recognized or validation fails
**Solutions**:

- Verify file format matches command (XML for `camt`, PDF for `pdf`, etc.)
- Check file isn't corrupted
- Try with a sample file first
- Look for specific error details in the error message (enhanced error types provide detailed context)
- Check for `ParseError`, `ValidationError`, or `InvalidFormatError` in the output

#### 3. "API quota exceeded"

**Problem**: Too many AI categorization requests
**Solutions**:

- Reduce `GEMINI_REQUESTS_PER_MINUTE`
- Add more keywords to `categories.yaml`
- Process files in smaller batches

#### 4. "Permission denied"

**Problem**: Cannot write output file
**Solutions**:

- Check output directory exists and is writable
- Verify file isn't open in another application
- Use absolute paths if relative paths fail

### Debug Mode

Enable detailed logging for troubleshooting by setting the log level as a CLI flag:

```bash
./camt-csv --log-level debug camt -i input.xml -o output.csv
```

### Understanding Error Messages

CAMT-CSV provides detailed error messages with context to help troubleshoot issues:

#### Parse Errors
```
CAMT: failed to parse amount='invalid': strconv.ParseFloat: parsing "invalid": invalid syntax
```
- **Parser**: Which parser encountered the error
- **Field**: What field failed to parse
- **Value**: The actual value that caused the issue

#### Validation Errors
```
validation failed for /path/to/file.xml: not a valid CAMT.053 XML document
```
- **File Path**: The file that failed validation
- **Reason**: Why validation failed

#### Data Extraction Errors
```
data extraction failed in file '/path/to/file.pdf' for field 'amount': unable to parse currency. Reason: no currency symbol found
```
- **File Path**: The file where extraction failed
- **Field**: Which field couldn't be extracted
- **Reason**: Detailed explanation of the failure

### Logging Configuration

Configure logging output format and level:

```bash
# JSON format for structured logging
./camt-csv --log-format json --log-level info camt -i input.xml -o output.csv

# Text format for human-readable output
./camt-csv --log-format text --log-level debug camt -i input.xml -o output.csv
```

### Getting Help

1.  **Command Help**: `./camt-csv [command] --help`
2.  **General Help**: `./camt-csv --help`
3.  **Version Info**: `./camt-csv --version`

## Examples

### Example 1: Basic CAMT.053 Conversion

```bash
# Convert XML bank statement to CSV
./camt-csv camt -i samples/camt053/statement.xml -o output/transactions.csv

# View the results
head -5 output/transactions.csv
```

### Example 2: Batch Processing with Custom Delimiter

1.  **Set the delimiter in `~/.camt-csv/camt-csv.yaml`**:

    ```yaml
    csv:
      delimiter: ";"
    ```

2.  **Process all files in a directory**:

    ```bash
    ./camt-csv batch -i input_files/ -o output_files/
    ```

### Example 3: AI-Powered Categorization

1.  **Configure AI categorization in `~/.camt-csv/camt-csv.yaml`**:

    ```yaml
    ai:
      enabled: true
    ```

2.  **Set your API key as an environment variable**:

    ```bash
    export GEMINI_API_KEY=your_api_key
    ```

3.  **Process with AI categorization**:

    ```bash
    ./camt-csv revolut -i revolut_export.csv -o categorized.csv
    ```

### Example 4: Custom Categories

1.  **Edit categories file**:

    ```bash
    nano database/categories.yaml
    ```

2.  **Add custom category**:

    ```yaml
    categories:
      - name: "Online Shopping"
        keywords:
          - "amazon"
          - "ebay"
          - "online"
          - "e-commerce"
    ```

3.  **Process transactions**:

    ```bash
    ./camt-csv camt -i statement.xml -o categorized.csv
    ```

### Example 5: Processing Revolut Investment Transactions

```bash
# Process Revolut investment CSV with detailed transaction categorization
./camt-csv revolut-investment -i revolut_investment_export.csv -o investment_transactions.csv

# View the processed investment transactions
head -10 investment_transactions.csv
```

**Sample Input (Revolut Investment CSV)**:
```csv
Date,Ticker,Type,Quantity,Price per share,Total Amount,Currency,FX Rate
2024-01-15T10:30:00.000Z,AAPL,BUY,10,$150.00,$1500.00,USD,1.0
2024-01-20T09:15:00.000Z,AAPL,DIVIDEND,,,$25.50,USD,1.0
2024-01-25T14:45:00.000Z,,CASH TOP-UP,,,$1000.00,USD,1.0
```

### Example 6: Debugging Failed Processing

```bash
# Process with detailed logging enabled via CLI flag
./camt-csv --log-level debug --log-format text pdf -i problematic.pdf -o debug.csv 2>&1 | tee debug.log

# Review debug information
less debug.log
```

---

## Next Steps

- **Explore Samples**: Check the `samples/` directory for example files
- **Customize Categories**: Edit `database/categories.yaml` for your needs
- **Set Up AI**: Configure Gemini API for intelligent categorization
- **Automate Processing**: Create scripts for regular batch processing

For technical details and development information, see the [Codebase Documentation](codebase_documentation.md).
