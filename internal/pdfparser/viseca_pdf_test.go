package pdfparser

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Continuation lines (category, rate, fee) follow their transaction and must not
// leak onto the next one, including across a page break.
func TestVisecaPDF_ContinuationLinesStayWithTheirTransaction(t *testing.T) {
	text := `Date de                                   Montant en
Date       valeur Détails                 Monnaie     Montant         CHF
4627 23XX XXXX 4748 Visa Gold, Frederic Jacquet
Limite de carte CHF 5'000
23.09.26 24.09.26 MICROSOFT#G1, MSBILL.INFO CH                                  12.70
                  Ordinateurs et logiciels
24.09.26 25.09.26 IMMUNOTEC, WWW.IMMUNOTEC IE        EUR      111.74         108.60
                  Télémarketing, vente directe
                  Taux de conversion 0.95545395 du 24.09.26    CHF     106.75
                  Frais de traitement 1.75%                    CHF       1.85
24.09.26 25.09.26 IMMUNOTEC, WWW.IMMUNOTEC IE        EUR       97.99          95.25
                  Télémarketing, vente directe
                  Taux de conversion 0.95545395 du 24.09.26    CHF      93.60
                  Frais de traitement 1.75%                    CHF       1.65
                   Total intermédiaire de la carte                          216.55
Page 1/3
Date       valeur Détails                 Monnaie     Montant         CHF
                   Report total                                             216.55
4627 23XX XXXX 4748 Visa Gold, Frederic Jacquet
27.09.26 28.09.26 HDRS SA, Vevey CH                                            160.00
                  Restaurants, bistros
19.10.26 24.09.26 APPLE.COM/BILL, ITUNES.COM IE                                  8.35 -
                  Biens numériques
`
	file := filepath.Join(t.TempDir(), "v.pdf")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0600))
	f, err := os.Open(file)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	adapter := NewAdapter(logging.NewLogrusAdapter("info", "text"), NewMockPDFExtractor(text, nil))
	txs, err := adapter.Parse(context.Background(), f)
	require.NoError(t, err)
	require.Len(t, txs, 5)

	want := []string{
		" - Ordinateurs et logiciels",
		" - Télémarketing, vente directe",
		" - Télémarketing, vente directe",
		" - Restaurants, bistros",
		" - Biens numériques",
	}
	for i, suffix := range want {
		assert.Contains(t, txs[i].Description, suffix, "tx %d", i)
		assert.NotContains(t, txs[i].Description, "Frais de traitement", "tx %d", i)
	}
	assert.Equal(t, "1.85", txs[1].Fees.String())
	assert.Equal(t, "0.95545395", txs[1].ExchangeRate.String())
	assert.Equal(t, "1.65", txs[2].Fees.String())
}
