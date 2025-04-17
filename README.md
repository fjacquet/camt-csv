# camt-csv
Convert file from CAMT053 to csv with transaction categorisation using AI

## Features

- Convert CAMT.053 XML files to CSV format with enhanced field extraction
- Categorize transactions using a hybrid approach:
  - Local keyword matching based on a customizable YAML configuration
  - Fallback to Gemini-2.0-fast model when local matching fails
- Clean CLI interface using Cobra
- Detailed logging with Logrus
- Convert PDF files to CSV format
- Batch processing for multiple files

## Installation

### Prerequisites

- Go 1.24 or higher
- pdftotext CLI tool (from Poppler Utils)
  - On macOS: `brew install poppler`
  - On Ubuntu/Debian: `apt-get install poppler-utils`
  - On Windows: Download binaries from [Poppler for Windows](http://blog.alivate.com.au/poppler-windows/)

### Building from source

```bash
git clone https://github.com/fjacquet/camt-csv.git
cd camt-csv
go build -o camt-csv ./cmd/camt-csv
```

## Usage

### Convert CAMT.053 XML to CSV

```bash
./camt-csv convert -i input.xml -o output.csv
```

### Convert viseca PDF to CSV

```bash
./camt-csv pdf -i input.pdf -o output.csv
```

### Batch Convert Multiple XML Files

```bash
./camt-csv batch -i input_directory -o output_directory
```

### Validate XML Format

```bash
./camt-csv validate -i input.xml
```

### Categorize transactions

```bash
./camt-csv categorize -s "Seller Name" -a "100.00 EUR" -d "2023-01-01" -i "Additional info"
```

## Project Structure

```
camt-csv/
├── cmd/
│   └── camt-csv/       # CLI application entry point
├── pkg/
│   ├── converter/      # XML to CSV conversion logic
│   ├── categorizer/    # Transaction categorization logic
│   ├── pdfparser/      # PDF to CSV conversion logic
│   └── config/         # Environment configuration
├── database/           # Configuration data files
│   └── categories.yaml # Transaction categorization rules
└── samples/            # Sample files for testing
    ├── camt053/        # Sample CAMT.053 XML files
    ├── csv/            # Output CSV files
    └── pdf/            # Sample PDF files
```

## CAMT.053 Format

CAMT.053 (Cash Management) is a standard XML format used by banks for account statements. It follows the ISO 20022 standard and contains detailed information about account transactions.

## Transaction Categorization

The application uses a hybrid approach to categorize transactions:

1. **Local Keyword Matching**: First, it attempts to match transaction descriptions and seller names against keywords defined in the `database/categories.yaml` file. This is fast and doesn't require API calls.

2. **AI-Based Categorization**: If no keyword match is found, the application falls back to using the Gemini-2.0-fast model to analyze the transaction details and assign a category.

### Customizing Categories

You can customize transaction categories by editing the `database/categories.yaml` file. The file has the following structure:

```yaml
categories:
  - name: "Category Name"
    keywords:
      - "keyword1"
      - "keyword2"
      - "keyword3"
  
  - name: "Another Category"
    keywords:
      - "keyword4"
      - "keyword5"
```

To add new categories or modify existing ones:

1. Edit the `database/categories.yaml` file
2. Add a new entry under the `categories` list or modify an existing entry
3. Restart the application for changes to take effect

This approach significantly reduces API calls for recurring transaction types, making the categorization process faster and more efficient.
