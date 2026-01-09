package categorizer

import (
	"context"
	"sync"
	"testing"
	"time"

	"fjacquet/camt-csv/internal/logging"
	"fjacquet/camt-csv/internal/models"
	"fjacquet/camt-csv/internal/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectMappingStrategy_Name(t *testing.T) {
	strategy := &DirectMappingStrategy{}
	assert.Equal(t, "DirectMapping", strategy.Name())
}

func TestDirectMappingStrategy_Categorize(t *testing.T) {
	tests := []struct {
		name             string
		transaction      Transaction
		creditorMappings map[string]string
		debtorMappings   map[string]string
		expectedCategory string
		expectedFound    bool
		expectedError    bool
	}{
		{
			name: "creditor mapping found",
			transaction: Transaction{
				PartyName: "COOP",
				IsDebtor:  false,
			},
			creditorMappings: map[string]string{
				"coop": models.CategoryGroceries,
			},
			debtorMappings:   map[string]string{},
			expectedCategory: models.CategoryGroceries,
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "debtor mapping found",
			transaction: Transaction{
				PartyName: "John Doe",
				IsDebtor:  true,
			},
			creditorMappings: map[string]string{},
			debtorMappings: map[string]string{
				"john doe": models.CategorySalary,
			},
			expectedCategory: models.CategorySalary,
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "case insensitive matching",
			transaction: Transaction{
				PartyName: "MIGROS",
				IsDebtor:  false,
			},
			creditorMappings: map[string]string{
				"migros": models.CategoryGroceries,
			},
			debtorMappings:   map[string]string{},
			expectedCategory: models.CategoryGroceries,
			expectedFound:    true,
			expectedError:    false,
		},
		{
			name: "no mapping found for creditor",
			transaction: Transaction{
				PartyName: "Unknown Store",
				IsDebtor:  false,
			},
			creditorMappings: map[string]string{
				"coop": models.CategoryGroceries,
			},
			debtorMappings: map[string]string{},
			expectedFound:  false,
			expectedError:  false,
		},
		{
			name: "no mapping found for debtor",
			transaction: Transaction{
				PartyName: "Unknown Person",
				IsDebtor:  true,
			},
			creditorMappings: map[string]string{},
			debtorMappings: map[string]string{
				"john doe": models.CategorySalary,
			},
			expectedFound: false,
			expectedError: false,
		},
		{
			name: "empty party name",
			transaction: Transaction{
				PartyName: "",
				IsDebtor:  false,
			},
			creditorMappings: map[string]string{
				"coop": models.CategoryGroceries,
			},
			debtorMappings: map[string]string{},
			expectedFound:  false,
			expectedError:  false,
		},
		{
			name: "whitespace only party name",
			transaction: Transaction{
				PartyName: "   ",
				IsDebtor:  false,
			},
			creditorMappings: map[string]string{
				"coop": models.CategoryGroceries,
			},
			debtorMappings: map[string]string{},
			expectedFound:  false,
			expectedError:  false,
		},
		{
			name: "wrong mapping type - debtor transaction with creditor mapping",
			transaction: Transaction{
				PartyName: "COOP",
				IsDebtor:  true, // This is a debtor transaction
			},
			creditorMappings: map[string]string{
				"coop": models.CategoryGroceries, // But mapping is in creditor
			},
			debtorMappings: map[string]string{},
			expectedFound:  false, // Should not find it
			expectedError:  false,
		},
		{
			name: "wrong mapping type - creditor transaction with debtor mapping",
			transaction: Transaction{
				PartyName: "John Doe",
				IsDebtor:  false, // This is a creditor transaction
			},
			creditorMappings: map[string]string{},
			debtorMappings: map[string]string{
				"john doe": models.CategorySalary, // But mapping is in debtor
			},
			expectedFound: false, // Should not find it
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock store
			mockStore := &store.MockCategoryStore{
				CreditorMappings: tt.creditorMappings,
				DebtorMappings:   tt.debtorMappings,
			}

			// Create mock logger
			mockLogger := &logging.MockLogger{}

			// Create strategy
			strategy := NewDirectMappingStrategy(mockStore, mockLogger)

			// Execute
			ctx := context.Background()
			category, found, err := strategy.Categorize(ctx, tt.transaction)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedFound, found)

			if tt.expectedFound {
				assert.Equal(t, tt.expectedCategory, category.Name)
				assert.NotEmpty(t, category.Description)
			}
		})
	}
}

