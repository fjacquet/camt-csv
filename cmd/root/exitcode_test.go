package root_test

import (
	"testing"

	"fjacquet/camt-csv/cmd/root"

	"github.com/stretchr/testify/assert"
)

// TestExitCode_RoundTrip verifies that a code recorded during a command is
// what main reads back once Execute has returned.
func TestExitCode_RoundTrip(t *testing.T) {
	root.ResetExitCode()
	defer root.ResetExitCode()

	assert.Equal(t, 0, root.ExitCode())

	root.SetExitCode(2)

	assert.Equal(t, 2, root.ExitCode())
}
