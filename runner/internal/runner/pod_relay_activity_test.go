package runner

import (
	"errors"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

type lifecycleReentrantPodIO struct {
	*mockPodIO
	pod     *Pod
	reentry chan string
}

func (i *lifecycleReentrantPodIO) reenterPodLifecycle(operation string) error {
	if !i.pod.WithActiveIO(func(PodIO) {}) {
		return errors.New("nested WithActiveIO was rejected")
	}
	if !i.pod.BroadcastRelayEvent(relay.MsgTypeAcpEvent, []byte(operation)) {
		return errors.New("nested relay broadcast was rejected")
	}
	i.reentry <- operation
	return nil
}

func (i *lifecycleReentrantPodIO) SendInput(text string) error {
	if err := i.reenterPodLifecycle("prompt"); err != nil {
		return err
	}
	return i.mockPodIO.SendInput(text)
}

func (i *lifecycleReentrantPodIO) SetPermissionMode(mode string) error {
	if err := i.reenterPodLifecycle("permission-mode"); err != nil {
		return err
	}
	return i.mockPodIO.SetPermissionMode(mode)
}

func TestInstalledACPInboundUsesAdmittedRuntimeWithoutLockReentry(t *testing.T) {
	const podKey = "guarded-acp-inbound"
	handler := newTestHandler()
	baseIO := &mockPodIO{}
	io := &lifecycleReentrantPodIO{mockPodIO: baseIO, reentry: make(chan string, 2)}
	pod := &Pod{PodKey: podKey, Status: PodStatusRunning, IO: io}
	io.pod = pod
	pod.Relay = NewACPPodRelay(podKey, nil, func(ctx RelayInboundContext, payload []byte) {
		handler.handleAcpRelayCommandInGeneration(pod, ctx, payload)
	}, nil)
	cloud := relay.NewMockClient("wss://relay.example.com")
	cloud.SetConnected(true)

	ticket, ok := pod.RelayLifecycle()
	generation, prepared := pod.WithRelayHandlerGeneration(ticket, cloud, func(modeRelay PodRelay, guard RelayInboundGuard) {
		modeRelay.SetupHandlers(cloud, guard)
	})
	if !ok || !prepared {
		t.Fatal("failed to prepare ACP inbound generation")
	}
	if !pod.TryInstallRelayClient(cloud, generation) {
		t.Fatal("failed to install ACP cloud owner")
	}

	commands := [][]byte{
		[]byte(`{"type":"prompt","prompt":"guarded"}`),
		[]byte(`{"type":"set_permission_mode","mode":"plan"}`),
	}
	for _, payload := range commands {
		done := make(chan struct{})
		go func(payload []byte) {
			cloud.SimulateMessage(relay.MsgTypeAcpCommand, payload)
			close(done)
		}(payload)
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("ACP inbound callback deadlocked while synchronously re-entering the Pod lifecycle")
		}
	}

	for _, want := range []string{"prompt", "permission-mode"} {
		select {
		case got := <-io.reentry:
			if got != want {
				t.Fatalf("lifecycle reentry = %q, want %q", got, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("missing %s lifecycle reentry", want)
		}
	}
	baseIO.mu.Lock()
	defer baseIO.mu.Unlock()
	if len(baseIO.inputs) != 1 || baseIO.inputs[0] != "guarded" {
		t.Fatalf("ACP inbound inputs = %v, want [guarded]", baseIO.inputs)
	}
	if baseIO.permMode != "plan" {
		t.Fatalf("ACP permission mode = %q, want plan", baseIO.permMode)
	}
}

func TestRelayRuntimeTransitionWaitsForActiveIOLease(t *testing.T) {
	const podKey = "runtime-activity-lease"
	oldIO := &stubPodIO{}
	pod := &Pod{PodKey: podKey, Status: PodStatusRunning, IO: oldIO}
	pod.Relay = NewPTYPodRelay(podKey, oldIO, nil, nil)

	callbackEntered := make(chan struct{})
	callbackRelease := make(chan struct{})
	callbackDone := make(chan bool, 1)
	go func() {
		callbackDone <- pod.WithActiveIO(func(io PodIO) {
			if io != oldIO {
				t.Errorf("active callback received %T, want old IO", io)
			}
			close(callbackEntered)
			<-callbackRelease
		})
	}()
	select {
	case <-callbackEntered:
	case <-time.After(2 * time.Second):
		t.Fatal("WithActiveIO callback did not acquire the runtime activity lease")
	}

	transitionReady := make(chan RelayRuntimeTransition, 1)
	go func() {
		transitionReady <- pod.BeginRelayRuntimeTransition()
	}()
	deadline := time.Now().Add(2 * time.Second)
	for {
		pod.relayMu.RLock()
		blocked := pod.relayRuntimeBlocked
		pod.relayMu.RUnlock()
		if blocked {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("runtime transition could not invalidate while callback executed outside the lifecycle lock")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case <-transitionReady:
		t.Fatal("runtime transition returned before the active IO lease drained")
	default:
	}

	close(callbackRelease)
	select {
	case admitted := <-callbackDone:
		if !admitted {
			t.Fatal("active IO callback was not admitted")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("active IO callback did not finish")
	}

	var transition RelayRuntimeTransition
	select {
	case transition = <-transitionReady:
	case <-time.After(2 * time.Second):
		t.Fatal("runtime transition did not finish after the activity lease drained")
	}
	newIO := &stubPodIO{}
	newRelay := NewPTYPodRelay(podKey, newIO, nil, nil)
	if !pod.installRuntimeDuringTransition(transition, PodRuntime{IO: newIO, Relay: newRelay}) {
		t.Fatal("failed to install replacement runtime")
	}
	if !pod.EndRelayRuntimeTransition(transition) {
		t.Fatal("failed to commit replacement runtime")
	}
	if !pod.WithActiveIO(func(io PodIO) {
		if io != newIO {
			t.Errorf("replacement callback received %T, want new IO", io)
		}
	}) {
		t.Fatal("replacement runtime did not accept IO callbacks")
	}
}
