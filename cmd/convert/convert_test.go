package convert

import (
	"fjacquet/camt-csv/internal/logging"
	"os"
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

func writeFile(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0750))
	require.NoError(t, os.WriteFile(path, []byte("x"), 0600))
}

// Without --recursive only the top level is listed.
func TestDiscoverInputs_NonRecursive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "top.csv"))
	writeFile(t, filepath.Join(dir, "sub", "nested.csv"))

	files, err := discoverInputs(dir, false)

	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, filepath.Join(dir, "top.csv"), files[0])
}

// The command registers --recursive, so it must actually descend. Before this,
// the flag was accepted and silently ignored.
func TestDiscoverInputs_Recursive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "top.csv"))
	writeFile(t, filepath.Join(dir, "sub", "nested.csv"))
	writeFile(t, filepath.Join(dir, "sub", "deeper", "deep.xml"))

	files, err := discoverInputs(dir, true)

	require.NoError(t, err)
	require.Len(t, files, 3)
	assert.Contains(t, files, filepath.Join(dir, "sub", "deeper", "deep.xml"))
}

func TestDiscoverInputs_SkipsHiddenEntries(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "visible.csv"))
	writeFile(t, filepath.Join(dir, ".manifest.json"))
	writeFile(t, filepath.Join(dir, ".hidden", "secret.csv"))

	files, err := discoverInputs(dir, true)

	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, filepath.Join(dir, "visible.csv"), files[0])
}

func TestDiscoverInputs_UnreadableDirectoryIsAnError(t *testing.T) {
	_, err := discoverInputs(filepath.Join(t.TempDir(), "missing"), false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read directory")
}

// Output mirrors the input tree, so two statements sharing a basename in
// different folders cannot overwrite each other.
func TestOutputPathFor_MirrorsInputTree(t *testing.T) {
	in, out := "/in", "/out"
	claimed := map[string]bool{}
	logger := logging.NewLogrusAdapter("error", "text")

	jan := outputPathFor(in, filepath.Join(in, "jan", "statement.xml"), out, claimed, logger)
	feb := outputPathFor(in, filepath.Join(in, "feb", "statement.xml"), out, claimed, logger)

	assert.Equal(t, filepath.Join(out, "jan", "statement.csv"), jan)
	assert.Equal(t, filepath.Join(out, "feb", "statement.csv"), feb)
	assert.NotEqual(t, jan, feb)
}

// Same directory, same stem, different extensions: the second one takes the
// source extension into its name rather than replacing the first result.
func TestOutputPathFor_DisambiguatesSameStem(t *testing.T) {
	in, out := "/in", "/out"
	claimed := map[string]bool{}
	logger := logging.NewLogrusAdapter("error", "text")

	first := outputPathFor(in, filepath.Join(in, "statement.pdf"), out, claimed, logger)
	second := outputPathFor(in, filepath.Join(in, "statement.csv"), out, claimed, logger)

	assert.Equal(t, filepath.Join(out, "statement.csv"), first)
	assert.Equal(t, filepath.Join(out, "statement-csv.csv"), second)
	assert.NotEqual(t, first, second)
}
