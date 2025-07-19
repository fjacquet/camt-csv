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

- **Multi-format Support**: CAMT.053 XML, PDF bank statements, Revolut CSV, Selma investment CSV, and generic debit CSV
- **Intelligent Categorization**: Hybrid approach using local keyword matching and AI fallback
- **Batch Processing**: Process multiple files at once
- **Configurable Output**: Customizable CSV delimiters and formats
- **Extensible Architecture**: Easy to add new file formats

## Installation

### Prerequisites

Before installing CAMT-CSV, ensure you have:

- **Go 1.24 or higher**: [Download Go](https://golang.org/dl/)
- **pdftotext CLI tool** (for PDF processing):
  - **macOS**: `brew install poppler`
  - **Ubuntu/Debian**: `apt-get install poppler-utils`
  - **Windows**: [Download Poppler for Windows](http://blog.alivate.com.au/poppler-windows/)

### Building from Source

```bash
git clone https://github.com/fjacquet/camt-csv.git
cd camt-csv
go build
```

This creates a `camt-csv` executable in your project directory.

### Verify Installation

```bash
./camt-csv --help
```

You should see the main help menu with available commands.

## Configuration

CAMT-CSV uses environment variables for configuration. You can set these in your shell or use a `.env` file in the project root.

### Setting Up Configuration

1. Copy the sample configuration file:
   ```bash
   cp .env.sample .env
   ```

2. Edit `.env` with your preferred settings:
   ```bash
   nano .env  # or your preferred editor
   ```

### Configuration Options

| Variable | Description | Default | Options |
|----------|-------------|---------|---------|
| `GEMINI_API_KEY` | API key for Gemini AI categorization | - | Your Google AI API key |
| `GEMINI_MODEL` | Gemini model for categorization | `gemini-2.0-flash` | Any valid Gemini model |
| `GEMINI_REQUESTS_PER_MINUTE` | Rate limit for AI API calls | `10` | Positive integer |
| `USE_AI_CATEGORIZATION` | Enable/disable AI categorization | `false` | `true`, `false` |
| `LOG_LEVEL` | Logging verbosity | `info` | `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic` |
| `LOG_FORMAT` | Log output format | `text` | `text`, `json` |
| `DATA_DIR` | Custom data directory path | - | Any valid directory path |
| `CSV_DELIMITER` | CSV output delimiter | `,` | Any single character (e.g., `;`) |

### Example Configuration

```bash
# .env file example
GEMINI_API_KEY=your_api_key_here
GEMINI_MODEL=gemini-2.0-flash
GEMINI_REQUESTS_PER_MINUTE=15
USE_AI_CATEGORIZATION=true
LOG_LEVEL=info
LOG_FORMAT=text
CSV_DELIMITER=;
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

1. Get a Google AI API key from [Google AI Studio](https://makersuite.google.com/app/apikey)
2. Set your API key:
   ```bash
   export GEMINI_API_KEY=your_api_key_here
   ```
3. Enable AI categorization:
   ```bash
   export USE_AI_CATEGORIZATION=true
   ```

### Custom Output Formats

#### Change CSV Delimiter

For European Excel compatibility:
```bash
export CSV_DELIMITER=";"
./camt-csv camt -i input.xml -o output.csv
```

#### Custom Data Directory

Store configuration files in a custom location:
```bash
export DATA_DIR="/path/to/custom/data"
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

1. **Direct Mapping Check**: 
   - Checks `database/creditors.yaml` and `database/debitors.yaml`
   - Exact, case-insensitive matches
   - Fastest method for known transactions

2. **Keyword Matching**:
   - Uses rules from `database/categories.yaml`
   - Matches against transaction descriptions and party names
   - No API calls required

3. **AI Categorization**:
   - Fallback to Gemini AI when local methods fail
   - Analyzes transaction context
   - Automatically learns new patterns

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
cat database/debitors.yaml   # For money spent
```

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
**Problem**: File not recognized
**Solutions**:
- Verify file format matches command (XML for `camt`, PDF for `pdf`, etc.)
- Check file isn't corrupted
- Try with a sample file first

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

Enable detailed logging for troubleshooting:
```bash
export LOG_LEVEL=debug
./camt-csv camt -i input.xml -o output.csv
```

### Getting Help

1. **Command Help**: `./camt-csv [command] --help`
2. **General Help**: `./camt-csv --help`
3. **Version Info**: `./camt-csv version`

## Examples

### Example 1: Basic CAMT.053 Conversion

```bash
# Convert XML bank statement to CSV
./camt-csv camt -i samples/camt053/statement.xml -o output/transactions.csv

# View the results
head -5 output/transactions.csv
```

### Example 2: Batch Processing with Custom Delimiter

```bash
# Set semicolon delimiter for European Excel
export CSV_DELIMITER=";"

# Process all files in directory
./camt-csv batch -i input_files/ -o output_files/

# Check results
ls output_files/
```

### Example 3: AI-Powered Categorization

```bash
# Configure AI categorization
export GEMINI_API_KEY=your_api_key
export USE_AI_CATEGORIZATION=true
export GEMINI_REQUESTS_PER_MINUTE=5

# Process with AI categorization
./camt-csv revolut -i revolut_export.csv -o categorized.csv

# Check learned patterns
cat database/creditors.yaml
```

### Example 4: Custom Categories

1. **Edit categories file**:
   ```bash
   nano database/categories.yaml
   ```

2. **Add custom category**:
   ```yaml
   categories:
     - name: "Online Shopping"
       keywords:
         - "amazon"
         - "ebay"
         - "online"
         - "e-commerce"
   ```

3. **Process transactions**:
   ```bash
   ./camt-csv camt -i statement.xml -o categorized.csv
   ```

### Example 5: Debugging Failed Processing

```bash
# Enable debug logging
export LOG_LEVEL=debug
export LOG_FORMAT=text

# Process with detailed output
./camt-csv pdf -i problematic.pdf -o debug.csv 2>&1 | tee debug.log

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
