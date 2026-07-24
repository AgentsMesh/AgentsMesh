package runner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
	"github.com/stretchr/testify/assert"
)

type recoveryOwnershipStore struct {
	*InMemoryPodStore
	dropAfterPut bool
	lastPut      *Pod
}

func (s *recoveryOwnershipStore) Put(podKey string, pod *Pod) {
	s.lastPut = pod
	s.InMemoryPodStore.Put(podKey, pod)
}

func (s *recoveryOwnershipStore) Get(podKey string) (*Pod, bool) {
	if s.dropAfterPut && s.lastPut != nil && s.lastPut.PodKey == podKey {
		s.dropAfterPut = false
		s.InMemoryPodStore.DeleteIf(podKey, s.lastPut)
		return nil, false
	}
	return s.InMemoryPodStore.Get(podKey)
}

type recoveryStartFailureIO struct {
	stubPodIO
	startErr     error
	stopCalled   bool
	teardownCall bool
}

func (i *recoveryStartFailureIO) Start() error { return i.startErr }
func (i *recoveryStartFailureIO) Stop()        { i.stopCalled = true }
func (i *recoveryStartFailureIO) Teardown() string {
	i.teardownCall = true
	return ""
}

// --- recoverDaemonSessions guard tests ---

func TestRecoverDaemonSessions_NilManager(t *testing.T) {
	r, _ := NewTestRunner(t)

	// Ensure podDaemonManager is nil (default from NewTestRunner).
	r.podDaemonManager = nil

	// Must return immediately without panic.
	r.recoverDaemonSessions()

	assert.Equal(t, 0, r.podStore.Count(), "no pods should be added when manager is nil")
}

func TestRecoverDaemonSessions_ManagerScanPaths(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		r, _ := NewTestRunner(t)
		manager, err := poddaemon.NewPodDaemonManager(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		r.podDaemonManager = manager
		r.recoverDaemonSessions()
		if r.podStore.Count() != 0 {
			t.Fatal("empty daemon directory recovered a pod")
		}
	})

	t.Run("directory read error", func(t *testing.T) {
		r, _ := NewTestRunner(t)
		path := filepath.Join(t.TempDir(), "not-a-directory")
		if err := os.WriteFile(path, []byte("file"), 0600); err != nil {
			t.Fatal(err)
		}
		manager, err := poddaemon.NewPodDaemonManager(path)
		if err != nil {
			t.Fatal(err)
		}
		r.podDaemonManager = manager
		r.recoverDaemonSessions()
		if r.podStore.Count() != 0 {
			t.Fatal("failed daemon scan recovered a pod")
		}
	})
}

func TestRecoverDaemonSessionsCleansFailedAttach(t *testing.T) {
	sandboxes := t.TempDir()
	sandbox := filepath.Join(sandboxes, "failed")
	if err := os.MkdirAll(sandbox, 0700); err != nil {
		t.Fatal(err)
	}
	state := &poddaemon.PodDaemonState{
		PodKey: "failed", SandboxPath: sandbox, Command: "fake",
		IPCAddr: "127.0.0.1:1", AuthToken: strings.Repeat("ab", 32),
		Cols: 80, Rows: 24,
	}
	if err := poddaemon.SaveState(state); err != nil {
		t.Fatal(err)
	}
	manager, err := poddaemon.NewPodDaemonManager(sandboxes)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := NewTestRunner(t)
	r.podDaemonManager = manager

	r.recoverDaemonSessions()

	if _, err := os.Stat(poddaemon.StatePath(sandbox)); !os.IsNotExist(err) {
		t.Fatalf("failed recovery retained state file: %v", err)
	}
}

func TestRecoverSingleSessionPublishesCompleteRuntime(t *testing.T) {
	addr, serverDone := startRecoveryDaemon(t)
	sandboxes := t.TempDir()
	manager, err := poddaemon.NewPodDaemonManager(sandboxes)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := NewTestRunner(t)
	r.podDaemonManager = manager
	state := &poddaemon.PodDaemonState{
		PodKey: "recovered", Agent: "claude", Command: "fake", WorkDir: t.TempDir(),
		SandboxPath: filepath.Join(sandboxes, "recovered"), IPCAddr: addr,
		AuthToken: strings.Repeat("ab", 32), Cols: 100, Rows: 30,
		VTHistoryLimit: 200, StartedAt: time.Now(), RepositoryURL: "repo", Branch: "main",
	}

	pod, err := r.recoverSingleSession(state)
	if err != nil {
		t.Fatalf("recoverSingleSession: %v", err)
	}
	if current, ok := r.podStore.Get(state.PodKey); !ok || current != pod {
		t.Fatal("recovered pod was not atomically published")
	}
	if pod.GetStatus() != PodStatusRunning || !pod.installedRuntime().valid() {
		t.Fatalf("recovered pod runtime incomplete: status=%s runtime=%+v", pod.GetStatus(), pod.installedRuntime())
	}
	pid := 0
	if !pod.WithActiveIO(func(io PodIO) { pid = io.GetPID() }) || pid != 321 {
		t.Fatalf("recovered PTY pid = %d, want 321", pid)
	}

	runtime := pod.installedRuntime()
	runtime.IO.Detach()
	runtime.IO.Teardown()
	waitRecoveryServer(t, serverDone)
}

