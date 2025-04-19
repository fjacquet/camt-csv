package categorizer

import (
	"testing"

	"fjacquet/camt-csv/internal/config"
)

func TestCategorizeTransaction(t *testing.T) {
	// Load environment variables from .env file
	config.LoadEnv()

	// Skip test if no API key is set and AI categorization is enabled
	if isAICategorizeEnabled() {
		apiKey := getGeminiAPIKey()
		if apiKey == "" {
			t.Skip("Skipping test: GEMINI_API_KEY environment variable not set while USE_AI_CATEGORIZATION=true")
		}
	}

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
			name: "Case Insensitive Match Test - Lowercase",
			transaction: Transaction{
				PartyName: "starbucks coffee", // lowercase version
				IsDebtor:  false,
				Amount:    "5.75 EUR",
				Date:      "2023-01-01",
				Info:      "Coffee purchase with lowercase name",
			},
			expectError: false,
		},
		{
			name: "Case Insensitive Match Test - Mixed Case",
			transaction: Transaction{
				PartyName: "StArBuCkS CoFfEe", // mixed case version
				IsDebtor:  false,
				Amount:    "5.75 EUR",
				Date:      "2023-01-01",
				Info:      "Coffee purchase with mixed case name",
			},
			expectError: false,
		},
		{
			name: "Utility Bill Transaction (Creditor)",
			transaction: Transaction{
				PartyName: "Electric Company Ltd",
				IsDebtor:  false,
				Amount:    "120.50 EUR",
				Date:      "2023-01-03",
				Info:      "Monthly electricity bill",
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
			category, err := CategorizeTransaction(tc.transaction)

			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if err == nil {
				// Verify that we got a category name
				if category.Name == "" {
					t.Errorf("Expected non-empty category name")
				}

				// Verify that the category is one of our predefined categories
				found := false
				for _, validCategory := range GetCategories() {
					if category.Name == validCategory {
						found = true
						break
					}
				}

				if !found && category.Name != "Uncategorized" {
					t.Errorf("Category '%s' is not in the predefined list", category.Name)
				}

				// Verify that we got a description
				if category.Description == "" {
					t.Errorf("Expected non-empty category description")
				}
			}
		})
	}
}

func TestCategorizeTransactionNoAPIKey(t *testing.T) {
	// Store original values
	originalAIEnabled := isAICategorizeEnabled()

	// Force AI categorization to be enabled for this test
	t.Setenv("USE_AI_CATEGORIZATION", "true")
	// Temporarily unset API key for this test
	t.Setenv("GEMINI_API_KEY", "")

	transaction := Transaction{
		PartyName: "Test Party",
		IsDebtor:  false,
		Amount:    "10.00 EUR",
		Date:      "2023-01-01",
		Info:      "Test transaction",
	}

	category, err := CategorizeTransaction(transaction)

	// With our new error handling, we shouldn't get an error anymore
	if err != nil {
		t.Errorf("Expected no error with missing API key, but got: %v", err)
	}

	// Should return "Miscellaneous" category when API categorization fails
	if category.Name != "Uncategorized" {
		t.Errorf("Expected category name 'Miscellaneous', got '%s'", category.Name)
	}

	// Should have a generic description for uncategorized transactions
	if category.Description != "Uncategorized transaction" {
		t.Errorf("Expected description 'Uncategorized transaction', got '%s'", category.Description)
	}

	// Reset environment for other tests
	if originalAIEnabled {
		t.Setenv("USE_AI_CATEGORIZATION", "true")
	} else {
		t.Setenv("USE_AI_CATEGORIZATION", "false")
	}
}
