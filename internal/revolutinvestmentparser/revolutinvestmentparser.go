package revolutinvestmentparser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"fjacquet/camt-csv/internal/models"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// RevolutInvestmentCSVRow represents a single row in a Revolut investment CSV file
type RevolutInvestmentCSVRow struct {
	Date          string `csv:"Date"`
	Ticker        string `csv:"Ticker"`
	Type          string `csv:"Type"`
	Quantity      string `csv:"Quantity"`
	PricePerShare string `csv:"Price per share"`
	TotalAmount   string `csv:"Total Amount"`
	Currency      string `csv:"Currency"`
	FXRate        string `csv:"FX Rate"`
}

var logger *logrus.Logger

func init() {
	logger = logrus.New()
}

// SetLogger sets the logger for the Revolut investment parser
func SetLogger(l *logrus.Logger) {
	logger = l
}

// ParseFile parses a Revolut investment CSV file and returns a slice of transactions
func ParseFile(filePath string) ([]models.Transaction, error) {
	logger.Infof("Parsing Revolut investment CSV file: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty or contains only headers")
	}

	// Validate headers
	expectedHeaders := []string{"Date", "Ticker", "Type", "Quantity", "Price per share", "Total Amount", "Currency", "FX Rate"}
	if len(records[0]) < len(expectedHeaders) {
		return nil, fmt.Errorf("CSV file has insufficient columns")
	}

	for i, header := range expectedHeaders {
		if strings.TrimSpace(records[0][i]) != header {
			return nil, fmt.Errorf("unexpected header at position %d: expected '%s', got '%s'", i, header, strings.TrimSpace(records[0][i]))
		}
	}

	var transactions []models.Transaction

	// Process each row (skip header)
	for i, record := range records[1:] {
		if len(record) < 8 {
			logger.Warnf("Skipping row %d: insufficient columns", i+2)
			continue
		}

		row := RevolutInvestmentCSVRow{
			Date:          record[0],
			Ticker:        record[1],
			Type:          record[2],
			Quantity:      record[3],
			PricePerShare: record[4],
			TotalAmount:   record[5],
			Currency:      record[6],
			FXRate:        record[7],
		}

		transaction, err := convertRowToTransaction(row)
		if err != nil {
			logger.Warnf("Failed to convert row %d to transaction: %v", i+2, err)
			continue
		}

		transactions = append(transactions, transaction)
	}

	logger.Infof("Successfully parsed %d transactions from Revolut investment CSV", len(transactions))
	return transactions, nil
}

// convertRowToTransaction converts a RevolutInvestmentCSVRow to a models.Transaction
func convertRowToTransaction(row RevolutInvestmentCSVRow) (models.Transaction, error) {
	var transaction models.Transaction

	// Parse date
	transaction.Date = formatDate(row.Date)
	transaction.ValueDate = transaction.Date

	// Set investment details
	transaction.Investment = row.Ticker
	transaction.Fund = row.Ticker
	transaction.Type = row.Type

	// Parse currency
	transaction.Currency = row.Currency
	transaction.OriginalCurrency = row.Currency

	// Parse FX rate
	if row.FXRate != "" {
		if fxRate, err := decimal.NewFromString(row.FXRate); err == nil {
			transaction.ExchangeRate = fxRate
		} else {
			logger.Warnf("Failed to parse FX rate '%s': %v", row.FXRate, err)
		}
	}

	// Handle different transaction types
	logger.Debugf("Processing transaction type: '%s'", row.Type)
	switch {
	case strings.Contains(row.Type, "BUY"):
		logger.Debugf("Processing BUY transaction")
		// Parse quantity
		if row.Quantity != "" {
			if quantity, err := decimal.NewFromString(row.Quantity); err == nil {
				transaction.NumberOfShares = int(quantity.IntPart())
			} else {
				logger.Warnf("Failed to parse quantity '%s': %v", row.Quantity, err)
			}
		}

		// Parse price per share
		if row.PricePerShare != "" {
			priceStr := strings.TrimPrefix(row.PricePerShare, "€")
			priceStr = strings.TrimPrefix(priceStr, "$")
			priceStr = strings.TrimPrefix(priceStr, "£")
			priceStr = strings.ReplaceAll(priceStr, ",", "")
			if price, err := decimal.NewFromString(priceStr); err == nil {
				transaction.AmountExclTax = price
			} else {
				logger.Warnf("Failed to parse price per share '%s': %v", row.PricePerShare, err)
			}
		}

		// Parse total amount
		if row.TotalAmount != "" {
			amountStr := strings.TrimPrefix(row.TotalAmount, "€")
			amountStr = strings.TrimPrefix(amountStr, "$")
			amountStr = strings.TrimPrefix(amountStr, "£")
			amountStr = strings.ReplaceAll(amountStr, ",", "")
			if amount, err := decimal.NewFromString(amountStr); err == nil {
				transaction.Amount = amount
				// For buy transactions, this is typically a debit
				transaction.DebitFlag = true
				transaction.CreditDebit = "DBIT"
				logger.Debugf("Set CreditDebit to DBIT for BUY transaction")
			} else {
				logger.Warnf("Failed to parse total amount '%s': %v", row.TotalAmount, err)
			}
		}

		transaction.Description = fmt.Sprintf("Buy %s shares of %s", row.Quantity, row.Ticker)

	case strings.Contains(row.Type, "DIVIDEND"):
		logger.Debugf("Processing DIVIDEND transaction")
		// Parse dividend amount
		if row.TotalAmount != "" {
			amountStr := strings.TrimPrefix(row.TotalAmount, "€")
			amountStr = strings.TrimPrefix(amountStr, "$")
			amountStr = strings.TrimPrefix(amountStr, "£")
			amountStr = strings.ReplaceAll(amountStr, ",", "")
			if amount, err := decimal.NewFromString(amountStr); err == nil {
				transaction.Amount = amount
				// Dividends are typically credits
				transaction.DebitFlag = false
				transaction.CreditDebit = "CRDT"
				transaction.Credit = amount
				logger.Debugf("Set CreditDebit to CRDT for DIVIDEND transaction")
			} else {
				logger.Warnf("Failed to parse dividend amount '%s': %v", row.TotalAmount, err)
			}
		}

		transaction.Description = fmt.Sprintf("Dividend from %s", row.Ticker)

	case strings.Contains(row.Type, "CASH TOP-UP"):
		logger.Debugf("Processing CASH TOP-UP transaction")
		// Parse cash top-up amount
		if row.TotalAmount != "" {
			amountStr := strings.TrimPrefix(row.TotalAmount, "€")
			amountStr = strings.TrimPrefix(amountStr, "$")
			amountStr = strings.TrimPrefix(amountStr, "£")
			amountStr = strings.ReplaceAll(amountStr, ",", "")
			if amount, err := decimal.NewFromString(amountStr); err == nil {
				transaction.Amount = amount
				// Cash top-ups are typically credits
				transaction.DebitFlag = false
				transaction.CreditDebit = "CRDT"
				transaction.Credit = amount
				logger.Debugf("Set CreditDebit to CRDT for CASH TOP-UP transaction")
			} else {
				logger.Warnf("Failed to parse cash top-up amount '%s': %v", row.TotalAmount, err)
			}
		}

		transaction.Description = "Cash top-up to investment account"

	default:
		logger.Debugf("Processing default transaction")
		// Handle other transaction types
		if row.TotalAmount != "" {
			amountStr := strings.TrimPrefix(row.TotalAmount, "€")
			amountStr = strings.TrimPrefix(amountStr, "$")
			amountStr = strings.TrimPrefix(amountStr, "£")
			amountStr = strings.ReplaceAll(amountStr, ",", "")
			if amount, err := decimal.NewFromString(amountStr); err == nil {
				transaction.Amount = amount
				// Default to debit for unknown transaction types
				transaction.DebitFlag = true
				transaction.CreditDebit = "DBIT"
				logger.Debugf("Set CreditDebit to DBIT for default transaction")
			} else {
				logger.Warnf("Failed to parse amount '%s': %v", row.TotalAmount, err)
			}
		}
		transaction.Description = fmt.Sprintf("%s transaction for %s", row.Type, row.Ticker)
	}

	// Set party name
	if row.Ticker != "" {
		transaction.PartyName = fmt.Sprintf("Revolut Investment - %s", row.Ticker)
	} else {
		transaction.PartyName = "Revolut Investment"
	}

	// Set name
	transaction.Name = transaction.PartyName

	// Update derived fields
	transaction.UpdateNameFromParties()
	transaction.UpdateDebitCreditAmounts()

	return transaction, nil
}

