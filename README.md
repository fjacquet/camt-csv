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
- Process Selma investment CSV files with intelligent categorization

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

## Configuration

The application can be configured using environment variables or an `.env` file in the project root. 
A sample configuration file `.env.sample` is provided as a template.

### Environment Variables

| Variable | Description | Default | Available Options |
|----------|-------------|---------|-------------------|
| GOOGLE_API_KEY | API key for Gemini AI (transaction categorization) | - | - |
| LOG_LEVEL | Controls the verbosity of logging | `info` | `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic` |
| LOG_FORMAT | Format of the log output | `text` | `text`, `json` |
| DATA_DIR | Directory for configuration files | `.` | Any valid directory path |

For example:

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Use JSON log format (useful for log aggregation)
export LOG_FORMAT=json

# Run the application
./camt-csv convert -i input.xml -o output.csv
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

### Process Selma CSV Files

```bash
./camt-csv selma -i input_selma.csv -o processed_output.csv
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
│   └── camt-csv/        # CLI application entry point
├── internal/            # Application-specific packages
│   ├── categorizer/     # Transaction categorization logic
│   ├── config/          # Environment configuration
│   ├── converter/       # XML to CSV conversion logic
│   ├── models/          # Data models used internally
│   ├── pdfparser/       # PDF to CSV conversion logic
│   └── selmaparser/     # Selma CSV processing logic
├── database/            # Configuration data files
│   └── categories.yaml  # Transaction categorization rules
└── samples/             # Sample files for testing
    ├── camt053/         # Sample CAMT.053 XML files
    ├── csv/             # Output CSV files
    └── pdf/             # Sample PDF files
```

According to Go best practices, all application-specific code is placed in the `internal/` directory, ensuring it cannot be imported by other projects. This follows the principle of encapsulation and helps maintain clear boundaries in the codebase.

## Standardized Parser Architecture

The application has been designed with a standardized parser architecture across all data source types. Each parser package follows the same interface pattern, making it easy to maintain and extend with new data sources.

### Parser Interface

All parsers implement the following standard functions:

| Function | Description |
|----------|-------------|
| `ParseFile(filePath string) ([]models.Transaction, error)` | Parses a source file and extracts transactions |
| `WriteToCSV(transactions []models.Transaction, csvFile string) error` | Writes a slice of transactions to a CSV file |
| `ValidateFormat(filePath string) (bool, error)` | Validates that a file is in the expected format |
| `ConvertToCSV(inputFile, outputFile string) error` | Convenience method to parse and write in one operation |
| `SetLogger(logger *logrus.Logger)` | Sets a configured logger for the parser |

### Available Parsers

1. **camtparser**: Parses CAMT.053 XML bank statement files
   ```go
   transactions, err := camtparser.ParseFile("statement.xml")
   ```

2. **pdfparser**: Extracts transaction data from PDF bank statements
   ```go
   transactions, err := pdfparser.ParseFile("statement.pdf")
   ```

3. **selmaparser**: Processes Selma investment CSV files
   ```go
   transactions, err := selmaparser.ParseFile("investments.csv")
   err = selmaparser.WriteToCSV(transactions, "output.csv")
   ```

### Internal Architecture

Each parser package follows a consistent structure:
- `xxxparser.go`: Contains the public API and main functions
- `xxxparser_helpers.go`: Contains internal implementation details and helper functions

This separation improves code organization by clearly distinguishing between the public interface and the implementation details.

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

## Selma CSV Processing

The Selma CSV processor is designed to work with investment transaction data from Selma, enhancing it with:

1. **Transaction Categorization**: Automatically identifies and categorizes different types of investment transactions:
   - Initial categorization based on transaction type (Dividend, Income, etc.)
   - Advanced AI-based categorization for unrecognized transactions
   - Integration with the same categorization engine used for CAMT transactions

2. **Stamp Duty Handling**: Associates stamp duty amounts with their corresponding trade transactions.

3. **Data Cleaning**: Filters out redundant entries and improves data organization.

### Sample Selma CSV Command

Process a Selma investment CSV file and output the categorized transactions:

```bash
# Basic processing
./camt-csv selma -i selma_transactions.csv -o processed_selma.csv

# View the processed output
cat processed_selma.csv
```

The processed output will include additional fields for categorization and associated stamp duty amounts.
