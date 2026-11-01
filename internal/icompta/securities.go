package icompta

import (
	"context"
	"fmt"
	"regexp"
)

var isinPrefix = regexp.MustCompile(`^([A-Z]{2}[A-Z0-9]{9}[0-9])-`)

// SecurityNames maps ISIN to the full name of each iCompta security named
// "ISIN-name". The database is opened read-only, so it is safe while iCompta runs.
func SecurityNames(ctx context.Context, path string) (map[string]string, error) {
	s, err := OpenReadOnly(ctx, path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = s.Close() }()
	rows, err := s.db.QueryContext(ctx, `SELECT name FROM ICSecurity`)
	if err != nil {
		return nil, fmt.Errorf("read iCompta securities: %w", err)
	}
	defer func() { _ = rows.Close() }()
	names := map[string]string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("read iCompta securities: %w", err)
		}
		if m := isinPrefix.FindStringSubmatch(name); m != nil {
			names[m[1]] = name
		}
	}
	return names, rows.Err()
}