func TestDirectMappingStrategy_UpdateMappings(t *testing.T) {
	// Create mock store
	mockStore := &store.MockCategoryStore{
		CreditorMappings: map[string]string{},
		DebtorMappings:   map[string]string{},
	}

	// Create mock logger
	mockLogger := &logging.MockLogger{}

	// Create strategy
	strategy := NewDirectMappingStrategy(mockStore, mockLogger)

	// Test updating creditor mapping
	strategy.UpdateCreditorMapping("New Store", models.CategoryShopping)

	// Test categorization with updated mapping
	ctx := context.Background()
	transaction := Transaction{
		PartyName: "New Store",
		IsDebtor:  false,
	}

	category, found, err := strategy.Categorize(ctx, transaction)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, models.CategoryShopping, category.Name)

	// Test updating debtor mapping
	strategy.UpdateDebtorMapping("New Person", models.CategorySalary)

	// Test categorization with updated mapping
	transaction = Transaction{
		PartyName: "New Person",
		IsDebtor:  true,
	}

	category, found, err = strategy.Categorize(ctx, transaction)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, models.CategorySalary, category.Name)
}

func TestDirectMappingStrategy_ReloadMappings(t *testing.T) {
	// Create mock store with initial mappings
	mockStore := &store.MockCategoryStore{
		CreditorMappings: map[string]string{
			"initial store": models.CategoryShopping,
		},
		DebtorMappings: map[string]string{
			"initial person": models.CategorySalary,
		},
	}

	// Create mock logger
	mockLogger := &logging.MockLogger{}

	// Create strategy
	strategy := NewDirectMappingStrategy(mockStore, mockLogger)

	// Verify initial mapping works
	ctx := context.Background()
	transaction := Transaction{
		PartyName: "Initial Store",
		IsDebtor:  false,
	}

	category, found, err := strategy.Categorize(ctx, transaction)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, models.CategoryShopping, category.Name)

	// Update the mock store with new mappings
	mockStore.CreditorMappings = map[string]string{
		"updated store": models.CategoryGroceries,
	}
	mockStore.DebtorMappings = map[string]string{
		"updated person": models.CategoryTransport,
	}

	// Reload mappings
	strategy.ReloadMappings()

	// Verify old mapping no longer works
	category, found, err = strategy.Categorize(ctx, transaction)
	require.NoError(t, err)
	assert.False(t, found)

	// Verify new mapping works
	transaction = Transaction{
		PartyName: "Updated Store",
		IsDebtor:  false,
	}

	category, found, err = strategy.Categorize(ctx, transaction)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, models.CategoryGroceries, category.Name)
}

func TestDirectMappingStrategy_ReloadMappings_RaceCondition(t *testing.T) {
	// This test verifies no race condition during concurrent reload and categorize
	mockStore := &store.MockCategoryStore{
		CreditorMappings: map[string]string{
			"coop": models.CategoryGroceries,
		},
		DebtorMappings: map[string]string{
			"john doe": models.CategorySalary,
		},
	}

	mockLogger := &logging.MockLogger{}
	strategy := NewDirectMappingStrategy(mockStore, mockLogger)

	// Start concurrent operations
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Goroutine 1: Continuously reload mappings
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			strategy.ReloadMappings()
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Goroutines 2-11: Continuously categorize
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				tx := Transaction{
					PartyName: "COOP",
					IsDebtor:  false,
				}
				_, found, err := strategy.Categorize(context.Background(), tx)
				if err != nil {
					errors <- err
				}
				// During reload, we might get found=false, but never an error
				if found {
					// Category was found - this is expected most of the time
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Verify no errors occurred
	for err := range errors {
		t.Errorf("Unexpected error during concurrent operations: %v", err)
	}
}
