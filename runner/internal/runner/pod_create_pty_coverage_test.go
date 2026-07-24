package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/anthropics/agentsmesh/runner/internal/testutil"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestBuildPTYPodDirectCoverage(t *testing.T) {
	t.Run("trace OSC and logger are published with one runtime", func(t *testing.T) {
		conn := client.NewMockConnection()
		logRoot := t.TempDir()
		cmd := &runnerv1.CreatePodCommand{
			PodKey:        "direct-pty",
			Perpetual:     true,
			SandboxConfig: &runnerv1.SandboxConfig{HttpCloneUrl: "https://example.invalid/repo.git", TicketSlug: "T-42"},
		}
		builder := NewPodBuilder(PodBuilderDeps{
			Config:         &config.Config{},
			ProgressSender: conn,
		}).WithCommand(cmd).
			WithPtySize(132, 43).
			WithVirtualTerminalHistoryLimit(321).
			WithOSCHandler(func(int, []string) {} /* presence is the contract */).
			WithPTYLogging(logRoot)

		provider := sdktrace.NewTracerProvider()
		defer provider.Shutdown(context.Background())
		ctx, span := provider.Tracer("runner-coverage").Start(context.Background(), "build-pty")
		defer span.End()
		env := map[string]string{"COVERAGE_ENV": "present"}
		pod, err := builder.buildPTYPod(
			ctx, "", t.TempDir(), "feature/coverage", []string{"hello"}, env, "echo",
		)
		if err != nil {
			t.Fatal(err)
		}
		defer pod.IO.Teardown()

		if pod.PodKey != cmd.PodKey || pod.Branch != "feature/coverage" || !pod.Perpetual {
			t.Fatalf("pod metadata not preserved: %+v", pod)
		}
		if !strings.HasPrefix(env["TRACEPARENT"], "00-") {
			t.Fatalf("TRACEPARENT not injected: %q", env["TRACEPARENT"])
		}
		if !containsEnvAssignment(pod.LaunchEnv, "TRACEPARENT="+env["TRACEPARENT"]) ||
			!containsEnvAssignment(pod.LaunchEnv, "COVERAGE_ENV=present") {
			t.Fatalf("captured launch environment missing values: %v", pod.LaunchEnv)
		}
		runtime := pod.installedRuntime()
		if !runtime.valid() || runtime.vtProvider == nil {
			t.Fatalf("PTY runtime was not atomically installed: %+v", runtime)
		}
		components := runtime.IO.(*PTYPodIO).components
		if components.PTYLogger == nil || components.VirtualTerminal == nil || components.Aggregator == nil {
			t.Fatalf("PTY components incomplete: %+v", components)
		}
		if _, err := os.Stat(filepath.Join(logRoot, cmd.PodKey, "raw.log")); err != nil {
			t.Fatalf("PTY logger was not initialized: %v", err)
		}
		if !hasPerpetualCoverageEvent(conn.GetEvents(), client.MessageType("pod_init_progress")) {
			t.Fatal("direct PTY build did not publish progress")
		}
	})

	t.Run("logger failure is best effort", func(t *testing.T) {
		blockedBase := filepath.Join(t.TempDir(), "not-a-directory")
		if err := os.WriteFile(blockedBase, []byte("file"), 0o600); err != nil {
			t.Fatal(err)
		}
		builder := NewPodBuilder(PodBuilderDeps{Config: &config.Config{}}).
			WithCommand(&runnerv1.CreatePodCommand{PodKey: "logger-failure"}).
			WithPTYLogging(blockedBase)
		pod, err := builder.buildPTYPod(
			context.Background(), "", t.TempDir(), "", nil, map[string]string{}, "echo",
		)
		if err != nil {
			t.Fatal(err)
		}
		defer pod.IO.Teardown()
		if got := pod.IO.(*PTYPodIO).components.PTYLogger; got != nil {
			t.Fatalf("logger unexpectedly initialized under file path: %+v", got)
		}
	})

	t.Run("empty command removes owned sandbox", func(t *testing.T) {
		sandbox := filepath.Join(t.TempDir(), "owned-sandbox")
		if err := os.MkdirAll(sandbox, 0o755); err != nil {
			t.Fatal(err)
		}
		builder := NewPodBuilder(PodBuilderDeps{Config: &config.Config{}}).
			WithCommand(&runnerv1.CreatePodCommand{PodKey: "empty-command"})
		pod, err := builder.buildPTYPod(
			context.Background(), sandbox, sandbox, "", nil, map[string]string{}, "",
		)
		if pod != nil || err == nil {
			t.Fatalf("empty command = (%v, %v), want nil PodError", pod, err)
		}
		podErr, ok := err.(*client.PodError)
		if !ok || podErr.Code != client.ErrCodeCommandStart {
			t.Fatalf("empty command error = %#v", err)
		}
		if _, statErr := os.Stat(sandbox); !os.IsNotExist(statErr) {
			t.Fatalf("owned sandbox survived terminal construction failure: %v", statErr)
		}
	})
}

