package linktransfers_test

import (
	"testing"

	"fjacquet/camt-csv/cmd/linktransfers"

	"github.com/stretchr/testify/assert"
)

func TestLinkTransfersCommand_Metadata(t *testing.T) {
	assert.Equal(t, "link-transfers", linktransfers.Cmd.Use)
	assert.Equal(t, "tools", linktransfers.Cmd.GroupID)
	names := map[string]bool{}
	for _, c := range linktransfers.Cmd.Commands() {
		names[c.Name()] = true
	}
	assert.True(t, names["preview"])
	assert.True(t, names["apply"])
}
