// Package common exports internal symbols for testing.
// This file is only compiled during tests.
package common

// SetExitFn replaces the exit-code recorder used by FolderConvert.
// Used in tests to capture the requested exit code. Returns a restore function.
func SetExitFn(fn func(int)) func() {
	prev := exitFn
	exitFn = fn
	return func() { exitFn = prev }
}
