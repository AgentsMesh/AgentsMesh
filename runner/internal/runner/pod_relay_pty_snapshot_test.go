package runner

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	runnerrelay "github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

func TestPTYPodRelayMaterializeSnapshotUsesLatestEmptyAndShortState(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	vterm.Feed([]byte("\x1b[?1049hold alternate-screen content long enough for the old cache heuristic"))
	relay := NewPTYPodRelay("pod-1", nil, &PTYComponents{VirtualTerminal: vterm}, nil)

	first := decodePTYSnapshot(t, relay.materializeSnapshot())
	if !first.IsAltScreen || len(first.SerializedContent) <= 20 {
		t.Fatalf("initial snapshot setup failed: %+v", first)
	}

	vterm.Feed([]byte("\x1b[?1049l"))
	vterm.Clear()
	vterm.Resize(120, 40)
	vterm.Feed([]byte("\x1b[2J\x1b[2;7H"))

	empty := decodePTYSnapshot(t, relay.materializeSnapshot())
	if empty.Cols != 120 || empty.Rows != 40 {
		t.Fatalf("empty snapshot kept stale dimensions: got %dx%d, want 120x40", empty.Cols, empty.Rows)
	}
	if empty.IsAltScreen {
		t.Fatal("empty snapshot kept stale alternate-screen mode")
	}
	if strings.Contains(empty.SerializedContent, "old") {
		t.Fatalf("blank snapshot kept stale content: %q", empty.SerializedContent)
	}
	for row, line := range empty.Lines {
		if line != "" {
			t.Fatalf("blank snapshot has visible content on row %d: %q", row, line)
		}
	}
	restoredBlank := vt.NewVirtualTerminal(empty.Cols, empty.Rows, 100)
	restoredBlank.Feed([]byte(empty.SerializedContent))
	if got := restoredBlank.GetDisplay(); got != "" {
		t.Fatalf("blank snapshot rendered stale content: %q", got)
	}

	vterm.Feed([]byte("\x1b[H\x1b[2Jx"))
	short := decodePTYSnapshot(t, relay.materializeSnapshot())
	if !strings.Contains(short.SerializedContent, "x") {
		t.Fatalf("short snapshot was replaced by stale content: %q", short.SerializedContent)
	}
	if strings.Contains(short.SerializedContent, "old") {
		t.Fatalf("short snapshot leaked prior cached content: %q", short.SerializedContent)
	}
}

func TestPTYPodRelayRecoverySnapshotCarriesResetScope(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	vterm.Feed([]byte("current state"))
	podRelay := NewPTYPodRelay("pod-1", nil, &PTYComponents{VirtualTerminal: vterm}, nil)

	var normalFields map[string]json.RawMessage
	if err := json.Unmarshal(podRelay.materializeSnapshot(), &normalFields); err != nil {
		t.Fatalf("decode normal snapshot fields: %v", err)
	}
	if raw, ok := normalFields["reset_all"]; !ok || string(raw) != "false" {
		t.Fatalf("normal snapshot must explicitly carry reset_all=false, got %s", raw)
	}

	var normal, recovery struct {
		ResetAll bool `json:"reset_all"`
	}
	if err := json.Unmarshal(podRelay.materializeSnapshot(), &normal); err != nil {
		t.Fatalf("decode normal snapshot: %v", err)
	}
	if err := json.Unmarshal(podRelay.materializeSnapshotEnvelope(true), &recovery); err != nil {
		t.Fatalf("decode recovery snapshot: %v", err)
	}
	if normal.ResetAll {
		t.Fatal("requester baseline unexpectedly resets existing subscribers")
	}
	if !recovery.ResetAll {
		t.Fatal("desynchronization recovery must reset every affected subscriber")
	}
}

