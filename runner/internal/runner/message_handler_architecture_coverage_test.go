package runner

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	runnerrelay "github.com/anthropics/agentsmesh/runner/internal/relay"
)

type callbackCapturingRelayClient struct {
	*runnerrelay.MockClient
	closeHandler     runnerrelay.CloseHandler
	reconnectHandler func()
}

func (c *callbackCapturingRelayClient) SetCloseHandler(handler runnerrelay.CloseHandler) {
	c.closeHandler = handler
	c.MockClient.SetCloseHandler(handler)
}

func (c *callbackCapturingRelayClient) SetReconnectHandler(handler func()) {
	c.reconnectHandler = handler
	c.MockClient.SetReconnectHandler(handler)
}

func TestMessageHandlerControlPlaneFailurePaths(t *testing.T) {
	store := NewInMemoryPodStore()
	conn := client.NewMockConnection()
	baseRunner := &Runner{cfg: &config.Config{}, podStore: store}
	handler := NewRunnerMessageHandler(baseRunner, store, conn)

	io := &mcpCoveragePodIO{
		inputErr:    errors.New("input failed"),
		snapshotErr: errors.New("snapshot failed"),
	}
	pod := &Pod{
		PodKey:          "control-plane-errors",
		InteractionMode: InteractionModePTY,
		Status:          PodStatusRunning,
		IO:              io,
	}
	store.Put(pod.PodKey, pod)

	err := handler.OnPodInput(client.PodInputRequest{PodKey: pod.PodKey, Data: []byte("input")})
	require.ErrorContains(t, err, "input failed")

	err = handler.OnObservePod(client.ObservePodRequest{
		RequestID: "snapshot-error",
		PodKey:    pod.PodKey,
		Lines:     10,
	})
	require.NoError(t, err, "snapshot failure is returned in the observe response")
	events := conn.GetEvents()
	require.NotEmpty(t, events)
	payload, ok := events[len(events)-1].Data.(map[string]interface{})
	require.True(t, ok)
	require.Equal(t, "snapshot failed", payload["error"])

	pod.SetStatus(PodStatusStopped)
	err = handler.OnObservePod(client.ObservePodRequest{
		RequestID: "unavailable",
		PodKey:    pod.PodKey,
	})
	require.NoError(t, err, "unavailable IO is returned in the observe response")

	pod.SetStatus(PodStatusRunning)
	err = handler.OnSendPrompt(&runnerv1.SendPromptCommand{
		PodKey: pod.PodKey,
		Prompt: "prompt",
	})
	require.ErrorContains(t, err, "input failed")
}

func TestRelayCallbacksRemainBoundToCurrentOwner(t *testing.T) {
	store := NewInMemoryPodStore()
	conn := client.NewMockConnection()
	baseRunner := &Runner{cfg: &config.Config{}, podStore: store}
	handler := NewRunnerMessageHandler(baseRunner, store, conn)

	io := &stubPodIO{}
	modeRelay := NewPTYPodRelay("callback-owner", io, nil, nil)
	relayClient := &callbackCapturingRelayClient{
		MockClient: runnerrelay.NewMockClient("wss://callback.example.com"),
	}
	relayClient.SetConnected(true)
	pod := &Pod{
		PodKey:        "callback-owner",
		Status:        PodStatusRunning,
		IO:            io,
		Relay:         modeRelay,
		RelayClient:   relayClient,
		cloudActivity: newPodActivity(),
	}
	store.Put(pod.PodKey, pod)

	handler.setupRelayClientHandlers(
		relayClient,
		pod,
		modeRelay,
		testRelayInboundGuard(),
		client.SubscribePodRequest{PodKey: pod.PodKey},
	)
	require.NotNil(t, relayClient.reconnectHandler)
	require.NotNil(t, relayClient.closeHandler)
	relayClient.reconnectHandler()
	require.Same(t, relayClient, pod.GetRelayClient())
	relayClient.closeHandler()
	require.Nil(t, pod.GetRelayClient())
}

func TestUnsubscribeMissingPodRemovesLocalRegistration(t *testing.T) {
	store := NewInMemoryPodStore()
	local := newCoverageLocalBroker()
	baseRunner := &Runner{cfg: &config.Config{}, podStore: store}
	handler := NewRunnerMessageHandler(
		gatedLocalHandlerContext{MessageHandlerContext: baseRunner, local: local},
		store,
		client.NewMockConnection(),
	)

	require.NoError(t, handler.OnUnsubscribePod(client.UnsubscribePodRequest{PodKey: "missing"}))
	require.Equal(t, []string{"missing"}, local.unregistered)
}
