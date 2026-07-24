package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	runnerrelay "github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

type coverageLocalBroker struct {
	messageHandlers map[byte]func([]byte)
	requestHandlers map[byte]runnerrelay.RequestHandler
	unregistered    []string
	connected       bool
	sent            int
}

func newCoverageLocalBroker() *coverageLocalBroker {
	return &coverageLocalBroker{
		messageHandlers: make(map[byte]func([]byte)),
		requestHandlers: make(map[byte]runnerrelay.RequestHandler),
	}
}

func (b *coverageLocalBroker) RegisterPod(string, string) {}

func (b *coverageLocalBroker) UnregisterPod(podKey string) {
	b.unregistered = append(b.unregistered, podKey)
}

func (b *coverageLocalBroker) SetMessageHandler(_ string, msgType byte, handler func([]byte)) {
	b.messageHandlers[msgType] = handler
}

func (b *coverageLocalBroker) SetRequestHandler(_ string, msgType byte, handler runnerrelay.RequestHandler) {
	b.requestHandlers[msgType] = handler
}

func (b *coverageLocalBroker) Send(string, byte, []byte) error {
	b.sent++
	return nil
}

func (b *coverageLocalBroker) Flush(context.Context, string) error { return nil }
func (b *coverageLocalBroker) IsPodConnected(string) bool          { return b.connected }
func (b *coverageLocalBroker) URL() string                         { return "ws://coverage-local" }

func TestPodActivityRejectsNilAndRetiredAcquisitions(t *testing.T) {
	var missing *podActivity
	_, ok := missing.acquire()
	require.False(t, ok)

	activity := newPodActivity()
	drained := activity.retire()
	_, ok = activity.acquire()
	require.False(t, ok)
	select {
	case <-drained:
	default:
		t.Fatal("idle retired activity did not drain")
	}
	waitPodActivities(nil, drained)
	(RelayHandlerGeneration{}).retire()
}

func TestRelayGenerationPublicGuardsAndCurrentOwner(t *testing.T) {
	io := &stubPodIO{}
	modeRelay := NewPTYPodRelay("generation", io, nil, nil)
	owner := runnerrelay.NewMockClient("wss://owner.example.com")
	other := runnerrelay.NewMockClient("wss://other.example.com")
	pod := &Pod{
		PodKey:        "generation",
		Status:        PodStatusRunning,
		IO:            io,
		Relay:         modeRelay,
		RelayClient:   owner,
		cloudActivity: newPodActivity(),
	}

	require.False(t, pod.WithCurrentRelayClient(nil, func(PodRelay) {}))
	require.False(t, pod.WithCurrentRelayClient(owner, nil))
	require.False(t, pod.WithCurrentRelayClient(other, func(PodRelay) {}))
	called := false
	require.True(t, pod.WithCurrentRelayClient(owner, func(got PodRelay) {
		called = true
		require.Same(t, modeRelay, got)
	}))
	require.True(t, called)

	pod.cloudActivity.retire()
	require.False(t, pod.WithCurrentRelayClient(owner, func(PodRelay) {
		t.Fatal("retired cloud generation executed")
	}))

	ticket, accepting := pod.RelayLifecycle()
	require.True(t, accepting)
	_, prepared := pod.WithRelayHandlerGeneration(ticket, nil, func(PodRelay, RelayInboundGuard) {})
	require.False(t, prepared)
	_, prepared = pod.WithRelayHandlerGeneration(ticket, owner, nil)
	require.False(t, prepared)
	require.False(t, pod.withRelayInbound(owner, 0, 0, newPodActivity(), nil))

	pod.SetStatus(PodStatusStopped)
	require.False(t, pod.WithRelayLifecycle(ticket, func(PodRelay) {
		t.Fatal("stopped pod accepted relay lifecycle work")
	}))
	pod.SetStatus(PodStatusRunning)
	pod.relayRuntimeBlocked = true
	require.False(t, pod.WithRelayLifecycle(ticket, func(PodRelay) {
		t.Fatal("blocked pod accepted relay lifecycle work")
	}))
}

func TestRelayInboundGuardRejectsMissingCallbacksAndLanes(t *testing.T) {
	var zero RelayInboundGuard
	require.False(t, zero.runCloud(func(RelayInboundContext) {
		t.Fatal("zero cloud guard executed callback")
	}))
	require.False(t, zero.runLocal(func(RelayInboundContext) {
		t.Fatal("zero local guard executed callback")
	}))

	guard := testRelayInboundGuard()
	require.False(t, guard.runCloud(nil))
	require.False(t, guard.runLocal(nil))
}

