package categorizer

import (
	"context"
	"os"
	"strings"
	"testing"

	"fjacquet/camt-csv/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBatchAnswer(t *testing.T) {
	raw := "```json\n{\"Kiro\": \"Abonnements\", \" Migros \": \"**Courses**\"}\n```"
	got, err := parseBatchAnswer(raw)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"kiro": "Abonnements", "migros": "Courses"}, got)

	_, err = parseBatchAnswer("I cannot help with that")
	assert.Error(t, err)
}

func TestBaseClientCategorizeBatch_OneRequestForManyParties(t *testing.T) {
	b := newBaseAIClient("test", testLogger(), "key", 60)
	var prompts []string
	complete := func(_ context.Context, prompt string) (string, error) {
		prompts = append(prompts, prompt)
		return `{"Kiro": "Abonnements", "Migros": "Courses"}`, nil
	}
	txs := []models.Transaction{{PartyName: "Kiro"}, {PartyName: "Migros"}}

	got, err := b.categorizeBatch(context.Background(), txs, complete)
	require.NoError(t, err)
	assert.Len(t, prompts, 1)
	assert.Contains(t, prompts[0], "Kiro")
	assert.Contains(t, prompts[0], "Migros")
	assert.Equal(t, "Abonnements", got["kiro"])
	assert.True(t, strings.Contains(prompts[0], "JSON"))
}

// The single-transaction prompt must stay byte-identical to the one produced
// before the preamble was split out for the batch prompt.
func TestBuildCategorizationPrompt_UnchangedByPreambleSplit(t *testing.T) {
	golden, err := os.ReadFile("testdata/single_prompt_golden.txt")
	require.NoError(t, err)
	tx := models.Transaction{PartyName: "Migros", Description: "Achat 100%", Amount: decimal.RequireFromString("-12.50")}
	assert.Equal(t, string(golden), buildCategorizationPrompt(tx))
}