func TestBuildPTYPodUsesRealDaemonFactory(t *testing.T) {
	manager, err := poddaemon.NewPodDaemonManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sandbox := t.TempDir()
	workDir := t.TempDir()
	command, args := testutil.SleepCommand(30)
	builder := NewPodBuilder(PodBuilderDeps{
		Config:           &config.Config{},
		PodDaemonManager: manager,
	}).WithCommand(&runnerv1.CreatePodCommand{
		PodKey:    "daemon-factory",
		Perpetual: true,
		SandboxConfig: &runnerv1.SandboxConfig{
			HttpCloneUrl: "https://example.invalid/daemon.git",
			TicketSlug:   "DAEMON-1",
		},
	})
	pod, err := builder.buildPTYPod(
		context.Background(), sandbox, workDir, "daemon-branch", args, map[string]string{}, command,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := pod.IO.Start(); err != nil {
		t.Fatalf("daemon-backed PTY start: %v", err)
	}
	t.Cleanup(func() {
		pod.IO.Stop()
		pod.IO.Teardown()
		_ = poddaemon.DeleteState(sandbox)
	})
	if pod.IO.GetPID() <= 0 {
		t.Fatalf("daemon-backed PTY PID = %d", pod.IO.GetPID())
	}
	state, err := poddaemon.LoadState(sandbox)
	if err != nil {
		t.Fatalf("load daemon state: %v", err)
	}
	if state.PodKey != pod.PodKey || state.Command != command || state.WorkDir != workDir ||
		state.Cols != 80 || state.Rows != 24 || !state.Perpetual {
		t.Fatalf("daemon factory lost CreateOpts: %+v", state)
	}
}

func TestOnCreatePodCancellationOwnsBuiltRuntime(t *testing.T) {
	conn := newBlockingReadyConnection()
	store := NewInMemoryPodStore()
	workspaceRoot := t.TempDir()
	runner, err := New(RunnerDeps{
		Config: &config.Config{
			WorkspaceRoot: workspaceRoot,
			LogPTY:        true,
			LogPTYDir:     filepath.Join(workspaceRoot, "pty-logs"),
		},
		Connection: conn,
		PodStore:   store,
	})
	if err != nil {
		t.Fatal(err)
	}
	command, args := testutil.SleepCommand(30)
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "cancel-during-build",
		LaunchCommand: command,
		LaunchArgs:    args,
	}
	result := make(chan error, 1)
	go func() { result <- runner.messageHandler.OnCreatePod(cmd) }()

	select {
	case <-conn.ready:
	case <-time.After(5 * time.Second):
		t.Fatal("pod build never reached ready publication")
	}
	pending, ok := store.Get(cmd.PodKey)
	if !ok || pending.GetStatus() != PodStatusInitializing {
		t.Fatalf("pending owner missing during build: %+v", pending)
	}
	if !store.DeleteIf(cmd.PodKey, pending) {
		t.Fatal("failed to model termination of the pending owner")
	}
	close(conn.release)

	select {
	case err := <-result:
		if err == nil || !strings.Contains(err.Error(), "terminated during build") {
			t.Fatalf("OnCreatePod cancellation error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("OnCreatePod did not finish after cancellation")
	}
	if _, ok := store.Get(cmd.PodKey); ok {
		t.Fatal("canceled pod reclaimed store ownership")
	}
	sandbox := filepath.Join(workspaceRoot, "sandboxes", cmd.PodKey)
	if _, statErr := os.Stat(sandbox); !os.IsNotExist(statErr) {
		t.Fatalf("canceled pod sandbox survived: %v", statErr)
	}
}

func TestWireAndStartPTYPodOwnershipAndFailures(t *testing.T) {
	t.Run("store owner is required", func(t *testing.T) {
		runner, _ := NewTestRunner(t)
		pod := newPerpetualCoveragePod("not-owned-at-start", &perpetualCoverageIO{})
		err := runner.messageHandler.wireAndStartPTYPod(
			pod, &runnerv1.CreatePodCommand{PodKey: pod.PodKey}, 80, 24,
		)
		if err == nil || !strings.Contains(err.Error(), "terminated before terminal start") {
			t.Fatalf("owner error = %v", err)
		}
	})

	t.Run("complete runtime is required", func(t *testing.T) {
		runner, _ := NewTestRunner(t)
		pod := &Pod{PodKey: "missing-runtime", Status: PodStatusInitializing}
		runner.podStore.Put(pod.PodKey, pod)
		err := runner.messageHandler.wireAndStartPTYPod(
			pod, &runnerv1.CreatePodCommand{PodKey: pod.PodKey}, 80, 24,
		)
		if err == nil || !strings.Contains(err.Error(), "runtime not available") {
			t.Fatalf("runtime error = %v", err)
		}
	})

	t.Run("start error retires owned runtime and sandbox", func(t *testing.T) {
		runner, conn := NewTestRunner(t)
		io := &perpetualCoverageIO{startErr: fmt.Errorf("injected start failure")}
		pod := newPerpetualCoveragePod("start-failure", io)
		pod.SetStatus(PodStatusInitializing)
		pod.SandboxPath = t.TempDir()
		runner.podStore.Put(pod.PodKey, pod)
		err := runner.messageHandler.wireAndStartPTYPod(
			pod, &runnerv1.CreatePodCommand{PodKey: pod.PodKey}, 100, 40,
		)
		if err == nil || !strings.Contains(err.Error(), "failed to start terminal") {
			t.Fatalf("start error = %v", err)
		}
		if _, ok := runner.podStore.Get(pod.PodKey); ok {
			t.Fatal("start-failed pod retained store ownership")
		}
		if runtime := pod.installedRuntime(); runtime.IO != nil || runtime.Relay != nil {
			t.Fatalf("start-failed runtime survived cleanup: %+v", runtime)
		}
		if io.stops != 1 || io.teardowns != 1 || io.exitHandler == nil || io.ioErrorHandler == nil {
			t.Fatalf("start failure calls/handlers = stop:%d teardown:%d exit:%v ioerr:%v",
				io.stops, io.teardowns, io.exitHandler != nil, io.ioErrorHandler != nil)
		}
		if !hasPerpetualCoverageEvent(conn.GetEvents(), client.MessageType("error")) {
			t.Fatal("start failure did not publish a pod error")
		}
	})

	t.Run("successful commit registers monitor and MCP only while current", func(t *testing.T) {
		runner, conn := NewTestRunner(t)
		io := &perpetualCoverageIO{pid: 4242}
		pod := newPerpetualCoveragePod("successful-commit", io)
		pod.SetStatus(PodStatusInitializing)
		runner.podStore.Put(pod.PodKey, pod)
		cmd := &runnerv1.CreatePodCommand{PodKey: pod.PodKey, LaunchCommand: "coverage-agent"}
		if err := runner.messageHandler.wireAndStartPTYPod(pod, cmd, 111, 37); err != nil {
			t.Fatal(err)
		}
		if pod.GetStatus() != PodStatusRunning || io.starts != 1 {
			t.Fatalf("successful commit status/starts = %s/%d", pod.GetStatus(), io.starts)
		}
		if !hasPerpetualCoverageEvent(conn.GetEvents(), client.MsgTypePodCreated) {
			t.Fatal("successful commit did not publish pod_created")
		}
		runner.messageHandler.registerMCPIfCurrent(pod.PodKey, pod, cmd.LaunchCommand)
		runner.podStore.Put(pod.PodKey, &Pod{PodKey: pod.PodKey, Status: PodStatusRunning})
		runner.messageHandler.registerMCPIfCurrent(pod.PodKey, pod, cmd.LaunchCommand)
	})
}

func TestOnCreatePodACPStartFailure(t *testing.T) {
	runner, _ := NewTestRunner(t)
	err := runner.messageHandler.OnCreatePod(&runnerv1.CreatePodCommand{
		PodKey:          "acp-start-failure",
		LaunchCommand:   filepath.Join(t.TempDir(), "missing-acp-agent"),
		InteractionMode: InteractionModeACP,
	})
	if err == nil || !strings.Contains(err.Error(), "failed to start ACP agent") {
		t.Fatalf("ACP start error = %v", err)
	}
}

func containsEnvAssignment(env []string, assignment string) bool {
	for _, item := range env {
		if item == assignment {
			return true
		}
	}
	return false
}

type blockingReadyConnection struct {
	*client.MockConnection
	ready     chan struct{}
	release   chan struct{}
	readyOnce sync.Once
}

func newBlockingReadyConnection() *blockingReadyConnection {
	return &blockingReadyConnection{
		MockConnection: client.NewMockConnection(),
		ready:          make(chan struct{}),
		release:        make(chan struct{}),
	}
}

func (c *blockingReadyConnection) SendPodInitProgress(
	podKey, phase string,
	progress int32,
	message string,
) error {
	if phase == "ready" {
		c.readyOnce.Do(func() { close(c.ready) })
		<-c.release
	}
	return c.MockConnection.SendPodInitProgress(podKey, phase, progress, message)
}