// formatDate standardizes the date format
func formatDate(dateStr string) string {
	// Parse ISO format date
	if t, err := time.Parse(time.RFC3339Nano, dateStr); err == nil {
		return t.Format("02.01.2006") // Return as DD.MM.YYYY
	}

	// If parsing fails, return original string
	return dateStr
}

// ValidateFormat validates if the given file is a valid Revolut investment CSV file
func ValidateFormat(filePath string) (bool, error) {
	logger.Infof("Validating Revolut investment CSV file format: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return false, fmt.Errorf("CSV file is empty")
		}
		return false, fmt.Errorf("failed to read CSV header: %w", err)
	}

	if len(records) < 8 {
		return false, fmt.Errorf("CSV file has insufficient columns")
	}

	expectedHeaders := []string{"Date", "Ticker", "Type", "Quantity", "Price per share", "Total Amount", "Currency", "FX Rate"}
	for i, header := range expectedHeaders {
		if strings.TrimSpace(records[i]) != header {
			return false, fmt.Errorf("unexpected header at position %d: expected '%s', got '%s'", i, header, strings.TrimSpace(records[i]))
		}
	}

	logger.Info("File format validation successful")
	return true, nil
}

// WriteToCSV writes transactions to a CSV file
func WriteToCSV(transactions []models.Transaction, csvFile string) error {
	logger.Infof("Writing %d transactions to CSV file: %s", len(transactions), csvFile)

	file, err := os.Create(csvFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"BookkeepingNumber", "Status", "Date", "ValueDate", "Name", "Description", "RemittanceInfo", "PartyName", "PartyIBAN",
		"Amount", "CreditDebit", "IsDebit", "Debit", "Credit", "Currency", "AmountExclTax", "AmountTax", "TaxRate",
		"Recipient", "InvestmentType", "Number", "Category", "Type", "Fund", "NumberOfShares", "Fees", "IBAN",
		"EntryReference", "Reference", "AccountServicer", "BankTxCode", "OriginalCurrency", "OriginalAmount", "ExchangeRate",
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write transactions
	for _, transaction := range transactions {
		record, err := transaction.MarshalCSV()
		if err != nil {
			logger.Warnf("Failed to marshal transaction: %v", err)
			continue
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write transaction: %w", err)
		}
	}

	logger.Info("Successfully wrote transactions to CSV file")
	return nil
}

// ConvertToCSV converts a Revolut investment CSV file to the standardized format
func ConvertToCSV(inputFile, outputFile string) error {
	logger.Infof("Converting Revolut investment CSV file from '%s' to '%s'", inputFile, outputFile)

	transactions, err := ParseFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to parse input file: %w", err)
	}

	if err := WriteToCSV(transactions, outputFile); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	logger.Info("Successfully converted Revolut investment CSV file")
	return nil
}
