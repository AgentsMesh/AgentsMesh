package runner

import (
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	runnerrelay "github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

func TestPTYPodRelayNewCloudGenerationCannotLoseRepairEvent(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	vterm.Feed([]byte("current authoritative state"))
	agg := aggregator.NewSmartAggregator(nil, aggregator.WithSmartBaseDelay(time.Hour))
	podRelay := NewPTYPodRelay("pod-1", nil, &PTYComponents{
		VirtualTerminal: vterm,
		Aggregator:      agg,
	}, nil)

	oldClient := newSwitchableRelayClient("wss://old.example.com", false)
	podRelay.OnRelayConnected(oldClient)
	oldClient.Reset()
	oldClient.fail.Store(true)
	agg.Write([]byte("old generation delta"))
	// A repair can supersede the drain barrier's generation, so assert the
	// writer outcomes directly instead of treating a stale barrier as delivery.
	agg.Flush()
	waitForSendAttempt(t, oldClient, runnerrelay.MsgTypeOutput, true)
	waitForSendAttempt(t, oldClient, runnerrelay.MsgTypeSnapshot, true)

	newClient := newSwitchableRelayClient("wss://new.example.com", false)
	podRelay.OnRelayConnected(newClient)
	newClient.Reset()
	// Only the delta fails; the resulting recovery snapshot must succeed.
	newClient.failNext.Store(true)
	agg.Write([]byte("new generation delta"))
	agg.Flush()
	waitForSendAttempt(t, newClient, runnerrelay.MsgTypeOutput, true)
	waitForSendAttempt(t, newClient, runnerrelay.MsgTypeSnapshot, false)
	message := lastMessageOfType(t, newClient.MockClient, runnerrelay.MsgTypeSnapshot)
	var envelope struct {
		ResetAll bool `json:"reset_all"`
	}
	if err := json.Unmarshal(message.Payload, &envelope); err != nil {
		t.Fatalf("decode new-generation recovery: %v", err)
	}
	if !envelope.ResetAll {
		t.Fatal("new generation was repaired with requester-only snapshot")
	}
	podRelay.OnRelayDisconnected(newClient)
	agg.Stop()
}

func TestPTYPodRelayRepairContinuesPastInitialRetryWindow(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	vterm.Feed([]byte("eventual recovery state"))
	agg := aggregator.NewSmartAggregator(nil, aggregator.WithSmartBaseDelay(time.Hour))
	podRelay := NewPTYPodRelay("pod-1", nil, &PTYComponents{
		VirtualTerminal: vterm,
		Aggregator:      agg,
	}, nil)
	client := newSwitchableRelayClient("wss://relay.example.com", false)
	podRelay.OnRelayConnected(client)
	client.Reset()
	client.fail.Store(true)

	agg.Write([]byte("delta that desynchronizes the lane"))
	agg.Flush()
	waitForSendAttempt(t, client, runnerrelay.MsgTypeOutput, true)
	for attempt := 0; attempt < snapshotRepairLogInterval; attempt++ {
		waitForSendAttemptWithin(t, client, runnerrelay.MsgTypeSnapshot, true, 4*time.Second)
	}
	client.fail.Store(false)

	waitForSendAttemptWithin(t, client, runnerrelay.MsgTypeSnapshot, false, 5*time.Second)
	message := lastMessageOfType(t, client.MockClient, runnerrelay.MsgTypeSnapshot)
	var envelope struct {
		ResetAll bool `json:"reset_all"`
	}
	if err := json.Unmarshal(message.Payload, &envelope); err != nil {
		t.Fatalf("decode eventual recovery snapshot: %v", err)
	}
	if !envelope.ResetAll {
		t.Fatal("eventual repair did not reset ready subscribers")
	}

	podRelay.OnRelayDisconnected(client)
	agg.Stop()
}

type switchableRelayClient struct {
	*runnerrelay.MockClient
	fail     atomic.Bool
	failNext atomic.Bool
	attempts chan relaySendAttempt
}

type relaySendAttempt struct {
	msgType byte
	failed  bool
}

func newSwitchableRelayClient(url string, fail bool) *switchableRelayClient {
	client := &switchableRelayClient{
		MockClient: runnerrelay.NewMockClient(url),
		attempts:   make(chan relaySendAttempt, 32),
	}
	client.SetConnected(true)
	client.fail.Store(fail)
	return client
}

func (c *switchableRelayClient) Send(msgType byte, payload []byte) error {
	failNext := c.failNext.CompareAndSwap(true, false)
	shouldFail := c.fail.Load() || failNext
	if shouldFail {
		select {
		case c.attempts <- relaySendAttempt{msgType: msgType, failed: true}:
		default:
		}
		return errors.New("forced relay send failure")
	}
	err := c.MockClient.Send(msgType, payload)
	select {
	case c.attempts <- relaySendAttempt{msgType: msgType, failed: err != nil}:
	default:
	}
	return err
}

func (c *switchableRelayClient) Reset() {
	c.MockClient.Reset()
	for {
		select {
		case <-c.attempts:
		default:
			return
		}
	}
}

func waitForSendAttempt(
	t *testing.T,
	client *switchableRelayClient,
	msgType byte,
	wantFailure bool,
) {
	waitForSendAttemptWithin(t, client, msgType, wantFailure, time.Second)
}

func waitForSendAttemptWithin(
	t *testing.T,
	client *switchableRelayClient,
	msgType byte,
	wantFailure bool,
	timeout time.Duration,
) {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case got := <-client.attempts:
			if got.msgType == msgType {
				if got.failed != wantFailure {
					t.Fatalf("relay send type %d failure = %v, want %v", msgType, got.failed, wantFailure)
				}
				return
			}
		case <-timer.C:
			t.Fatalf("relay send type %d was not attempted", msgType)
		}
	}
}
