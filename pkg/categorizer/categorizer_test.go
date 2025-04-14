package categorizer

import (
	"testing"

	"fjacquet/camt-csv/pkg/config"
)

func TestCategorizeTransaction(t *testing.T) {
	// Load environment variables from .env file
	config.LoadEnv()
	
	// Skip test if no API key is set
	apiKey := config.GetGeminiAPIKey()
	if apiKey == "" {
		t.Skip("Skipping test: GEMINI_API_KEY environment variable not set")
	}

	// Test cases
	testCases := []struct {
		name        string
		transaction Transaction
		expectError bool
	}{
		{
			name: "Coffee Shop Transaction",
			transaction: Transaction{
				Payee:  "Starbucks Coffee",
				Amount: "5.75 EUR",
				Date:   "2023-01-01",
				Info:   "Coffee purchase",
			},
			expectError: false,
		},
		{
			name: "Grocery Store Transaction",
			transaction: Transaction{
				Payee:  "Whole Foods Market",
				Amount: "87.32 EUR",
				Date:   "2023-01-02",
				Info:   "Weekly groceries",
			},
			expectError: false,
		},
		{
			name: "Utility Bill Transaction",
			transaction: Transaction{
				Payee:  "Electric Company Ltd",
				Amount: "120.50 EUR",
				Date:   "2023-01-03",
				Info:   "Monthly electricity bill",
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
	// Store original API key
	originalAPIKey := config.GetGeminiAPIKey()
	
	// Temporarily unset API key for this test
	t.Setenv("GEMINI_API_KEY", "")
	defer t.Setenv("GEMINI_API_KEY", originalAPIKey)
	
	transaction := Transaction{
		Payee:  "Test Seller",
		Amount: "10.00 EUR",
		Date:   "2023-01-01",
		Info:   "Test transaction",
	}
	
	category, err := CategorizeTransaction(transaction)
	
	// Should return an error when API key is not set
	if err == nil {
		t.Errorf("Expected error due to missing API key, but got none")
	}
	
	// Should return "Uncategorized" category
	if category.Name != "Uncategorized" {
		t.Errorf("Expected category name 'Uncategorized', got '%s'", category.Name)
	}
	
	// Should have a description about the missing API key
	if category.Description != "No API key provided for categorization" {
		t.Errorf("Expected description about missing API key, got '%s'", category.Description)
	}
}
