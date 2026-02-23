// Package common exports internal symbols for testing.
// This file is only compiled during tests.
package common

// SetOsExitFn replaces the os.Exit function used by FolderConvert.
// Used in tests to prevent the process from exiting. Returns a restore function.
func SetOsExitFn(fn func(int)) func() {
	prev := osExitFn
	osExitFn = fn
	return func() { osExitFn = prev }
}
