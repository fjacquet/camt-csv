package convert

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Outputs are named after their inputs, so the command refuses to write into
// the input directory. The comparison must see through relative paths.
func TestSameDirectory(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"identical paths", dir, dir, true},
		{"trailing slash", dir, dir + "/", true},
		{"dot segment", dir, filepath.Join(dir, "."), true},
		{"round trip through a child", dir, filepath.Join(dir, "sub", ".."), true},
		{"distinct directories", dir, filepath.Join(dir, "out"), false},
		{"sibling directories", filepath.Join(dir, "in"), filepath.Join(dir, "out"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sameDirectory(tt.a, tt.b))
		})
	}
}

// The error message shown when detection fails lists the formats that were
// tried, so it must stay in sync with the detection order.
func TestParserTypeNames(t *testing.T) {
	names := parserTypeNames()

	require.NotEmpty(t, names)
	assert.Contains(t, names, "camt")
	assert.Contains(t, names, "pdf")
	assert.Contains(t, names, "revolut")
	assert.Contains(t, names, "revolut-crypto")
	assert.Contains(t, names, "revolut-investment")
	assert.Contains(t, names, "selma")
	assert.Contains(t, names, "debit")

	for _, name := range names {
		assert.NotEmpty(t, name)
	}
}

func TestCmd_IsRegisteredWithFlags(t *testing.T) {
	assert.Equal(t, "convert", Cmd.Use)
	assert.NotNil(t, Cmd.Flags().Lookup("format"), "convert must accept --format")
	assert.NotNil(t, Cmd.Flags().Lookup("recursive"), "convert must accept --recursive")
}
