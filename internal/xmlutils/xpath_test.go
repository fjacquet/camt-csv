package xmlutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrEmpty(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		index    int
		expected string
	}{
		{
			name:     "valid index returns value",
			slice:    []string{"a", "b", "c"},
			index:    1,
			expected: "b",
		},
		{
			name:     "first index",
			slice:    []string{"first", "second"},
			index:    0,
			expected: "first",
		},
		{
			name:     "last valid index",
			slice:    []string{"x", "y", "z"},
			index:    2,
			expected: "z",
		},
		{
			name:     "index out of bounds returns empty",
			slice:    []string{"a", "b"},
			index:    5,
			expected: "",
		},
		{
			name:     "empty slice returns empty",
			slice:    []string{},
			index:    0,
			expected: "",
		},
		{
			name:     "nil slice returns empty",
			slice:    nil,
			index:    0,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetOrEmpty(tt.slice, tt.index)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple text unchanged",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "removes extra whitespace",
			input:    "Hello    World",
			expected: "Hello World",
		},
		{
			name:     "removes newlines",
			input:    "Hello\nWorld",
			expected: "Hello World",
		},
		{
			name:     "removes tabs",
			input:    "Hello\tWorld",
			expected: "Hello World",
		},
		{
			name:     "trims leading/trailing whitespace",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
		{
			name:     "removes Remittance Info prefix",
			input:    "Remittance Info: Payment for services",
			expected: "Payment for services",
		},
		{
			name:     "removes Remittance Information prefix",
			input:    "Remittance Information: Invoice 123",
			expected: "Invoice 123",
		},
		{
			name:     "removes Additional Entry Info prefix",
			input:    "Additional Entry Info: Extra data",
			expected: "Extra data",
		},
		{
			name:     "removes Additional Transaction Info prefix",
			input:    "Additional Transaction Info: Details here",
			expected: "Details here",
		},
		{
			name:     "removes Details prefix",
			input:    "Details: Some details",
			expected: "Some details",
		},
		{
			name:     "removes End-to-End prefix",
			input:    "End-to-End: REF123456",
			expected: "REF123456",
		},
		{
			name:     "replaces IBAN patterns",
			input:    "Payment from CH9300762011623852957",
			expected: "Payment from IBAN",
		},
		{
			name:     "removes excessive separators",
			input:    "Hello,,,World...Test;;;Final",
			expected: "Hello World Test Final",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "handles only whitespace",
			input:    "   \n\t   ",
			expected: "",
		},
		{
			name:     "complex mixed content",
			input:    "Remittance Info: Payment,,,  from  \n CH9300762011623852957;;;",
			expected: "Payment from IBAN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadXMLFile(t *testing.T) {
	t.Run("loads valid XML file", func(t *testing.T) {
		tmpDir := t.TempDir()
		xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<root>
	<item>Value1</item>
	<item>Value2</item>
</root>`
		xmlPath := filepath.Join(tmpDir, "test.xml")
		err := os.WriteFile(xmlPath, []byte(xmlContent), 0600)
		require.NoError(t, err)

		root, err := LoadXMLFile(xmlPath)
		require.NoError(t, err)
		assert.NotNil(t, root)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := LoadXMLFile("/non/existent/file.xml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open XML file")
	})

	t.Run("returns error for invalid XML", func(t *testing.T) {
		tmpDir := t.TempDir()
		xmlPath := filepath.Join(tmpDir, "invalid.xml")
		err := os.WriteFile(xmlPath, []byte("<invalid><unclosed>"), 0600)
		require.NoError(t, err)

		_, err = LoadXMLFile(xmlPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse XML file")
	})
}

func TestExtractFromXML(t *testing.T) {
	tmpDir := t.TempDir()
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<root>
	<items>
		<item>First</item>
		<item>Second</item>
		<item>Third</item>
	</items>
	<single>OnlyOne</single>
</root>`
	xmlPath := filepath.Join(tmpDir, "test.xml")
	err := os.WriteFile(xmlPath, []byte(xmlContent), 0600)
	require.NoError(t, err)

	root, err := LoadXMLFile(xmlPath)
	require.NoError(t, err)

	t.Run("extracts multiple values", func(t *testing.T) {
		values, err := ExtractFromXML(root, "//item")
		require.NoError(t, err)
		assert.Len(t, values, 3)
		assert.Equal(t, "First", values[0])
		assert.Equal(t, "Second", values[1])
		assert.Equal(t, "Third", values[2])
	})

	t.Run("extracts single value", func(t *testing.T) {
		values, err := ExtractFromXML(root, "//single")
		require.NoError(t, err)
		assert.Len(t, values, 1)
		assert.Equal(t, "OnlyOne", values[0])
	})

	t.Run("returns empty for no matches", func(t *testing.T) {
		values, err := ExtractFromXML(root, "//nonexistent")
		require.NoError(t, err)
		assert.Empty(t, values)
	})

	t.Run("returns error for invalid xpath", func(t *testing.T) {
		_, err := ExtractFromXML(root, "[invalid xpath")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to compile XPath")
	})
}

func TestExtractWithXPath(t *testing.T) {
	tmpDir := t.TempDir()
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<root>
	<data>TestValue</data>
</root>`
	xmlPath := filepath.Join(tmpDir, "test.xml")
	err := os.WriteFile(xmlPath, []byte(xmlContent), 0600)
	require.NoError(t, err)

	t.Run("extracts value from file", func(t *testing.T) {
		values, err := ExtractWithXPath(xmlPath, "//data")
		require.NoError(t, err)
		assert.Len(t, values, 1)
		assert.Equal(t, "TestValue", values[0])
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := ExtractWithXPath("/non/existent/file.xml", "//data")
		assert.Error(t, err)
	})
}

func TestDefaultCamt053XPaths(t *testing.T) {
	xpaths := DefaultCamt053XPaths()

	t.Run("entry xpaths are set", func(t *testing.T) {
		assert.NotEmpty(t, xpaths.Entry.Amount)
		assert.NotEmpty(t, xpaths.Entry.Currency)
		assert.NotEmpty(t, xpaths.Entry.CreditDebitInd)
		assert.NotEmpty(t, xpaths.Entry.BookingDate)
		assert.NotEmpty(t, xpaths.Entry.ValueDate)
	})

	t.Run("references xpaths are set", func(t *testing.T) {
		assert.NotEmpty(t, xpaths.References.EndToEndID)
		assert.NotEmpty(t, xpaths.References.TransactionID)
	})

	t.Run("party xpaths are set", func(t *testing.T) {
		assert.NotEmpty(t, xpaths.Party.DebtorName)
		assert.NotEmpty(t, xpaths.Party.CreditorName)
	})

	t.Run("account xpaths are set", func(t *testing.T) {
		assert.NotEmpty(t, xpaths.Account.IBAN)
	})

	t.Run("xpaths start with //", func(t *testing.T) {
		assert.True(t, len(xpaths.Entry.Amount) > 2)
		assert.Equal(t, "//", xpaths.Entry.Amount[:2])
	})
}
