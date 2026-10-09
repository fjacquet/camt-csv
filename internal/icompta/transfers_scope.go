package icompta

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ICAccount.class values: a real account, a folder ("accounts group"). Persons
// (ICPerson) and anything else are never in scope.
const (
	classAccount = "ICAccount"
	classFolder  = "ICAccountsGroup"
)

// Account is one row of ICAccount.
type Account struct {
	ID, Name, Class, ParentID, CurrencyID string
}

// ResolveScope returns the IDs of the real accounts under the named folders, at
// any depth. A name that matches no folder, or several, is an error: a silent
// wrong scope would link transfers to accounts the user does not manage.
func ResolveScope(accounts []Account, folders []string) (map[string]bool, error) {
	if len(folders) == 0 {
		return nil, errors.New("at least one --folder is required")
	}
	byID := make(map[string]Account, len(accounts))
	for _, a := range accounts {
		byID[a.ID] = a
	}
	chosen := map[string]bool{}
	for _, f := range folders {
		var hits []Account
		for _, a := range accounts {
			if a.Class == classFolder && normalize(a.Name) == normalize(f) {
				hits = append(hits, a)
			}
		}
		switch len(hits) {
		case 0:
			return nil, fmt.Errorf("folder %q not found; folders are: %s", f, strings.Join(folderNames(accounts), ", "))
		case 1:
			chosen[hits[0].ID] = true
		default:
			paths := make([]string, len(hits))
			for i, h := range hits {
				paths[i] = folderPath(byID, h)
			}
			sort.Strings(paths)
			return nil, fmt.Errorf("folder %q is ambiguous: %s", f, strings.Join(paths, ", "))
		}
	}
	scope := map[string]bool{}
	for _, a := range accounts {
		if a.Class == classAccount && underAny(byID, a, chosen) {
			scope[a.ID] = true
		}
	}
	return scope, nil
}

// underAny walks a's parent chain; seen guards against a corrupt cycle.
func underAny(byID map[string]Account, a Account, chosen map[string]bool) bool {
	seen := map[string]bool{}
	for p := a.ParentID; p != "" && !seen[p]; p = byID[p].ParentID {
		if chosen[p] {
			return true
		}
		seen[p] = true
	}
	return false
}

func folderPath(byID map[string]Account, a Account) string {
	parts := []string{a.Name}
	seen := map[string]bool{a.ID: true}
	for p := a.ParentID; p != "" && !seen[p]; p = byID[p].ParentID {
		seen[p] = true
		parts = append([]string{byID[p].Name}, parts...)
	}
	return strings.Join(parts, " / ")
}

func folderNames(accounts []Account) []string {
	var names []string
	for _, a := range accounts {
		if a.Class == classFolder {
			names = append(names, a.Name)
		}
	}
	sort.Strings(names)
	return names
}
