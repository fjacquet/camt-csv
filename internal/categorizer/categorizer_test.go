package categorizer

import (
	"testing"
	"os"

	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/store"
)

func TestCategorizeTransaction(t *testing.T) {
	// Load environment variables from .env file
	config.LoadEnv()

	// Set test mode to disable API calls
	os.Setenv("TEST_MODE", "true")
	defer os.Unsetenv("TEST_MODE")
	
	// Create a mock store for testing
	mockStore := &store.CategoryStore{
		CategoriesFile: "testdata/categories.yaml",
		CreditorsFile:  "testdata/creditors.yaml",
		DebitorsFile:   "testdata/debitors.yaml",
	}
	
	// Set up the test categorizer
	SetTestCategoryStore(mockStore)

	// Test cases
	testCases := []struct {
		name        string
		transaction Transaction
		expectError bool
	}{
		{
			name: "Coffee Shop Transaction (Creditor)",
			transaction: Transaction{
				PartyName: "Starbucks Coffee",
				IsDebtor:  false,
				Amount:    "5.75 EUR",
				Date:      "2023-01-01",
				Info:      "Coffee purchase",
			},
			expectError: false,
		},
		{
			name: "Grocery Store Transaction (Creditor)",
			transaction: Transaction{
				PartyName: "Whole Foods Market",
				IsDebtor:  false,
				Amount:    "87.32 EUR",
				Date:      "2023-01-02",
				Info:      "Weekly groceries",
			},
			expectError: false,
		},
		{
			name: "Unknown Transaction",
			transaction: Transaction{
				PartyName: "Unknown Merchant",
				IsDebtor:  false,
				Amount:    "10.00 EUR",
				Date:      "2023-01-05", 
				Info:      "Unknown purchase",
			},
			expectError: false,
		},
		{
			name: "Salary Transaction (Debtor)",
			transaction: Transaction{
				PartyName: "Acme Corp",
				IsDebtor:  true,
				Amount:    "3000.00 EUR",
				Date:      "2023-01-15",
				Info:      "Monthly salary",
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use the global categorizer since we've set it up with the test store
			category, err := CategorizeTransaction(tc.transaction)

			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Did not expect error but got: %v", err)
			}
			if err == nil {
				t.Logf("Category: %s", category.Name)
			}
		})
	}
}

// Simplified test for categorization with no API key
func TestCategorizeTransactionNoAPIKey(t *testing.T) {
	// Force test mode to prevent any API calls
	os.Setenv("TEST_MODE", "true")
	defer os.Unsetenv("TEST_MODE")
	
	// Create a test transaction
	transaction := Transaction{
		PartyName: "New Merchant",
		IsDebtor:  false,
		Amount:    "15.00 EUR",
		Date:      "2023-01-20",
		Info:      "Test transaction",
	}
	
	// Use the global categorizer
	category, err := CategorizeTransaction(transaction)
	
	// Validation
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Should fall back to "Uncategorized" category in test mode
	if category.Name != "Uncategorized" {
		t.Errorf("Expected 'Uncategorized' category but got '%s'", category.Name)
	}
	
	t.Logf("Category: %s, Description: %s", category.Name, category.Description)
}
