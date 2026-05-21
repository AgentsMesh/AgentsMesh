//go:build windows

package processmgr

import "testing"

// Windows process kernel objects are reference-counted by open handles, so
// there is no zombie state to assert against. cmdProcess.reapLoop's call to
// cmd.Wait() closes the runtime's handle the moment the child exits; by the
// time tests reach this assertion, the OS has already torn down the entry.
func assertReaped(t *testing.T, pid int) {
	t.Helper()
}