func TestRelayRuntimeTransitionRejectsInvalidAndStaleWork(t *testing.T) {
	for _, status := range []string{PodStatusStopped, PodStatusFailed} {
		pod := &Pod{Status: status}
		_, ok := pod.TryBeginRelayRuntimeTransition()
		require.False(t, ok)
	}
	blocked := &Pod{Status: PodStatusRunning, relayRuntimeBlocked: true}
	_, ok := blocked.TryBeginRelayRuntimeTransition()
	require.False(t, ok)

	pod := &Pod{PodKey: "runtime", Status: PodStatusRunning}
	require.False(t, pod.installRuntime(PodRuntime{}))
	require.False(t, pod.installRuntimeDuringTransition(0, PodRuntime{}))
	require.False(t, pod.WithActiveIO(nil))
	require.False(t, pod.clearRuntimeDuringTransition(1))

	lease := pod.BeginRelayRuntimeTransition()
	require.True(t, pod.RelayRuntimeTransitionCurrent(lease))
	require.False(t, pod.EndRelayRuntimeTransition(lease+1))
	require.False(t, pod.clearRuntimeDuringTransition(lease+1))

	io := &stubPodIO{}
	modeRelay := NewPTYPodRelay(pod.PodKey, io, nil, nil)
	runtime := PodRuntime{IO: io, Relay: modeRelay}
	require.False(t, pod.installRuntimeDuringTransition(lease+1, runtime))
	pod.SetStatus(PodStatusStopped)
	require.False(t, pod.installRuntimeDuringTransition(lease, runtime))
	require.False(t, pod.EndRelayRuntimeTransition(lease))
	require.False(t, pod.TryInstallRelayClient(nil, RelayHandlerGeneration{}))
	require.False(t, pod.ClearRelayClientIf(nil))
}

func TestPTYSnapshotDeliveryDefensiveBranches(t *testing.T) {
	empty := &PTYPodRelay{podKey: "snapshot-empty"}
	require.False(t, empty.deliverCloudSnapshot(false, nil))
	empty.sendSnapshotToLocal(nil)
	empty.sendSnapshotToLocal(func(byte, []byte) {
		t.Fatal("snapshot reply ran without components")
	})
	require.False(t, empty.deliverSnapshot("missing", false, snapshotDestination, nil))

	withoutVT := &PTYPodRelay{podKey: "snapshot-no-vt", components: &PTYComponents{}}
	require.False(t, withoutVT.deliverRequesterSnapshotLocked(func([]byte) error { return nil }))
	require.False(t, withoutVT.deliverSnapshot("missing", false, snapshotDestination, nil))

	vterm := vt.NewVirtualTerminal(80, 24, 100)
	vterm.Feed([]byte("authoritative state"))
	components := &PTYComponents{VirtualTerminal: vterm}
	podRelay := &PTYPodRelay{podKey: "snapshot", components: components}
	require.False(t, podRelay.deliverCloudSnapshotLocked(false, nil))
	require.False(t, podRelay.deliverRequesterSnapshotLocked(func([]byte) error {
		return errors.New("requester send failed")
	}))

	disconnected := runnerrelay.NewMockClient("wss://disconnected.example.com")
	require.False(t, podRelay.deliverCloudSnapshotLocked(false, disconnected))
	current := runnerrelay.NewMockClient("wss://current.example.com")
	current.SetConnected(true)
	expected := runnerrelay.NewMockClient("wss://expected.example.com")
	expected.SetConnected(true)
	podRelay.setCloudClient(current)
	require.False(t, podRelay.deliverCloudSnapshotLocked(false, expected))

	var replyType byte
	podRelay.sendSnapshotToLocal(func(msgType byte, data []byte) {
		replyType = msgType
		require.NotEmpty(t, data)
	})
	require.EqualValues(t, runnerrelay.MsgTypeSnapshot, replyType)

	stoppedAgg := aggregator.NewSmartAggregator(nil)
	stoppedAgg.Stop()
	stoppedRelay := &PTYPodRelay{
		podKey:     "stopped-aggregator",
		components: &PTYComponents{VirtualTerminal: vterm, Aggregator: stoppedAgg},
	}
	require.False(t, stoppedRelay.deliverSnapshot(
		"missing", false, snapshotDestination, func([]byte) error { return nil },
	))

	client := runnerrelay.NewMockClient("wss://boundary.example.com")
	client.SetConnected(true)
	boundaryAgg := aggregator.NewSmartAggregator(nil)
	boundaryAgg.SetOutputDestination(
		aggregator.OutputDestinationCloud,
		&cloudOutputWriter{client: client},
	)
	noSnapshot := &PTYPodRelay{
		podKey:     "boundary-no-snapshot",
		components: &PTYComponents{Aggregator: boundaryAgg},
	}
	require.False(t, noSnapshot.deliverSnapshot(
		aggregator.OutputDestinationCloud,
		false,
		snapshotDestination,
		func([]byte) error { return nil },
	))
	boundaryAgg.Stop()

	commitAgg := aggregator.NewSmartAggregator(nil)
	commitAgg.SetOutputDestination(
		aggregator.OutputDestinationCloud,
		&cloudOutputWriter{client: client},
	)
	commitRelay := &PTYPodRelay{
		podKey: "boundary-commit",
		components: &PTYComponents{
			VirtualTerminal: vterm,
			Aggregator:      commitAgg,
		},
	}
	require.False(t, commitRelay.deliverSnapshot(
		aggregator.OutputDestinationCloud,
		false,
		snapshotDestination,
		func([]byte) error {
			commitAgg.Stop()
			return nil
		},
	))
}

