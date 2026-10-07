package csvsafe

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeRoundTrip(t *testing.T) {
	for _, s := range []string{"", "Migros", `=HYPERLINK("x")`, "+1", "-2", "@SUM(A1)", "\tx", "'=already", "'plain"} {
		assert.Equal(t, s, Unescape(Escape(s)), s)
	}
	assert.Equal(t, `'=HYPERLINK("x")`, Escape(`=HYPERLINK("x")`))
	assert.Equal(t, "''=already", Escape("'=already"), "an escaped-looking value is escaped again")
	assert.Equal(t, "Migros", Escape("Migros"))
}
