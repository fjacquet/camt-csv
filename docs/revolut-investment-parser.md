# Revolut Investment Parser

## Overview

The Revolut Investment Parser is a new feature that extends the CAMT-CSV project to support parsing Revolut investment transaction CSV files. This parser handles the specific format used by Revolut for investment activities, which differs from the standard transaction format.

## Supported File Format

The parser supports Revolut investment CSV files with the following headers:

```csv
Date,Ticker,Type,Quantity,Price per share,Total Amount,Currency,FX Rate
```

### Field Descriptions

| Field | Description |
|-------|-------------|
| Date | Transaction date in ISO format (e.g., 2025-05-30T10:31:02.786456Z) |
| Ticker | Investment ticker symbol (e.g., AAPL, VUAA) |
| Type | Transaction type (e.g., BUY - MARKET, DIVIDEND, CASH TOP-UP) |
| Quantity | Number of shares for buy/sell transactions |
| Price per share | Price per share in the transaction currency |
| Total Amount | Total transaction amount including fees |
| Currency | Transaction currency (e.g., USD, EUR) |
| FX Rate | Foreign exchange rate for currency conversion |

## Transaction Types

The parser recognizes the following transaction types:

1. **BUY - MARKET**: Purchase of investment shares
2. **SELL - MARKET**: Sale of investment shares
3. **DIVIDEND**: Dividend payments received
4. **CASH TOP-UP**: Cash additions to investment account

## Data Mapping

The parser maps Revolut investment CSV fields to the standard transaction model as follows:

| Revolut Investment Field | Standard Transaction Field | Notes |
|--------------------------|----------------------------|-------|
| Date | Date, ValueDate | Parsed from ISO format to DD.MM.YYYY |
| Ticker | Investment, Fund | Investment ticker symbol |
| Type | Type, InvestmentType | BUY/SELL/DIVIDEND |
| Quantity | NumberOfShares | For buy/sell transactions |
| Price per share | AmountExclTax | Price per share |
| Total Amount | Amount | Total transaction amount |
| Currency | Currency, OriginalCurrency | Transaction currency |
| FX Rate | ExchangeRate | Foreign exchange rate |

## Command Line Usage

The Revolut Investment parser is accessible through a new subcommand:

```bash
./camt-csv revolut-investment -i input-file.csv -o output-file.csv
```

### Options

- `-i, --input string`: Path to the input Revolut investment CSV file
- `-o, --output string`: Path to the output standardized CSV file
- `-v, --validate`: Validate the input file format without processing

### Examples

```bash
# Convert a Revolut investment CSV file
./camt-csv revolut-investment -i revolut-investments.csv -o transactions.csv

# Validate a Revolut investment CSV file format
./camt-csv revolut-investment -i revolut-investments.csv -v
```

## Implementation Details

### Package Structure

The feature is implemented in a new package:

```bash
internal/revolutinvestmentparser/
├── adapter.go
├── revolutinvestmentparser.go
└── revolutinvestmentparser_test.go
```

### Interface Compliance

The parser implements the standard `parser.Parser` interface:

```go
type Parser interface {
    ParseFile(filePath string) ([]models.Transaction, error)
    WriteToCSV(transactions []models.Transaction, csvFile string) error
    ValidateFormat(filePath string) (bool, error)
    ConvertToCSV(inputFile, outputFile string) error
    SetLogger(logger *logrus.Logger)
}
```

### Categorization

Investment transactions are categorized using the existing hybrid approach:

1. Direct mapping from YAML databases (`creditors.yaml`, `debitors.yaml`)
2. Keyword matching from `categories.yaml`
3. AI fallback using Google Gemini API (when enabled)

Common investment-related categories include:

- "Investments" for BUY/SELL transactions
- "Dividends" for dividend payments
- "Transfers" for cash top-ups

## Testing

The implementation includes comprehensive tests following the project's testing strategy:

1. Unit tests for parsing functions
2. Integration tests for the full parser workflow
3. Validation tests for different transaction types
4. Edge case tests for malformed data

## Future Enhancements

Potential future enhancements for the Revolut Investment parser:

1. Support for additional transaction types (e.g., SELL, TRANSFER)
2. Enhanced categorization for specific investment types
3. Performance optimizations for large investment portfolios
4. Support for other investment platforms with similar formats