func TestPTYOutputRepairHelperBranches(t *testing.T) {
	podRelay := &PTYPodRelay{podKey: "repair"}
	state, epoch := podRelay.repairState("unknown")
	require.Nil(t, state)
	require.Zero(t, epoch)
	podRelay.outputHealthHandler()(aggregator.OutputHealth{Destination: "unknown"})

	requestRepairEpoch(&podRelay.cloudRepair, 3)
	requestRepairEpoch(&podRelay.cloudRepair, 2)
	require.Equal(t, uint64(3), podRelay.cloudRepair.requestedEpoch.Load())
	require.Equal(t, 100*time.Millisecond, snapshotRepairBackoff(0))
	require.Equal(t, snapshotRepairMaxBackoff, snapshotRepairBackoff(100))
	require.Equal(t, uint64(1), podRelay.destinationEpoch(aggregator.OutputDestinationLocal))
	podRelay.cloudEpoch.Store(9)
	require.Equal(t, uint64(9), podRelay.destinationEpoch(aggregator.OutputDestinationCloud))
	require.True(t, podRelay.repairOutputDestination("unknown"))
	require.False(t, podRelay.outputDestinationDesynced(aggregator.OutputDestinationCloud))

	vterm := vt.NewVirtualTerminal(80, 24, 100)
	vterm.Feed([]byte("repair state"))
	podRelay.components = &PTYComponents{VirtualTerminal: vterm}
	require.False(t, podRelay.repairOutputDestination(aggregator.OutputDestinationLocal))
}

func TestACPPodRelayDelegatesLocalGenerationHandlers(t *testing.T) {
	local := newCoverageLocalBroker()
	cloud := runnerrelay.NewMockClient("wss://acp.example.com")
	cloud.SetConnected(true)
	var commands [][]byte
	podRelay := NewACPPodRelay("acp-local", nil, func(_ RelayInboundContext, payload []byte) {
		commands = append(commands, append([]byte(nil), payload...))
	}, local)
	podRelay.SetupHandlers(cloud, testRelayInboundGuard())

	command := local.messageHandlers[runnerrelay.MsgTypeAcpCommand]
	require.NotNil(t, command)
	command([]byte("local command"))
	require.Equal(t, [][]byte{[]byte("local command")}, commands)

	request := local.requestHandlers[runnerrelay.MsgTypeSnapshotRequest]
	require.NotNil(t, request)
	replies := 0
	request(nil, func(byte, []byte) { replies++ })
	require.Zero(t, replies, "nil ACP session has no snapshot")

	podRelay.RecoverSnapshot(cloud)
	require.Zero(t, cloud.CountSentByType(runnerrelay.MsgTypeAcpSnapshot))
}

func TestPTYPodRelayDelegatesLocalResizeThroughGenerationGuard(t *testing.T) {
	local := newCoverageLocalBroker()
	var gotCols, gotRows int
	io := &stubPodIOWithTerminal{onResize: func(cols, rows int) (bool, error) {
		gotCols, gotRows = cols, rows
		return true, nil
	}}
	podRelay := NewPTYPodRelay("pty-local-resize", io, &PTYComponents{}, local)
	podRelay.SetupHandlers(runnerrelay.NewMockClient("wss://cloud.example.com"), testRelayInboundGuard())

	resize := local.messageHandlers[runnerrelay.MsgTypeResize]
	require.NotNil(t, resize)
	resize(encodeResizePayload(132, 43))
	require.Equal(t, 132, gotCols)
	require.Equal(t, 43, gotRows)
}

func TestPodStoreDeleteIfRejectsMissingAndReplacedOwners(t *testing.T) {
	store := NewInMemoryPodStore()
	require.False(t, store.DeleteIf("missing", &Pod{}))
	current := &Pod{PodKey: "owned"}
	store.Put(current.PodKey, current)
	require.False(t, store.DeleteIf(current.PodKey, &Pod{PodKey: current.PodKey}))
	require.Same(t, current, store.Delete(current.PodKey))
}
