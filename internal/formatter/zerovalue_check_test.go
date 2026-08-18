package formatter

import (
	"testing"

	"fjacquet/camt-csv/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIComptaFormatsZeroValueTransaction guards the uninitialised struct path:
// Transaction literals built without decimal fields carry a nil big.Int, and the
// formatter must render them rather than panic.
func TestIComptaFormatsZeroValueTransaction(t *testing.T) {
	rows, err := NewIComptaFormatter().Format([]models.Transaction{{}})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Len(t, rows[0], 20)
	assert.Equal(t, "", rows[0][15]) // NumberOfShares
	assert.Equal(t, "", rows[0][16]) // Fees
	assert.Equal(t, "", rows[0][19]) // TaxRate
}
