package debitparser

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/categorizer"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSetLogger(t *testing.T) {
	// Create a test logger
	testLogger := logrus.New()
	testLogger.SetLevel(logrus.DebugLevel)

	// Set the logger
	SetLogger(testLogger)

	// Verify that the package logger was updated
	if log.Level != logrus.DebugLevel {
		t.Error("SetLogger did not correctly update the logger")
	}
}

func TestValidateFormat(t *testing.T) {
	// Create a temporary valid debit CSV file
	validContent := `Bénéficiaire;Date;Montant;Monnaie
PMT CARTE RATP;15.04.2025;-4,21;CHF
PMT CARTE Parking-Relais Lausa;02.04.2025;-4,00;CHF`

	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "valid.csv")
	os.WriteFile(validFile, []byte(validContent), 0644)

	// Create a temporary invalid CSV file
	invalidContent := `SomeHeader1;SomeHeader2
Value1;Value2`

	invalidFile := filepath.Join(tempDir, "invalid.csv")
	os.WriteFile(invalidFile, []byte(invalidContent), 0644)

	// Test validation on valid file
	valid, err := ValidateFormat(validFile)
	if err != nil {
		t.Errorf("ValidateFormat returned an error for a valid file: %v", err)
	}
	if !valid {
		t.Errorf("ValidateFormat returned false for a valid file")
	}

	// Test validation on invalid file
	valid, err = ValidateFormat(invalidFile)
	if err != nil {
		t.Errorf("ValidateFormat returned an error for an invalid file: %v", err)
	}
	if valid {
		t.Errorf("ValidateFormat returned true for an invalid file")
	}
}

func TestParseFile(t *testing.T) {
	// Create a temporary valid debit CSV file
	validContent := `Bénéficiaire;Date;Montant;Monnaie
PMT CARTE RATP;15.04.2025;-4,21;CHF
PMT CARTE Parking-Relais Lausa;02.04.2025;-4,00;CHF
RETRAIT BCV MONTREUX FORUM;28.03.2025;-260,00;CHF`

	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "valid.csv")
	os.WriteFile(validFile, []byte(validContent), 0644)

	// Parse the file
	transactions, err := ParseFile(validFile)
	if err != nil {
		t.Fatalf("ParseFile returned an error: %v", err)
	}

	// Check the number of transactions
	if len(transactions) != 3 {
		t.Errorf("Expected 3 transactions, got %d", len(transactions))
	}

	// Check the first transaction
	assert.Equal(t, "15.04.2025", transactions[0].Date)
	assert.Equal(t, "RATP", transactions[0].Description)
	assert.Equal(t, models.ParseAmount("4.21"), transactions[0].Amount)
	assert.Equal(t, "CHF", transactions[0].Currency)
	assert.Equal(t, "DBIT", transactions[0].CreditDebit)
}

func setupTestCategorizer(t *testing.T) {
	t.Helper()
	tempDir := t.TempDir()
	categoriesFile := filepath.Join(tempDir, "categories.yaml")
	creditorsFile := filepath.Join(tempDir, "creditors.yaml")
	debitorsFile := filepath.Join(tempDir, "debitors.yaml")
	os.WriteFile(categoriesFile, []byte("[]"), 0644)
	os.WriteFile(creditorsFile, []byte("{}"), 0644)
	os.WriteFile(debitorsFile, []byte("{}"), 0644)
	store := store.NewCategoryStore(categoriesFile, creditorsFile, debitorsFile)
	categorizer.SetTestCategoryStore(store)
	t.Cleanup(func() {
		categorizer.SetTestCategoryStore(nil)
	})
}

func TestWriteToCSV(t *testing.T) {
	setupTestCategorizer(t)
	// Create test transactions
	transactions := []struct {
		description string
		date        string
		amount      string
		currency    string
		creditDebit string
	}{
		{"RATP", "15.04.2025", "4.21", "CHF", "DBIT"},
		{"Parking-Relais Lausa", "02.04.2025", "4.00", "CHF", "DBIT"},
	}

	// Convert to models.Transaction
	var modelTransactions []struct {
		Description string
		Date        string
		ValueDate   string
		Amount      string
		Currency    string
		CreditDebit string
		Payee       string
	}
	for _, tx := range transactions {
		modelTransactions = append(modelTransactions, struct {
			Description string
			Date        string
			ValueDate   string
			Amount      string
			Currency    string
			CreditDebit string
			Payee       string
		}{
			Description: tx.description,
			Date:        tx.date,
			ValueDate:   tx.date,
			Amount:      tx.amount,
			Currency:    tx.currency,
			CreditDebit: tx.creditDebit,
			Payee:       tx.description,
		})
	}

	// Create a temporary output file
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.csv")

	// Write transactions to CSV
	err := WriteToCSV(nil, outputFile)
	if err == nil {
		t.Errorf("WriteToCSV should have returned an error for nil transactions")
	}
}

func TestConvertToCSV(t *testing.T) {
	setupTestCategorizer(t)
	// Create a temporary valid debit CSV file
	validContent := `Bénéficiaire;Date;Montant;Monnaie
PMT CARTE RATP;15.04.2025;-4,21;CHF
PMT CARTE Parking-Relais Lausa;02.04.2025;-4,00;CHF`

	tempDir := t.TempDir()
	inputFile := filepath.Join(tempDir, "input.csv")
	os.WriteFile(inputFile, []byte(validContent), 0644)

	outputFile := filepath.Join(tempDir, "output.csv")

	// Convert to CSV
	err := ConvertToCSV(inputFile, outputFile)
	if err != nil {
		t.Fatalf("ConvertToCSV returned an error: %v", err)
	}

	// Check if output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Output file was not created")
	}

	// Test with invalid input file
	err = ConvertToCSV("nonexistent.csv", outputFile)
	if err == nil {
		t.Errorf("ConvertToCSV should have returned an error for a nonexistent file")
	}
}
