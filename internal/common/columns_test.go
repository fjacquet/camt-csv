package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMissingColumn(t *testing.T) {
	header := []string{"\ufeffDate", " Amount ", "Currency"}
	assert.Equal(t, "", MissingColumn(header, "Date", "Amount", "Currency"))
	assert.Equal(t, "State", MissingColumn(header, "Date", "State"))
}