func TestRecoverDaemonSessionsScansAndPublishesLiveSession(t *testing.T) {
	addr, serverDone := startRecoveryDaemon(t)
	sandboxes := t.TempDir()
	sandbox := filepath.Join(sandboxes, "scanned-recovery")
	if err := os.MkdirAll(sandbox, 0700); err != nil {
		t.Fatal(err)
	}
	state := &poddaemon.PodDaemonState{
		PodKey: "scanned-recovery", Agent: "claude", Command: "fake", WorkDir: t.TempDir(),
		SandboxPath: sandbox, IPCAddr: addr, AuthToken: strings.Repeat("ab", 32),
		Cols: 100, Rows: 30, VTHistoryLimit: 200, StartedAt: time.Now(),
	}
	if err := poddaemon.SaveState(state); err != nil {
		t.Fatal(err)
	}
	manager, err := poddaemon.NewPodDaemonManager(sandboxes)
	if err != nil {
		t.Fatal(err)
	}
	runner, _ := NewTestRunner(t)
	runner.podDaemonManager = manager

	runner.recoverDaemonSessions()
	pod, found := runner.podStore.Get(state.PodKey)
	if !found || pod.GetStatus() != PodStatusRunning {
		t.Fatalf("scanned live session was not published: found=%v pod=%+v", found, pod)
	}
	runtime := pod.installedRuntime()
	runtime.IO.Detach()
	runtime.IO.Teardown()
	waitRecoveryServer(t, serverDone)
}

func TestRecoverDaemonSessionsDoesNotReportSessionThatLostStoreOwnership(t *testing.T) {
	addr, serverDone := startRecoveryDaemon(t)
	sandboxes := t.TempDir()
	sandbox := filepath.Join(sandboxes, "lost-owner")
	if err := os.MkdirAll(sandbox, 0700); err != nil {
		t.Fatal(err)
	}
	state := &poddaemon.PodDaemonState{
		PodKey: "lost-owner", Agent: "claude", Command: "fake", WorkDir: t.TempDir(),
		SandboxPath: sandbox, IPCAddr: addr, AuthToken: strings.Repeat("ab", 32),
		Cols: 80, Rows: 24, VTHistoryLimit: 100, StartedAt: time.Now(),
	}
	if err := poddaemon.SaveState(state); err != nil {
		t.Fatal(err)
	}
	manager, err := poddaemon.NewPodDaemonManager(sandboxes)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := NewTestRunner(t)
	store := &recoveryOwnershipStore{InMemoryPodStore: NewInMemoryPodStore(), dropAfterPut: true}
	r.podStore = store
	r.messageHandler.podStore = store
	r.podDaemonManager = manager

	r.recoverDaemonSessions()
	if store.lastPut == nil {
		t.Fatal("recovery did not publish a candidate pod")
	}
	if _, owned := store.InMemoryPodStore.Get(state.PodKey); owned {
		t.Fatal("recovery restored ownership after the commit check rejected it")
	}
	runtime := store.lastPut.installedRuntime()
	runtime.IO.Detach()
	runtime.IO.Teardown()
	waitRecoveryServer(t, serverDone)
}

func TestStartRecoveredRuntimeRollsBackFailedIO(t *testing.T) {
	store := NewInMemoryPodStore()
	pod := &Pod{PodKey: "start-failure"}
	startErr := errors.New("injected recovery start failure")
	io := &recoveryStartFailureIO{startErr: startErr}
	runtime := PodRuntime{IO: io, Relay: NewPTYPodRelay(pod.PodKey, io, nil, nil)}
	store.Put(pod.PodKey, pod)
	pod.lifecycleMu.Lock()
	attachedClosed := false

	err := startRecoveredRuntime(store, pod, runtime, func() { attachedClosed = true })
	if err == nil || !strings.Contains(err.Error(), startErr.Error()) {
		t.Fatalf("startRecoveredRuntime error = %v, want %v", err, startErr)
	}
	if _, owned := store.Get(pod.PodKey); owned {
		t.Fatal("failed recovered runtime remained in the pod store")
	}
	if !io.stopCalled || !io.teardownCall || !attachedClosed {
		t.Fatalf("rollback incomplete: stop=%v teardown=%v attached_closed=%v",
			io.stopCalled, io.teardownCall, attachedClosed)
	}
	if !pod.lifecycleMu.TryLock() {
		t.Fatal("failed recovery retained pod lifecycle lock")
	}
	pod.lifecycleMu.Unlock()
}

func TestRecoverSingleSessionClosesAttachWhenTerminalCreationFails(t *testing.T) {
	addr, serverDone := startRecoveryDaemon(t)
	manager, err := poddaemon.NewPodDaemonManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r, _ := NewTestRunner(t)
	r.podDaemonManager = manager
	state := &poddaemon.PodDaemonState{
		PodKey: "invalid", Command: "", IPCAddr: addr,
		AuthToken: strings.Repeat("ab", 32), Cols: 80, Rows: 24,
	}

	if _, err := r.recoverSingleSession(state); err == nil || !strings.Contains(err.Error(), "create terminal") {
		t.Fatalf("recoverSingleSession error = %v, want terminal creation failure", err)
	}
	waitRecoveryServer(t, serverDone)
}

func startRecoveryDaemon(t *testing.T) (string, <-chan error) {
	t.Helper()
	listener, err := poddaemon.Listen()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	done := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		messageType, _, err := poddaemon.ReadMessage(conn)
		if err != nil {
			done <- err
			return
		}
		if messageType != poddaemon.MsgAttach {
			done <- fmt.Errorf("first daemon message = %#x, want attach", messageType)
			return
		}
		ack := []byte(`{"pid":321,"cols":100,"rows":30,"alive":true}`)
		if err := poddaemon.WriteMessage(conn, poddaemon.MsgAttachAck, ack); err != nil {
			done <- err
			return
		}
		for {
			messageType, _, err = poddaemon.ReadMessage(conn)
			if err != nil {
				done <- nil
				return
			}
			if messageType == poddaemon.MsgDetach {
				done <- nil
				return
			}
		}
	}()
	return listener.Addr().String(), done
}

func waitRecoveryServer(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("recovery daemon did not observe detach")
	}
}
