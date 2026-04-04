//go:build integration

package runner

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// buildRelayTestPod creates a PTY pod and starts its terminal for relay routing tests.
func buildRelayTestPod(t *testing.T, agentfile string, opts ...func(*runnerv1.CreatePodCommand)) *Pod {
	t.Helper()
	runner := &Runner{cfg: &config.Config{WorkspaceRoot: t.TempDir()}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:          "relay-" + t.Name(),
		AgentfileSource: agentfile,
	}
	for _, o := range opts {
		o(cmd)
	}
	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).WithPtySize(80, 24).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	comps := testPTYComponents(pod)
	comps.Terminal.SetOutputHandler(NewPTYOutputHandler(pod.PodKey, comps, pod.NotifyStateDetectorWithScreen))
	if err := comps.Terminal.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	return pod
}

// newRelayHandler creates a RunnerMessageHandler with an injected mock relay factory.
// Returns the handler and a function to retrieve the last created mock client.
func newRelayHandler(t *testing.T, store PodStore) (*RunnerMessageHandler, func() *relay.MockClient) {
	t.Helper()
	mockConn := client.NewMockConnection()
	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	var mu sync.Mutex
	var lastMock *relay.MockClient
	handler.relayClientFactory = func(url, podKey, token string, _ *slog.Logger) relay.RelayClient {
		mc := relay.NewMockClient(url)
		mu.Lock()
		lastMock = mc
		mu.Unlock()
		return mc
	}
	getMock := func() *relay.MockClient {
		mu.Lock()
		defer mu.Unlock()
		return lastMock
	}
	return handler, getMock
}

func TestRelayRouting_SubscribeUnsubscribe_Integration(t *testing.T) {
	af := "AGENT sleep\nMODE pty\nPROMPT_POSITION prepend\n"
	pod := buildRelayTestPod(t, af, func(c *runnerv1.CreatePodCommand) {
		c.LaunchArgs = []string{"10"}
	})
	defer testPTYComponents(pod).Terminal.Stop()

	store := NewInMemoryPodStore()
	store.Put(pod.PodKey, pod)
	handler, getMock := newRelayHandler(t, store)

	// Subscribe
	err := handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey: pod.PodKey, RelayURL: "wss://relay.test", RunnerToken: "tok",
	})
	if err != nil {
		t.Fatalf("OnSubscribePod failed: %v", err)
	}

	mc := getMock()
	if mc == nil {
		t.Fatal("mock relay client was not created")
	}
	if !mc.ConnectCalled {
		t.Error("Connect() was not called")
	}
	if !mc.StartCalled {
		t.Error("Start() was not called")
	}

	// Snapshot should have been sent (MsgTypeSnapshot = 0x01)
	if mc.CountSentByType(relay.MsgTypeSnapshot) == 0 {
		t.Error("expected at least one snapshot message sent via relay")
	}

	// Unsubscribe
	err = handler.OnUnsubscribePod(client.UnsubscribePodRequest{PodKey: pod.PodKey})
	if err != nil {
		t.Fatalf("OnUnsubscribePod failed: %v", err)
	}
	if !mc.StopCalled {
		t.Error("Stop() was not called on unsubscribe")
	}
	if pod.GetRelayClient() != nil {
		t.Error("relay client should be nil after unsubscribe")
	}
}

func TestRelayRouting_TerminalOutputToRelay_Integration(t *testing.T) {
	af := "AGENT echo\nMODE pty\nPROMPT_POSITION prepend\n"
	pod := buildRelayTestPod(t, af, func(c *runnerv1.CreatePodCommand) {
		c.InitialPrompt = "relay-test-marker"
	})
	defer testPTYComponents(pod).Terminal.Stop()

	store := NewInMemoryPodStore()
	store.Put(pod.PodKey, pod)
	handler, getMock := newRelayHandler(t, store)

	err := handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey: pod.PodKey, RelayURL: "wss://relay.test", RunnerToken: "tok",
	})
	if err != nil {
		t.Fatalf("OnSubscribePod failed: %v", err)
	}
	mc := getMock()

	// Wait for echo output to flow through aggregator -> relay
	deadline := time.After(5 * time.Second)
	for {
		if mc.CountSentByType(relay.MsgTypeOutput) > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout: no output messages sent to relay")
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Verify payload contains the marker string
	var found bool
	for _, msg := range mc.SentMessages {
		if msg.Type == relay.MsgTypeOutput && strings.Contains(string(msg.Payload), "relay-test-marker") {
			found = true
			break
		}
	}
	if !found {
		t.Error("relay output did not contain 'relay-test-marker'")
	}
}

func TestRelayRouting_RelayInputToPod_Integration(t *testing.T) {
	pod := buildRelayTestPod(t, "AGENT cat\nMODE pty\nPROMPT_POSITION prepend\n")
	defer testPTYComponents(pod).Terminal.Stop()

	store := NewInMemoryPodStore()
	store.Put(pod.PodKey, pod)
	handler, getMock := newRelayHandler(t, store)

	err := handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey: pod.PodKey, RelayURL: "wss://relay.test", RunnerToken: "tok",
	})
	if err != nil {
		t.Fatalf("OnSubscribePod failed: %v", err)
	}
	mc := getMock()

	// Simulate browser sending input via relay (MsgTypeInput handler)
	mc.SimulateMessage(relay.MsgTypeInput, []byte("relay-input-marker\n"))

	// cat echoes back through VT; poll the VT screen for the marker
	deadline := time.After(5 * time.Second)
	for {
		snap := testPTYComponents(pod).VirtualTerminal.GetScreenSnapshot()
		if strings.Contains(snap, "relay-input-marker") {
			return // success
		}
		select {
		case <-deadline:
			t.Fatalf("VT snapshot never contained 'relay-input-marker', got: %q",
				testPTYComponents(pod).VirtualTerminal.GetScreenSnapshot())
		case <-time.After(50 * time.Millisecond):
		}
	}
}
