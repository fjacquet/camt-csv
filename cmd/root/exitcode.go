package root

// pendingExitCode holds the process exit code requested by a command while it
// was still running. Calling os.Exit from inside a command would skip the root
// PersistentPostRun hook, which is what saves creditor/debitor mappings and
// closes the container, so commands record the code here instead and main
// exits with it once Execute has returned.
var pendingExitCode int

// SetExitCode records the exit code the process should terminate with.
func SetExitCode(code int) {
	pendingExitCode = code
}

// ExitCode returns the recorded exit code (0 when nothing failed).
func ExitCode() int {
	return pendingExitCode
}

// ResetExitCode clears the recorded exit code. Intended for tests.
func ResetExitCode() {
	pendingExitCode = 0
}
