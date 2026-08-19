package convert

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"fjacquet/camt-csv/cmd/common"
	"fjacquet/camt-csv/internal/batch"
	"fjacquet/camt-csv/internal/config"
	"fjacquet/camt-csv/internal/container"
	"fjacquet/camt-csv/internal/parser"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// revolutSampleCSV is copied from internal/container/detect_test.go — its
// constants are unexported and in another package, so duplicating the one
// row this file needs is correct here.
const revolutSampleCSV = `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2026-03-15 10:00:00,2026-03-15 10:00:00,Coop Pronto,-24.50,0.00,CHF,COMPLETED,975.50`

// newTestContainer builds a container with AI disabled so tests never reach
// a network.
func newTestContainer(t *testing.T) *container.Container {
	t.Helper()
	t.Setenv("TEST_MODE", "true")

	cfg := &config.Config{}
	cfg.Log.Level = "error"
	cfg.Log.Format = "text"

	c, err := container.NewContainer(cfg)
	require.NoError(t, err)
	return c
}

// mustResolve returns the parser one run would use for path, failing the test
// if resolution errors.
func mustResolve(t *testing.T, c *container.Container, from, path string) parser.FullParser {
	t.Helper()
	resolve, err := common.ResolverFor(c, from)
	require.NoError(t, err)
	p, err := resolve(path)
	require.NoError(t, err)
	return p
}

// --from must offer exactly the registered parser types, so the flag cannot
// drift from what the container can actually build.
func TestResolverFor_AcceptsEveryDetectableType(t *testing.T) {
	c := newTestContainer(t)

	for _, pt := range container.DetectionOrder() {
		resolve, err := common.ResolverFor(c, string(pt))
		require.NoError(t, err, "--from %s must be accepted", pt)

		p, err := resolve("/irrelevant/path.csv")
		require.NoError(t, err)
		assert.NotNil(t, p)
	}
}

func TestResolverFor_RejectsUnknownFormat(t *testing.T) {
	c := newTestContainer(t)

	_, err := common.ResolverFor(c, "postbank")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "postbank")
	assert.Contains(t, err.Error(), "camt", "the error must list the valid values")
}

// --from is an escape hatch, not a filter: it pins the parser even for a file
// the detector would have routed elsewhere, so a wrong pin fails loudly rather
// than silently falling back to the right parser.
func TestResolverFor_PinBypassesDetection(t *testing.T) {
	c := newTestContainer(t)
	dir := t.TempDir()
	revolutFile := filepath.Join(dir, "statement.csv")
	require.NoError(t, os.WriteFile(revolutFile, []byte(revolutSampleCSV), 0600))

	resolve, err := common.ResolverFor(c, "selma")
	require.NoError(t, err)

	p, err := resolve(revolutFile)
	require.NoError(t, err, "the pin resolves regardless of content")

	valid, _ := p.ValidateFormat(revolutFile)
	assert.False(t, valid, "the pinned Selma parser must not accept a Revolut file")
}

// With no --from, resolution falls back to detection and an unrecognized file
// is reported as such rather than guessed at.
func TestResolverFor_DetectsWhenUnset(t *testing.T) {
	c := newTestContainer(t)
	dir := t.TempDir()
	unknown := filepath.Join(dir, "mystery.csv")
	require.NoError(t, os.WriteFile(unknown, []byte("a,b,c\n1,2,3\n"), 0600))

	resolve, err := common.ResolverFor(c, "")
	require.NoError(t, err)

	_, err = resolve(unknown)
	assert.ErrorIs(t, err, batch.ErrNoParser)
}

// A single-file conversion's outcome is its exit code. A one-entry run report
// adds nothing, so none is written.
func TestConvert_SingleFileWritesNoManifest(t *testing.T) {
	c := newTestContainer(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "statement.csv")
	require.NoError(t, os.WriteFile(input, []byte(revolutSampleCSV), 0600))
	outputFile := filepath.Join(t.TempDir(), "out.csv")

	err := common.ProcessFileWithErrorFormatted(context.Background(),
		mustResolve(t, c, "", input), input, outputFile, false, c.GetLogger(), c, "standard")

	require.NoError(t, err)
	assert.FileExists(t, outputFile)
	assert.NoFileExists(t, batch.ManifestPathFor(outputFile))
}

// -o names a file. Pointing it at an existing directory is a convenience the
// PDF command used to offer, and it survives: the folder being read names the
// output written inside it.
func TestResolveOutputFile_ExistingDirectoryGetsGeneratedName(t *testing.T) {
	inputDir := filepath.Join(t.TempDir(), "releves-2024")
	require.NoError(t, os.MkdirAll(inputDir, 0750))
	outputDir := t.TempDir()

	got, err := common.ResolveOutputFile(inputDir, outputDir)

	require.NoError(t, err)
	assert.Equal(t, filepath.Join(outputDir, "releves-2024.csv"), got)
}

func TestResolveOutputFile_PlainPathIsUnchanged(t *testing.T) {
	inputDir := t.TempDir()
	want := filepath.Join(t.TempDir(), "out.csv")

	got, err := common.ResolveOutputFile(inputDir, want)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// Writing inside the folder being read means a later --recursive run reads its
// own output back as input. Refuse it rather than rely on no validator ever
// accepting our own CSV.
func TestResolveOutputFile_RefusesOutputUnderInput(t *testing.T) {
	inputDir := t.TempDir()
	nested := filepath.Join(inputDir, "sub")
	require.NoError(t, os.MkdirAll(nested, 0750))

	_, err := common.ResolveOutputFile(inputDir, filepath.Join(nested, "out.csv"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "input")
}

func TestCmd_IsRegisteredWithFlags(t *testing.T) {
	assert.Equal(t, "convert", Cmd.Use)
	assert.NotNil(t, Cmd.Flags().Lookup("format"), "convert must accept --format")
	assert.NotNil(t, Cmd.Flags().Lookup("recursive"), "convert must accept --recursive")
	assert.NotNil(t, Cmd.Flags().Lookup("from"), "convert must accept --from")
	assert.NotNil(t, Cmd.Flags().Lookup("keep-payments"), "convert must accept --keep-payments")
}
