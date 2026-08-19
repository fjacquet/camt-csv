package main

import (
	"sort"
	"testing"

	"fjacquet/camt-csv/cmd/root"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests live in package main, not cmd/root, because subcommands are
// registered by main.go's init(), which cmd/root cannot see (cmd/convert
// imports cmd/root, so the reverse import would cycle). go test builds a
// real binary for package main, running init() without calling main(), so
// root.Cmd here carries the real command tree.

// The root namespace holds verbs only, grouped so the primary function and
// the diagnostic do not read as peers.
func TestRootCommand_HasOnlyConvertAndCategorize(t *testing.T) {
	var names []string
	for _, c := range root.Cmd.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		names = append(names, c.Name())
	}
	sort.Strings(names)

	assert.Equal(t, []string{"categorize", "convert"}, names)
}

func TestRootCommand_RemovedFormatCommandsAreGone(t *testing.T) {
	removed := []string{"camt", "pdf", "selma", "viseca", "revolut",
		"revolut-crypto", "revolut-investment", "debit"}

	for _, name := range removed {
		for _, c := range root.Cmd.Commands() {
			assert.NotEqual(t, name, c.Name(), "%s must be gone; use convert --from %s", name, name)
		}
	}
}

func TestRootCommand_GroupsSeparatePrimaryFromAccessory(t *testing.T) {
	byName := map[string]*cobra.Command{}
	for _, c := range root.Cmd.Commands() {
		byName[c.Name()] = c
	}

	require.Contains(t, byName, "convert")
	require.Contains(t, byName, "categorize")
	assert.Equal(t, "conversion", byName["convert"].GroupID)
	assert.Equal(t, "tools", byName["categorize"].GroupID)
}

// The tool has handled seven formats since long before this change; the help
// text still described it as a CAMT.053 converter.
func TestRootCommand_ShortDescriptionIsNotCAMTOnly(t *testing.T) {
	assert.NotContains(t, root.Cmd.Short, "CAMT.053 XML files to CSV",
		"the tool converts seven formats, not one")
}