func TestPTYPodRelaySnapshotEnvelopeCarriesTerminalContinuationState(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	vterm.Feed([]byte("\x1b[?25l\x1b[?2004h\x1b[31"))
	podRelay := NewPTYPodRelay("pod-1", nil, &PTYComponents{VirtualTerminal: vterm}, nil)

	var envelope struct {
		ParserPrefix  []int    `json:"parser_prefix"`
		TerminalModes []string `json:"terminal_modes"`
	}
	if err := json.Unmarshal(podRelay.materializeSnapshot(), &envelope); err != nil {
		t.Fatalf("decode snapshot continuation: %v", err)
	}
	wantPrefix := []int{0x1b, '[', '3', '1'}
	if len(envelope.ParserPrefix) != len(wantPrefix) {
		t.Fatalf("parser prefix = %v, want %v", envelope.ParserPrefix, wantPrefix)
	}
	for i := range wantPrefix {
		if envelope.ParserPrefix[i] != wantPrefix[i] {
			t.Fatalf("parser prefix = %v, want %v", envelope.ParserPrefix, wantPrefix)
		}
	}
	if !containsString(envelope.TerminalModes, "\x1b[?25l") ||
		!containsString(envelope.TerminalModes, "\x1b[?2004h") {
		t.Fatalf("terminal modes missing Claude state: %q", envelope.TerminalModes)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestPTYPodRelayPublisherReconnectSendsRecoverySnapshot(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	vterm.Feed([]byte("state after publisher gap"))
	agg := aggregator.NewSmartAggregator(nil)
	podRelay := NewPTYPodRelay("pod-1", nil, &PTYComponents{
		VirtualTerminal: vterm,
		Aggregator:      agg,
	}, nil)
	client := runnerrelay.NewMockClient("wss://relay.example.com")
	client.SetConnected(true)
	podRelay.OnRelayConnected(client)

	podRelay.RecoverSnapshot(client)
	message := lastMessageOfType(t, client, runnerrelay.MsgTypeSnapshot)
	var envelope struct {
		ResetAll bool `json:"reset_all"`
	}
	if err := json.Unmarshal(message.Payload, &envelope); err != nil {
		t.Fatalf("decode recovery snapshot: %v", err)
	}
	if !envelope.ResetAll {
		t.Fatal("publisher reconnect used requester-only snapshot")
	}
	podRelay.OnRelayDisconnected(client)
	agg.Stop()
}

func TestPTYPodRelayPublisherReplacementAttachesWithRecoverySnapshot(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	vterm.Feed([]byte("authoritative replacement state"))
	agg := aggregator.NewSmartAggregator(nil)
	podRelay := NewPTYPodRelay("pod-1", nil, &PTYComponents{
		VirtualTerminal: vterm,
		Aggregator:      agg,
	}, nil)

	first := runnerrelay.NewMockClient("wss://first.example.com")
	first.SetConnected(true)
	podRelay.OnRelayConnected(first)
	firstSnapshot := lastMessageOfType(t, first, runnerrelay.MsgTypeSnapshot)
	var firstEnvelope struct {
		ResetAll bool `json:"reset_all"`
	}
	if err := json.Unmarshal(firstSnapshot.Payload, &firstEnvelope); err != nil {
		t.Fatalf("decode first publisher snapshot: %v", err)
	}
	if !firstEnvelope.ResetAll {
		t.Fatal("publisher attach must reset subscribers from any prior runner lifecycle")
	}

	replacement := runnerrelay.NewMockClient("wss://replacement.example.com")
	replacement.SetConnected(true)
	podRelay.OnRelayConnected(replacement)
	replacementSnapshot := lastMessageOfType(t, replacement, runnerrelay.MsgTypeSnapshot)
	var replacementEnvelope struct {
		ResetAll bool `json:"reset_all"`
	}
	if err := json.Unmarshal(replacementSnapshot.Payload, &replacementEnvelope); err != nil {
		t.Fatalf("decode replacement publisher snapshot: %v", err)
	}
	if !replacementEnvelope.ResetAll {
		t.Fatal("replacement publisher used requester-only baseline for ready subscribers")
	}

	podRelay.OnRelayDisconnected(replacement)
	agg.Stop()
}

func TestPTYPodRelayAggregationOverflowRecoversFromVT(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	agg := aggregator.NewSmartAggregator(nil, aggregator.WithSmartMaxSize(8))
	components := &PTYComponents{VirtualTerminal: vterm, Aggregator: agg}
	podRelay := NewPTYPodRelay("pod-1", nil, components, nil)
	client := runnerrelay.NewMockClient("wss://relay.example.com")
	client.SetConnected(true)
	podRelay.OnRelayConnected(client)
	client.Reset()

	NewPTYOutputHandler("pod-1", components, nil)([]byte("authoritative overflow state"))
	deadline := time.Now().Add(time.Second)
	for client.CountSentByType(runnerrelay.MsgTypeSnapshot) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	message := lastMessageOfType(t, client, runnerrelay.MsgTypeSnapshot)
	var envelope struct {
		ResetAll bool     `json:"reset_all"`
		Lines    []string `json:"lines"`
	}
	if err := json.Unmarshal(message.Payload, &envelope); err != nil {
		t.Fatalf("decode overflow recovery: %v", err)
	}
	if !envelope.ResetAll || len(envelope.Lines) == 0 || envelope.Lines[0] != "authoritative overflow state" {
		t.Fatalf("overflow recovery snapshot = %+v", envelope)
	}
	if got := client.CountSentByType(runnerrelay.MsgTypeOutput); got != 0 {
		t.Fatalf("overflow sent %d partial raw output messages", got)
	}
	podRelay.OnRelayDisconnected(client)
	agg.Stop()
}

func TestPTYPodRelayOverflowKeepsSplitUTF8AfterRecovery(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	agg := aggregator.NewSmartAggregator(nil,
		aggregator.WithSmartMaxSize(1),
		aggregator.WithSmartBaseDelay(time.Hour),
	)
	components := &PTYComponents{VirtualTerminal: vterm, Aggregator: agg}
	podRelay := NewPTYPodRelay("pod-1", nil, components, nil)
	client := runnerrelay.NewMockClient("wss://relay.example.com")
	client.SetConnected(true)
	podRelay.OnRelayConnected(client)
	client.Reset()
	handler := NewPTYOutputHandler("pod-1", components, nil)

	handler([]byte{0xe4, 0xb8})
	deadline := time.Now().Add(time.Second)
	for client.CountSentByType(runnerrelay.MsgTypeSnapshot) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	lastMessageOfType(t, client, runnerrelay.MsgTypeSnapshot)
	client.Reset()

	handler([]byte{0xad})
	if !agg.FlushAndWait(time.Second) {
		t.Fatal("completed UTF-8 rune did not drain after recovery")
	}
	message := lastMessageOfType(t, client, runnerrelay.MsgTypeOutput)
	if string(message.Payload) != "中" {
		t.Fatalf("post-recovery UTF-8 delta = %x, want %x", message.Payload, []byte("中"))
	}
	if got := vterm.GetSnapshot().Lines[0]; got != "中" {
		t.Fatalf("authoritative VT state = %q, want 中", got)
	}

	podRelay.OnRelayDisconnected(client)
	agg.Stop()
}

func TestPTYPodRelaySnapshotWaitsForFeedAndRawDelivery(t *testing.T) {
	vterm := vt.NewVirtualTerminal(80, 24, 100)
	agg := aggregator.NewSmartAggregator(nil)
	components := &PTYComponents{VirtualTerminal: vterm, Aggregator: agg}
	podRelay := NewPTYPodRelay("pod-1", nil, components, nil)
	client := runnerrelay.NewMockClient("wss://relay.example.com")
	client.SetConnected(true)
	podRelay.OnRelayConnected(client)
	client.Reset()

	feedComplete := make(chan struct{})
	releaseOutput := make(chan struct{})
	outputDone := make(chan struct{})
	handler := NewPTYOutputHandler("pod-1", components, func(_ int, _ []string) {
		close(feedComplete)
		<-releaseOutput
	})
	go func() {
		defer close(outputDone)
		handler([]byte("ordered chunk"))
	}()
	<-feedComplete

	snapshotDone := make(chan struct{})
	go func() {
		defer close(snapshotDone)
		podRelay.SendSnapshot(client)
	}()

	time.Sleep(20 * time.Millisecond)
	if got := client.CountSentByType(runnerrelay.MsgTypeSnapshot); got != 0 {
		t.Fatalf("snapshot crossed Feed/Write transaction: got %d early snapshot", got)
	}
	close(releaseOutput)
	<-outputDone
	<-snapshotDone

	outputIndex, snapshotIndex := -1, -1
	for index, message := range client.SentMessages {
		switch message.Type {
		case runnerrelay.MsgTypeOutput:
			if outputIndex < 0 {
				outputIndex = index
			}
		case runnerrelay.MsgTypeSnapshot:
			if snapshotIndex < 0 {
				snapshotIndex = index
			}
		}
	}
	if outputIndex < 0 || snapshotIndex < 0 || outputIndex >= snapshotIndex {
		t.Fatalf("expected raw output before snapshot, messages=%+v", client.SentMessages)
	}

	podRelay.OnRelayDisconnected(client)
	agg.Stop()
}

func decodePTYSnapshot(t *testing.T, payload []byte) *vt.TerminalSnapshot {
	t.Helper()
	if payload == nil {
		t.Fatal("snapshot payload is nil")
	}
	var snapshot vt.TerminalSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	return &snapshot
}

func lastMessageOfType(t *testing.T, client *runnerrelay.MockClient, msgType byte) runnerrelay.MockSentMessage {
	t.Helper()
	for index := len(client.SentMessages) - 1; index >= 0; index-- {
		if client.SentMessages[index].Type == msgType {
			return client.SentMessages[index]
		}
	}
	t.Fatalf("message type %d not sent", msgType)
	return runnerrelay.MockSentMessage{}
}
