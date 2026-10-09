package icompta

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scopeAccounts() []Account {
	return []Account{
		{ID: "F", Name: "Fred", Class: classFolder},
		{ID: "FR", Name: "France", Class: classFolder, ParentID: "F"},
		{ID: "CA", Name: "CA-Fred", Class: classFolder, ParentID: "FR"},
		{ID: "A1", Name: "Compte cheque", Class: classAccount, ParentID: "CA"}, // two levels under Fred
		{ID: "A2", Name: "Revolut CHF", Class: classAccount, ParentID: "F"},
		{ID: "L", Name: "Lydie", Class: classFolder},
		{ID: "A3", Name: "Livret A", Class: classAccount, ParentID: "L"},
		{ID: "P", Name: "Florence", Class: "ICPerson", ParentID: "F"},
		{ID: "A4", Name: "Orphan", Class: classAccount},
	}
}

func TestResolveScope_NestedAccountsOfOneFolder(t *testing.T) {
	scope, err := ResolveScope(scopeAccounts(), []string{"Fred"})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"A1": true, "A2": true}, scope,
		"accounts at any depth under Fred; never folders, persons or other folders' accounts")
}

func TestResolveScope_SeveralFolders(t *testing.T) {
	scope, err := ResolveScope(scopeAccounts(), []string{"Fred", "Lydie"})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"A1": true, "A2": true, "A3": true}, scope)
}

// Review focus 1: a wrong-case name must fail loudly and say what exists.
func TestResolveScope_UnknownFolderListsAvailable(t *testing.T) {
	_, err := ResolveScope(scopeAccounts(), []string{"fred"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `folder "fred" not found`)
	assert.Contains(t, err.Error(), "CA-Fred, France, Fred, Lydie")
}

func TestResolveScope_DuplicateFolderNameNamesBothPaths(t *testing.T) {
	accts := append(scopeAccounts(), Account{ID: "F2", Name: "Fred", Class: classFolder, ParentID: "L"})
	_, err := ResolveScope(accts, []string{"Fred"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Fred, Lydie / Fred")
}

func TestResolveScope_NoFolderIsAnError(t *testing.T) {
	_, err := ResolveScope(scopeAccounts(), nil)
	require.Error(t, err)
}

func TestResolveScope_ParentCycleTerminates(t *testing.T) {
	accts := []Account{
		{ID: "X", Name: "X", Class: classFolder, ParentID: "Y"},
		{ID: "Y", Name: "Y", Class: classFolder, ParentID: "X"},
		{ID: "A", Name: "A", Class: classAccount, ParentID: "X"},
	}
	scope, err := ResolveScope(accts, []string{"Y"})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"A": true}, scope)
}
