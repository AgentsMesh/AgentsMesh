package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// TestRunnerOption configures RunnerDeps for testing.
type TestRunnerOption func(*RunnerDeps)

// WithTestConfig overrides the test runner's config.
func WithTestConfig(cfg *config.Config) TestRunnerOption {
	return func(d *RunnerDeps) { d.Config = cfg }
}

// WithTestConnection overrides the test runner's connection.
func WithTestConnection(conn client.Connection) TestRunnerOption {
	return func(d *RunnerDeps) { d.Connection = conn }
}

// WithTestPodStore overrides the test runner's pod store.
func WithTestPodStore(store PodStore) TestRunnerOption {
	return func(d *RunnerDeps) { d.PodStore = store }
}

// NewTestRunner creates a Runner suitable for unit tests with sensible defaults.
// Returns the Runner and the MockConnection for assertion/verification.
func NewTestRunner(t *testing.T, opts ...TestRunnerOption) (*Runner, *client.MockConnection) {
	t.Helper()

	mockConn := client.NewMockConnection()
	deps := RunnerDeps{
		Config: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:     t.TempDir(),
			NodeID:            "test-node",
			OrgSlug:           "test-org",
		},
		Connection: mockConn,
	}

	for _, opt := range opts {
		opt(&deps)
	}

	// If an option replaced the connection, extract the MockConnection if possible.
	mc, _ := deps.Connection.(*client.MockConnection)
	if mc == nil {
		mc = mockConn
	}

	r, err := New(deps)
	if err != nil {
		t.Fatalf("NewTestRunner: %v", err)
	}

	return r, mc
}

// testPTYComponents extracts the PTYComponents from a Pod's IO for test assertions.
// Returns nil if the Pod does not use PTY mode.
func testPTYComponents(pod *Pod) *PTYComponents {
	if ptyIO, ok := pod.IO.(*PTYPodIO); ok {
		return ptyIO.components
	}
	return nil
}

// testNewPTYPod creates a minimal PTY Pod with VirtualTerminal for testing.
func testNewPTYPod(podKey string, vterm *vt.VirtualTerminal) *Pod {
	pod := &Pod{
		PodKey:          podKey,
		InteractionMode: InteractionModePTY,
		Status:          PodStatusRunning,
		vtProvider:      func() *vt.VirtualTerminal { return vterm },
	}
	comps := &PTYComponents{VirtualTerminal: vterm}
	pod.IO = NewPTYPodIO(podKey, comps, PTYPodIODeps{
		GetOrCreateDetector: pod.GetOrCreateStateDetector,
		SubscribeState:      pod.SubscribeStateChange,
		UnsubscribeState:    pod.UnsubscribeStateChange,
		GetPTYError:         pod.GetPTYError,
	})
	return pod
}
