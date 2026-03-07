package relay

import "testing"

// newTestManager creates a Manager and registers t.Cleanup to stop it,
// preventing healthCheckLoop goroutine leaks across tests.
func newTestManager(t *testing.T, opts ...ManagerOption) *Manager {
	t.Helper()
	m := NewManagerWithOptions(opts...)
	t.Cleanup(func() { m.Stop() })
	return m
}
