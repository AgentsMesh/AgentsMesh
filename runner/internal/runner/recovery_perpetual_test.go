package runner

import (
	"os"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

func TestRestartDeadPerpetualDaemonRejectsInvalidState(t *testing.T) {
	runner, _ := NewTestRunner(t)
	manager, err := poddaemon.NewPodDaemonManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runner.podDaemonManager = manager

	_, err = runner.restartDeadPerpetualDaemon(&poddaemon.PodDaemonState{
		PodKey: "invalid-restart", Command: "ignored", Perpetual: true,
	})
	if err == nil || !strings.Contains(err.Error(), "create daemon session") {
		t.Fatalf("restartDeadPerpetualDaemon error = %v, want create failure", err)
	}
}

func TestRestartDeadPerpetualDaemonRebuildsRuntime(t *testing.T) {
	runner, _ := NewTestRunner(t)
	manager, err := poddaemon.NewPodDaemonManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runner.podDaemonManager = manager
	command, args := testutil.SleepCommand(30)
	sandbox := t.TempDir()
	state := &poddaemon.PodDaemonState{
		PodKey:         "restarted-recovery",
		Agent:          "coverage-agent",
		Command:        command,
		Args:           args,
		WorkDir:        sandbox,
		Env:            os.Environ(),
		Cols:           100,
		Rows:           30,
		SandboxPath:    sandbox,
		RepositoryURL:  "https://example.invalid/recovery.git",
		Branch:         "recovery-branch",
		TicketSlug:     "RECOVERY-1",
		VTHistoryLimit: 200,
		Perpetual:      true,
	}

	pod, err := runner.restartDeadPerpetualDaemon(state)
	if err != nil {
		t.Fatalf("restartDeadPerpetualDaemon: %v", err)
	}
	t.Cleanup(func() {
		if runtime := pod.installedRuntime(); runtime.valid() {
			runtime.IO.Stop()
			runtime.IO.Teardown()
		}
		_ = manager.CleanupSession(sandbox)
	})

	if pod.GetStatus() != PodStatusRunning || !pod.installedRuntime().valid() {
		t.Fatalf("restarted recovery runtime incomplete: status=%s runtime=%+v",
			pod.GetStatus(), pod.installedRuntime())
	}
	if pod.Agent != state.Agent || pod.RepositoryURL != state.RepositoryURL || pod.Branch != state.Branch {
		t.Fatalf("restarted recovery lost metadata: %+v", pod)
	}
}
