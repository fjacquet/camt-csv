package formatter

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pluginReferenceFile is regenerated from the ICImportPlugin table of the
// iCompta document; see docs/icompta-plugin-setup.md.
const pluginReferenceFile = "../../.planning/reference/icompta-import-plugins.txt"

type icomptaPlugin struct {
	name      string
	separator string
	mapping   map[string]string
}

// loadPluginReference parses the generated plugin reference into structs.
// Lines are "name|separator|dateFormat|encoding|transactionsMapping"; blank
// lines and "#" comments are ignored.
func loadPluginReference(t *testing.T) []icomptaPlugin {
	t.Helper()

	f, err := os.Open(filepath.Clean(pluginReferenceFile))
	require.NoError(t, err, "plugin reference missing; regenerate it from the iCompta document")
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("close plugin reference: %v", err)
		}
	}()

	var plugins []icomptaPlugin
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.SplitN(line, "|", 5)
		require.Len(t, fields, 5, "malformed plugin line: %s", line)

		mapping := map[string]string{}
		require.NoError(t, json.Unmarshal([]byte(fields[4]), &mapping),
			"plugin %s has invalid transactionsMapping JSON", fields[0])

		plugins = append(plugins, icomptaPlugin{name: fields[0], separator: fields[1], mapping: mapping})
	}
	require.NoError(t, scanner.Err())
	require.NotEmpty(t, plugins, "plugin reference contained no plugins")

	return plugins
}

// TestIComptaHeaderCoversPluginMappings is the regression guard for the class of
// bug where the formatter silently stops emitting a column an import plugin
// depends on. iCompta resolves columns by name against the header row, so a
// mapping that names a missing column resolves to nothing and the field is
// dropped on import without any error.
func TestIComptaHeaderCoversPluginMappings(t *testing.T) {
	header := map[string]bool{}
	for _, column := range NewIComptaFormatter().Header() {
		header[column] = true
	}

	for _, plugin := range loadPluginReference(t) {
		t.Run(plugin.name, func(t *testing.T) {
			for field, column := range plugin.mapping {
				assert.True(t, header[column],
					"plugin %s maps %q to column %q, which the icompta formatter does not emit",
					plugin.name, field, column)
			}
		})
	}
}

// TestIComptaPluginsUseFormatterDelimiter catches plugins configured for a
// different separator than the formatter writes. A comma-configured plugin
// reads a semicolon file as a single column and imports nothing usable.
func TestIComptaPluginsUseFormatterDelimiter(t *testing.T) {
	require.Equal(t, ';', NewIComptaFormatter().Delimiter(),
		"this test assumes the icompta formatter is semicolon-delimited")

	for _, plugin := range loadPluginReference(t) {
		assert.Equal(t, "LACharacterSeparator.SemicolonSeparator", plugin.separator,
			"plugin %s is not configured for the semicolon output the formatter writes", plugin.name)
	}
}
