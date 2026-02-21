package store

import (
	"errors"
	"testing"

	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockCategoryStore_LoadCategories(t *testing.T) {
	tests := []struct {
		name           string
		categories     []models.CategoryConfig
		loadError      error
		wantCategories []models.CategoryConfig
		wantErr        bool
	}{
		{
			name: "returns configured categories",
			categories: []models.CategoryConfig{
				{Name: "Food", Keywords: []string{"restaurant", "grocery"}},
				{Name: "Transport", Keywords: []string{"bus", "train"}},
			},
			wantCategories: []models.CategoryConfig{
				{Name: "Food", Keywords: []string{"restaurant", "grocery"}},
				{Name: "Transport", Keywords: []string{"bus", "train"}},
			},
		},
		{
			name:           "returns empty slice when no categories configured",
			categories:     []models.CategoryConfig{},
			wantCategories: []models.CategoryConfig{},
		},
		{
			name:           "returns nil when nil categories configured",
			categories:     nil,
			wantCategories: nil,
		},
		{
			name:       "returns error when configured",
			loadError:  errors.New("load categories failed"),
			categories: []models.CategoryConfig{{Name: "Ignored"}},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCategoryStore{
				Categories:          tt.categories,
				LoadCategoriesError: tt.loadError,
			}

			got, err := mock.LoadCategories()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantCategories, got)
		})
	}
}

func TestMockCategoryStore_LoadCreditorMappings(t *testing.T) {
	tests := []struct {
		name     string
		mappings map[string]string
		loadErr  error
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "returns configured mappings as copy",
			mappings: map[string]string{"Alice": "Food", "Bob": "Transport"},
			wantLen:  2,
		},
		{
			name:     "returns empty map when nil mappings",
			mappings: nil,
			wantLen:  0,
		},
		{
			name:     "returns empty map when empty mappings",
			mappings: map[string]string{},
			wantLen:  0,
		},
		{
			name:    "returns error when configured",
			loadErr: errors.New("creditor load failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCategoryStore{
				CreditorMappings:          tt.mappings,
				LoadCreditorMappingsError: tt.loadErr,
			}

			got, err := mock.LoadCreditorMappings()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestMockCategoryStore_LoadCreditorMappings_ReturnsCopy(t *testing.T) {
	original := map[string]string{"Alice": "Food"}
	mock := &MockCategoryStore{CreditorMappings: original}

	got, err := mock.LoadCreditorMappings()
	require.NoError(t, err)

	// Modify the returned map
	got["Alice"] = "Modified"

	// Original should be unchanged
	assert.Equal(t, "Food", mock.CreditorMappings["Alice"])
}

func TestMockCategoryStore_LoadDebtorMappings(t *testing.T) {
	tests := []struct {
		name     string
		mappings map[string]string
		loadErr  error
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "returns configured mappings as copy",
			mappings: map[string]string{"Employer": "Salary"},
			wantLen:  1,
		},
		{
			name:     "returns empty map when nil mappings",
			mappings: nil,
			wantLen:  0,
		},
		{
			name:     "returns empty map when empty mappings",
			mappings: map[string]string{},
			wantLen:  0,
		},
		{
			name:    "returns error when configured",
			loadErr: errors.New("debtor load failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCategoryStore{
				DebtorMappings:          tt.mappings,
				LoadDebtorMappingsError: tt.loadErr,
			}

			got, err := mock.LoadDebtorMappings()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestMockCategoryStore_LoadDebtorMappings_ReturnsCopy(t *testing.T) {
	original := map[string]string{"Employer": "Salary"}
	mock := &MockCategoryStore{DebtorMappings: original}

	got, err := mock.LoadDebtorMappings()
	require.NoError(t, err)

	got["Employer"] = "Modified"
	assert.Equal(t, "Salary", mock.DebtorMappings["Employer"])
}

func TestMockCategoryStore_SaveCreditorMappings(t *testing.T) {
	tests := []struct {
		name            string
		initialMappings map[string]string
		saveInput       map[string]string
		saveErr         error
		wantErr         bool
		wantStored      map[string]string
	}{
		{
			name:       "saves to nil initial mappings",
			saveInput:  map[string]string{"New": "Category"},
			wantStored: map[string]string{"New": "Category"},
		},
		{
			name:            "merges with existing mappings",
			initialMappings: map[string]string{"Existing": "Cat1"},
			saveInput:       map[string]string{"New": "Cat2"},
			wantStored:      map[string]string{"Existing": "Cat1", "New": "Cat2"},
		},
		{
			name:            "overwrites existing key",
			initialMappings: map[string]string{"Key": "Old"},
			saveInput:       map[string]string{"Key": "New"},
			wantStored:      map[string]string{"Key": "New"},
		},
		{
			name:    "returns error when configured",
			saveErr: errors.New("save creditor failed"),
			wantErr: true,
		},
		{
			name:       "saves empty map",
			saveInput:  map[string]string{},
			wantStored: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCategoryStore{
				CreditorMappings:          tt.initialMappings,
				SaveCreditorMappingsError: tt.saveErr,
			}

			err := mock.SaveCreditorMappings(tt.saveInput)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			for k, v := range tt.wantStored {
				assert.Equal(t, v, mock.CreditorMappings[k])
			}
		})
	}
}

func TestMockCategoryStore_SaveDebtorMappings(t *testing.T) {
	tests := []struct {
		name            string
		initialMappings map[string]string
		saveInput       map[string]string
		saveErr         error
		wantErr         bool
		wantStored      map[string]string
	}{
		{
			name:       "saves to nil initial mappings",
			saveInput:  map[string]string{"Employer": "Salary"},
			wantStored: map[string]string{"Employer": "Salary"},
		},
		{
			name:            "merges with existing mappings",
			initialMappings: map[string]string{"Existing": "Cat1"},
			saveInput:       map[string]string{"New": "Cat2"},
			wantStored:      map[string]string{"Existing": "Cat1", "New": "Cat2"},
		},
		{
			name:    "returns error when configured",
			saveErr: errors.New("save debtor failed"),
			wantErr: true,
		},
		{
			name:       "saves empty map",
			saveInput:  map[string]string{},
			wantStored: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCategoryStore{
				DebtorMappings:          tt.initialMappings,
				SaveDebtorMappingsError: tt.saveErr,
			}

			err := mock.SaveDebtorMappings(tt.saveInput)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			for k, v := range tt.wantStored {
				assert.Equal(t, v, mock.DebtorMappings[k])
			}
		})
	}
}

func TestMockCategoryStore_FindConfigFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantPath string
	}{
		{
			name:     "returns mock path with filename",
			filename: "categories.yaml",
			wantPath: "/mock/path/categories.yaml",
		},
		{
			name:     "returns mock path with different filename",
			filename: "creditors.yaml",
			wantPath: "/mock/path/creditors.yaml",
		},
		{
			name:     "handles empty filename",
			filename: "",
			wantPath: "/mock/path/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCategoryStore{}

			got, err := mock.FindConfigFile(tt.filename)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantPath, got)
		})
	}
}

func TestMockCategoryStore_ImplementsInterface(t *testing.T) {
	// Compile-time check that MockCategoryStore satisfies CategoryStoreInterface.
	// This uses the categorizer package's interface indirectly by verifying the
	// same method signatures.
	mock := &MockCategoryStore{}

	_, _ = mock.LoadCategories()
	_, _ = mock.LoadCreditorMappings()
	_, _ = mock.LoadDebtorMappings()
	_ = mock.SaveCreditorMappings(nil)
	_ = mock.SaveDebtorMappings(nil)
	_, _ = mock.FindConfigFile("")
}
