package recategorize_test

import (
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/cmd/recategorize"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sub(t *testing.T, name string) *cobra.Command {
	t.Helper()
	for _, c := range recategorize.Cmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	t.Fatalf("subcommand %q not found", name)
	return nil
}

func TestRecategorizeCommand_Metadata(t *testing.T) {
	assert.Equal(t, "recategorize", recategorize.Cmd.Use)
	assert.Equal(t, "tools", recategorize.Cmd.GroupID)
	assert.NotNil(t, sub(t, "preview"))
	assert.NotNil(t, sub(t, "apply"))
}

func TestPreviewCommand_FlagsAndArgs(t *testing.T) {
	preview := sub(t, "preview")
	require.NotNil(t, preview.Flags().Lookup("db"))
	require.NotNil(t, preview.Flags().Lookup("output"))
	assert.Equal(t, "o", preview.Flags().Lookup("output").Shorthand)
	assert.Error(t, preview.Args(preview, []string{"stray"}), "positional arguments are typos")
}

func TestApplyCommand_FlagsAndArgs(t *testing.T) {
	apply := sub(t, "apply")
	require.NotNil(t, apply.Flags().Lookup("db"))
	require.NotNil(t, apply.Flags().Lookup("report"))
	assert.Error(t, apply.Args(apply, []string{"stray"}))
}

// A missing or malformed report must fail before anything touches the database.
func TestApplyCommand_RejectsBadReport(t *testing.T) {
	dir := t.TempDir()
	notAReport := filepath.Join(dir, "r.csv")
	require.NoError(t, os.WriteFile(notAReport, []byte("a,b\n"), 0o600))

	apply := sub(t, "apply")
	assert.Error(t, apply.RunE(apply, nil), "missing flags")

	require.NoError(t, apply.Flags().Set("db", filepath.Join(dir, "x.cdb")))
	require.NoError(t, apply.Flags().Set("report", notAReport))
	err := apply.RunE(apply, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "report")
}
