package batch_test

import (
	"testing"

	"fjacquet/camt-csv/cmd/batch"

	"github.com/stretchr/testify/assert"
)

func TestBatchCommand_CommandMetadata(t *testing.T) {
	assert.Equal(t, "batch", batch.Cmd.Use)
	assert.Contains(t, batch.Cmd.Short, "Batch process")
	assert.NotNil(t, batch.Cmd.Run)
}

func TestBatchCommand_LongDescription(t *testing.T) {
	assert.Contains(t, batch.Cmd.Long, "Batch process files")
	assert.Contains(t, batch.Cmd.Long, "input directory")
	assert.Contains(t, batch.Cmd.Long, "another directory")
}

func TestBatchCommand_Example(t *testing.T) {
	assert.Contains(t, batch.Cmd.Long, "Example")
	assert.Contains(t, batch.Cmd.Long, "batch")
}
