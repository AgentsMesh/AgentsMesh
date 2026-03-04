package channel

import (
	"bytes"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/agentsmesh/relay/internal/protocol"
)

func TestNewTerminalChannel(t *testing.T) {
	ch := NewTerminalChannel("pod-1", 200*time.Millisecond, nil, nil)
	if ch.PodKey != "pod-1" {
		t.Fatalf("PodKey: got %q, want %q", ch.PodKey, "pod-1")
	}
	if ch.IsClosed() {
		t.Fatal("expected IsClosed false")
	}
	if ch.SubscriberCount() != 0 {
		t.Fatalf("SubscriberCount: got %d, want 0", ch.SubscriberCount())
	}
	if ch.GetPublisher() != nil {
		t.Fatal("expected GetPublisher nil")
	}
}

func TestNewTerminalChannelWithConfig(t *testing.T) {
	cfg := testChannelConfig()
	cfg.OutputBufferCount = 42
	ch := NewTerminalChannelWithConfig("pod-cfg", cfg, nil, nil)
	if ch.PodKey != "pod-cfg" {
		t.Fatalf("PodKey: got %q, want %q", ch.PodKey, "pod-cfg")
	}
	if ch.config.OutputBufferCount != 42 {
		t.Fatalf("OutputBufferCount: got %d, want 42", ch.config.OutputBufferCount)
	}
}

func TestTerminalChannel_SetPublisher(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-pub", testChannelConfig(), nil, nil)
	serverConn, _ := createWSPair(t)

	ch.SetPublisher(serverConn)

	if ch.GetPublisher() == nil {
		t.Fatal("expected GetPublisher non-nil after SetPublisher")
	}
	if ch.IsPublisherDisconnected() {
		t.Fatal("expected IsPublisherDisconnected false")
	}
}

func TestTerminalChannel_SetPublisher_Reconnect(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-recon", testChannelConfig(), nil, nil)

	// Create publisher WS pair and subscriber WS pair
	pubServer, pubClient := createWSPair(t)
	subServer, subClient := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	// Close the publisher client side to trigger disconnect
	pubClient.Close()

	// Wait for publisher disconnect to be detected
	waitFor(t, func() bool {
		return ch.IsPublisherDisconnected()
	}, 2*time.Second)

	// Read RunnerDisconnected from subscriber client
	_, data, err := subClient.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read RunnerDisconnected: %v", err)
	}
	msg, err := protocol.DecodeMessage(data)
	if err != nil {
		t.Fatalf("decode RunnerDisconnected: %v", err)
	}
	if msg.Type != protocol.MsgTypeRunnerDisconnected {
		t.Fatalf("expected MsgTypeRunnerDisconnected (0x%02x), got 0x%02x", protocol.MsgTypeRunnerDisconnected, msg.Type)
	}

	// Reconnect with new publisher
	newPubServer, _ := createWSPair(t)
	ch.SetPublisher(newPubServer)

	// Read RunnerReconnected from subscriber client
	_, data, err = subClient.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read RunnerReconnected: %v", err)
	}
	msg, err = protocol.DecodeMessage(data)
	if err != nil {
		t.Fatalf("decode RunnerReconnected: %v", err)
	}
	if msg.Type != protocol.MsgTypeRunnerReconnected {
		t.Fatalf("expected MsgTypeRunnerReconnected (0x%02x), got 0x%02x", protocol.MsgTypeRunnerReconnected, msg.Type)
	}

	if ch.IsPublisherDisconnected() {
		t.Fatal("expected IsPublisherDisconnected false after reconnect")
	}
}

func TestTerminalChannel_AddSubscriber(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-sub", testChannelConfig(), nil, nil)
	serverConn, _ := createWSPair(t)

	ch.AddSubscriber("s1", serverConn)

	if ch.SubscriberCount() != 1 {
		t.Fatalf("SubscriberCount: got %d, want 1", ch.SubscriberCount())
	}
}

func TestTerminalChannel_AddSubscriber_ReceivesBuffer(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-buf-recv", testChannelConfig(), nil, nil)

	// Buffer some output messages
	msg1 := protocol.EncodeOutput([]byte("hello"))
	msg2 := protocol.EncodeOutput([]byte("world"))
	ch.bufferOutput(msg1)
	ch.bufferOutput(msg2)

	// Add subscriber
	serverConn, clientConn := createWSPair(t)
	ch.AddSubscriber("s1", serverConn)

	// Read buffered messages from subscriber client
	var received [][]byte
	for i := 0; i < 2; i++ {
		clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, data, err := clientConn.ReadMessage()
		if err != nil {
			t.Fatalf("read buffered message %d: %v", i, err)
		}
		received = append(received, data)
	}

	if !bytes.Equal(received[0], msg1) {
		t.Fatalf("first buffered message mismatch: got %v, want %v", received[0], msg1)
	}
	if !bytes.Equal(received[1], msg2) {
		t.Fatalf("second buffered message mismatch: got %v, want %v", received[1], msg2)
	}
}

func TestTerminalChannel_RemoveSubscriber(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-rm", testChannelConfig(), nil, nil)
	serverConn, _ := createWSPair(t)

	ch.AddSubscriber("s1", serverConn)
	ch.RequestControl("s1")

	ch.RemoveSubscriber("s1")

	if ch.SubscriberCount() != 0 {
		t.Fatalf("SubscriberCount: got %d, want 0", ch.SubscriberCount())
	}
	// After removal, control should be released (any ID can input)
	if !ch.CanInput("other") {
		t.Fatal("expected CanInput true after controller removed")
	}
}

func TestTerminalChannel_Broadcast(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-bc", testChannelConfig(), nil, nil)

	s1Server, s1Client := createWSPair(t)
	s2Server, s2Client := createWSPair(t)

	ch.AddSubscriber("s1", s1Server)
	ch.AddSubscriber("s2", s2Server)

	data := []byte("broadcast-data")
	ch.Broadcast(data)

	// Read from both
	for _, tc := range []struct {
		name string
		conn *websocket.Conn
	}{
		{"s1", s1Client},
		{"s2", s2Client},
	} {
		tc.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, got, err := tc.conn.ReadMessage()
		if err != nil {
			t.Fatalf("read %s: %v", tc.name, err)
		}
		if !bytes.Equal(got, data) {
			t.Fatalf("%s: got %v, want %v", tc.name, got, data)
		}
	}
}

func TestTerminalChannel_BufferOutput(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-bo", testChannelConfig(), nil, nil)

	ch.bufferOutput([]byte("a"))
	ch.bufferOutput([]byte("b"))
	ch.bufferOutput([]byte("c"))

	buf := ch.getBufferedOutput()
	if len(buf) != 3 {
		t.Fatalf("buffer len: got %d, want 3", len(buf))
	}
	if !bytes.Equal(buf[0], []byte("a")) || !bytes.Equal(buf[1], []byte("b")) || !bytes.Equal(buf[2], []byte("c")) {
		t.Fatal("buffer content mismatch")
	}
}

func TestTerminalChannel_BufferOutput_CountLimit(t *testing.T) {
	cfg := testChannelConfig()
	cfg.OutputBufferCount = 5
	cfg.OutputBufferSize = 100000 // large size so count is the limiting factor
	ch := NewTerminalChannelWithConfig("pod-bo-count", cfg, nil, nil)

	for i := 0; i < 7; i++ {
		ch.bufferOutput([]byte{byte(i)})
	}

	buf := ch.getBufferedOutput()
	if len(buf) != 5 {
		t.Fatalf("buffer len: got %d, want 5", len(buf))
	}
	// Oldest (0,1) should be evicted, remaining: 2,3,4,5,6
	if buf[0][0] != 2 {
		t.Fatalf("oldest message: got %d, want 2", buf[0][0])
	}
	if buf[4][0] != 6 {
		t.Fatalf("newest message: got %d, want 6", buf[4][0])
	}
}

func TestTerminalChannel_BufferOutput_SizeLimit(t *testing.T) {
	cfg := testChannelConfig()
	cfg.OutputBufferSize = 10  // very small size limit
	cfg.OutputBufferCount = 100 // large count so size is the limiting factor
	ch := NewTerminalChannelWithConfig("pod-bo-size", cfg, nil, nil)

	// Each message is 5 bytes. Size limit is 10, so max 2 messages.
	ch.bufferOutput([]byte("aaaaa")) // 5 bytes, total=5
	ch.bufferOutput([]byte("bbbbb")) // 5 bytes, total=10
	ch.bufferOutput([]byte("ccccc")) // need to evict "aaaaa" to fit, total=10

	buf := ch.getBufferedOutput()
	if len(buf) != 2 {
		t.Fatalf("buffer len: got %d, want 2", len(buf))
	}
	if !bytes.Equal(buf[0], []byte("bbbbb")) {
		t.Fatalf("first message: got %q, want %q", buf[0], "bbbbb")
	}
	if !bytes.Equal(buf[1], []byte("ccccc")) {
		t.Fatalf("second message: got %q, want %q", buf[1], "ccccc")
	}
}

func TestTerminalChannel_BufferOutput_OversizedSingle(t *testing.T) {
	cfg := testChannelConfig()
	cfg.OutputBufferSize = 10
	ch := NewTerminalChannelWithConfig("pod-bo-over", cfg, nil, nil)

	// Single message larger than OutputBufferSize should be skipped
	bigMsg := make([]byte, 11)
	ch.bufferOutput(bigMsg)

	buf := ch.getBufferedOutput()
	if len(buf) != 0 {
		t.Fatalf("buffer len: got %d, want 0 (oversized message should be skipped)", len(buf))
	}
}

func TestTerminalChannel_CanInput(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-ci", testChannelConfig(), nil, nil)

	// No controller: anyone can input
	if !ch.CanInput("s1") {
		t.Fatal("expected CanInput true for s1 (no controller)")
	}
	if !ch.CanInput("s2") {
		t.Fatal("expected CanInput true for s2 (no controller)")
	}

	// Grant control to s1
	ch.RequestControl("s1")

	if !ch.CanInput("s1") {
		t.Fatal("expected CanInput true for s1 (controller)")
	}
	if ch.CanInput("s2") {
		t.Fatal("expected CanInput false for s2 (not controller)")
	}
}

func TestTerminalChannel_RequestControl(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-rc", testChannelConfig(), nil, nil)

	if !ch.RequestControl("s1") {
		t.Fatal("expected RequestControl to succeed for s1")
	}
	if ch.RequestControl("s2") {
		t.Fatal("expected RequestControl to fail for s2 (s1 already has control)")
	}
}

func TestTerminalChannel_ReleaseControl(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-rlc", testChannelConfig(), nil, nil)

	ch.RequestControl("s1")

	// Release by non-controller does nothing
	ch.ReleaseControl("s2")
	if ch.CanInput("s2") {
		t.Fatal("expected CanInput false for s2 after non-controller release")
	}

	// Release by controller succeeds
	ch.ReleaseControl("s1")
	if !ch.CanInput("s2") {
		t.Fatal("expected CanInput true for s2 after controller release")
	}
}

func TestTerminalChannel_ForwardPubToSub(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-fwd-ps", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	subServer, subClient := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	// Write Output message from publisher client (simulating runner sending data)
	payload := []byte("terminal output data")
	outMsg := protocol.EncodeOutput(payload)
	if err := pubClient.WriteMessage(websocket.BinaryMessage, outMsg); err != nil {
		t.Fatalf("write to pubClient: %v", err)
	}

	// Read from subscriber client
	subClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := subClient.ReadMessage()
	if err != nil {
		t.Fatalf("read from subClient: %v", err)
	}
	if !bytes.Equal(data, outMsg) {
		t.Fatalf("forwarded data mismatch: got %v, want %v", data, outMsg)
	}
}

func TestTerminalChannel_ForwardSubToPub(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-fwd-sp", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	subServer, subClient := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	// Test 1: Input message forwarded to publisher
	inputPayload := []byte("user input")
	inputMsg := protocol.EncodeInput(inputPayload)
	if err := subClient.WriteMessage(websocket.BinaryMessage, inputMsg); err != nil {
		t.Fatalf("write input to subClient: %v", err)
	}

	pubClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := pubClient.ReadMessage()
	if err != nil {
		t.Fatalf("read input from pubClient: %v", err)
	}
	if !bytes.Equal(data, inputMsg) {
		t.Fatalf("forwarded input mismatch: got %v, want %v", data, inputMsg)
	}

	// Test 2: Ping results in Pong back to subscriber (not forwarded to publisher)
	pingMsg := protocol.EncodePing()
	if err := subClient.WriteMessage(websocket.BinaryMessage, pingMsg); err != nil {
		t.Fatalf("write ping to subClient: %v", err)
	}

	subClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err = subClient.ReadMessage()
	if err != nil {
		t.Fatalf("read pong from subClient: %v", err)
	}
	msg, err := protocol.DecodeMessage(data)
	if err != nil {
		t.Fatalf("decode pong: %v", err)
	}
	if msg.Type != protocol.MsgTypePong {
		t.Fatalf("expected MsgTypePong (0x%02x), got 0x%02x", protocol.MsgTypePong, msg.Type)
	}
}

func TestTerminalChannel_PublisherDisconnect_Timeout(t *testing.T) {
	cfg := testChannelConfig()
	cfg.PublisherReconnectTimeout = 200 * time.Millisecond
	ch := NewTerminalChannelWithConfig("pod-pdt", cfg, nil, nil)

	pubServer, pubClient := createWSPair(t)
	subServer, _ := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	// Close publisher client to trigger disconnect
	pubClient.Close()

	// Wait for channel to close due to reconnect timeout
	waitFor(t, func() bool {
		return ch.IsClosed()
	}, 2*time.Second)

	if !ch.IsClosed() {
		t.Fatal("expected channel to be closed after publisher reconnect timeout")
	}
}

func TestTerminalChannel_Close(t *testing.T) {
	closedCount := 0
	var closedKey string
	onClosed := func(podKey string) {
		closedCount++
		closedKey = podKey
	}

	ch := NewTerminalChannelWithConfig("pod-close", testChannelConfig(), nil, onClosed)

	// First close
	ch.Close()
	if !ch.IsClosed() {
		t.Fatal("expected IsClosed true after Close")
	}
	if closedCount != 1 {
		t.Fatalf("onChannelClosed called %d times, want 1", closedCount)
	}
	if closedKey != "pod-close" {
		t.Fatalf("onChannelClosed podKey: got %q, want %q", closedKey, "pod-close")
	}

	// Second close (idempotent)
	ch.Close()
	if closedCount != 1 {
		t.Fatalf("onChannelClosed called %d times after second Close, want 1", closedCount)
	}
}

func TestTerminalChannel_ControlRequest_ViaForwarding(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-ctrl-fwd", testChannelConfig(), nil, nil)

	pubServer, _ := createWSPair(t)
	s1Server, s1Client := createWSPair(t)
	s2Server, s2Client := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", s1Server)
	ch.AddSubscriber("s2", s2Server)

	// --- s1 sends Control "request" → should get "granted" ---
	reqMsg := &protocol.ControlRequest{Action: "request", BrowserID: "s1"}
	reqData, err := protocol.EncodeControlRequest(reqMsg)
	if err != nil {
		t.Fatalf("encode control request: %v", err)
	}
	if err := s1Client.WriteMessage(websocket.BinaryMessage, reqData); err != nil {
		t.Fatalf("s1 write control request: %v", err)
	}

	s1Client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, respData, err := s1Client.ReadMessage()
	if err != nil {
		t.Fatalf("s1 read control response: %v", err)
	}
	msg, err := protocol.DecodeMessage(respData)
	if err != nil {
		t.Fatalf("decode control response: %v", err)
	}
	if msg.Type != protocol.MsgTypeControl {
		t.Fatalf("expected MsgTypeControl (0x%02x), got 0x%02x", protocol.MsgTypeControl, msg.Type)
	}
	resp, err := protocol.DecodeControlRequest(msg.Payload)
	if err != nil {
		t.Fatalf("decode control request body: %v", err)
	}
	if resp.Action != "granted" {
		t.Fatalf("expected action 'granted', got %q", resp.Action)
	}
	if resp.Controller != "s1" {
		t.Fatalf("expected controller 's1', got %q", resp.Controller)
	}

	// --- s2 sends Control "request" → should get "denied" ---
	reqMsg2 := &protocol.ControlRequest{Action: "request", BrowserID: "s2"}
	reqData2, err := protocol.EncodeControlRequest(reqMsg2)
	if err != nil {
		t.Fatalf("encode control request s2: %v", err)
	}
	if err := s2Client.WriteMessage(websocket.BinaryMessage, reqData2); err != nil {
		t.Fatalf("s2 write control request: %v", err)
	}

	s2Client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, respData2, err := s2Client.ReadMessage()
	if err != nil {
		t.Fatalf("s2 read control response: %v", err)
	}
	msg2, err := protocol.DecodeMessage(respData2)
	if err != nil {
		t.Fatalf("decode control response s2: %v", err)
	}
	if msg2.Type != protocol.MsgTypeControl {
		t.Fatalf("expected MsgTypeControl (0x%02x), got 0x%02x", protocol.MsgTypeControl, msg2.Type)
	}
	resp2, err := protocol.DecodeControlRequest(msg2.Payload)
	if err != nil {
		t.Fatalf("decode control request body s2: %v", err)
	}
	if resp2.Action != "denied" {
		t.Fatalf("expected action 'denied', got %q", resp2.Action)
	}
	if resp2.Controller != "s1" {
		t.Fatalf("expected controller 's1' in denied response, got %q", resp2.Controller)
	}

	// --- s1 sends Control "query" → should get "status" with controller=s1 ---
	queryMsg := &protocol.ControlRequest{Action: "query", BrowserID: "s1"}
	queryData, err := protocol.EncodeControlRequest(queryMsg)
	if err != nil {
		t.Fatalf("encode control query: %v", err)
	}
	if err := s1Client.WriteMessage(websocket.BinaryMessage, queryData); err != nil {
		t.Fatalf("s1 write control query: %v", err)
	}

	s1Client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, queryRespData, err := s1Client.ReadMessage()
	if err != nil {
		t.Fatalf("s1 read control query response: %v", err)
	}
	queryRespMsg, err := protocol.DecodeMessage(queryRespData)
	if err != nil {
		t.Fatalf("decode control query response: %v", err)
	}
	if queryRespMsg.Type != protocol.MsgTypeControl {
		t.Fatalf("expected MsgTypeControl, got 0x%02x", queryRespMsg.Type)
	}
	queryResp, err := protocol.DecodeControlRequest(queryRespMsg.Payload)
	if err != nil {
		t.Fatalf("decode control query body: %v", err)
	}
	if queryResp.Action != "status" {
		t.Fatalf("expected action 'status', got %q", queryResp.Action)
	}
	if queryResp.Controller != "s1" {
		t.Fatalf("expected controller 's1' in status, got %q", queryResp.Controller)
	}

	// --- s1 sends Control "release" → should get "released" ---
	releaseMsg := &protocol.ControlRequest{Action: "release", BrowserID: "s1"}
	releaseData, err := protocol.EncodeControlRequest(releaseMsg)
	if err != nil {
		t.Fatalf("encode control release: %v", err)
	}
	if err := s1Client.WriteMessage(websocket.BinaryMessage, releaseData); err != nil {
		t.Fatalf("s1 write control release: %v", err)
	}

	s1Client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, releaseRespData, err := s1Client.ReadMessage()
	if err != nil {
		t.Fatalf("s1 read control release response: %v", err)
	}
	releaseRespMsg, err := protocol.DecodeMessage(releaseRespData)
	if err != nil {
		t.Fatalf("decode control release response: %v", err)
	}
	if releaseRespMsg.Type != protocol.MsgTypeControl {
		t.Fatalf("expected MsgTypeControl, got 0x%02x", releaseRespMsg.Type)
	}
	releaseResp, err := protocol.DecodeControlRequest(releaseRespMsg.Payload)
	if err != nil {
		t.Fatalf("decode control release body: %v", err)
	}
	if releaseResp.Action != "released" {
		t.Fatalf("expected action 'released', got %q", releaseResp.Action)
	}
	if releaseResp.Controller != "" {
		t.Fatalf("expected empty controller after release, got %q", releaseResp.Controller)
	}
}

func TestTerminalChannel_ForwardSubToPub_ImagePaste(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-img-paste", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	subServer, subClient := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	// Subscriber sends ImagePaste message (no controller so CanInput is true)
	imgData, err := protocol.EncodeImagePaste("image/png", []byte("fake-png-data"))
	if err != nil {
		t.Fatalf("encode image paste: %v", err)
	}
	if err := subClient.WriteMessage(websocket.BinaryMessage, imgData); err != nil {
		t.Fatalf("write image paste: %v", err)
	}

	// Read from publisher — should receive the image paste data
	pubClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := pubClient.ReadMessage()
	if err != nil {
		t.Fatalf("read image paste from pubClient: %v", err)
	}
	if !bytes.Equal(data, imgData) {
		t.Fatalf("image paste data mismatch: got %v, want %v", data, imgData)
	}
}

func TestTerminalChannel_ForwardSubToPub_InputRejected(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-input-rej", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	s1Server, s1Client := createWSPair(t)
	s2Server, s2Client := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", s1Server)
	ch.AddSubscriber("s2", s2Server)

	// Grant control to s1
	if !ch.RequestControl("s1") {
		t.Fatal("expected RequestControl to succeed for s1")
	}

	// s2 sends input — should be silently rejected (not forwarded)
	s2InputMsg := protocol.EncodeInput([]byte("rejected"))
	if err := s2Client.WriteMessage(websocket.BinaryMessage, s2InputMsg); err != nil {
		t.Fatalf("s2 write input: %v", err)
	}

	// s1 sends input — should be forwarded
	s1InputMsg := protocol.EncodeInput([]byte("accepted"))
	if err := s1Client.WriteMessage(websocket.BinaryMessage, s1InputMsg); err != nil {
		t.Fatalf("s1 write input: %v", err)
	}

	// Read from publisher — should only get s1's message
	pubClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := pubClient.ReadMessage()
	if err != nil {
		t.Fatalf("read from pubClient: %v", err)
	}
	if !bytes.Equal(data, s1InputMsg) {
		t.Fatalf("expected s1 input, got %v", data)
	}

	// Verify publisher doesn't receive s2's rejected message within a short window
	pubClient.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = pubClient.ReadMessage()
	if err == nil {
		t.Fatal("expected no more messages from publisher (s2 input should have been rejected)")
	}
}

func TestTerminalChannel_ForwardSubToPub_ImagePasteRejected(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-img-rej", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	s1Server, s1Client := createWSPair(t)
	s2Server, s2Client := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", s1Server)
	ch.AddSubscriber("s2", s2Server)

	// Grant control to s1
	if !ch.RequestControl("s1") {
		t.Fatal("expected RequestControl to succeed for s1")
	}

	// s2 sends image paste — should be silently rejected
	s2ImgMsg, err := protocol.EncodeImagePaste("image/png", []byte("rejected-img"))
	if err != nil {
		t.Fatalf("encode image paste: %v", err)
	}
	if err := s2Client.WriteMessage(websocket.BinaryMessage, s2ImgMsg); err != nil {
		t.Fatalf("s2 write image paste: %v", err)
	}

	// s1 sends image paste — should be forwarded
	s1ImgMsg, err := protocol.EncodeImagePaste("image/png", []byte("accepted-img"))
	if err != nil {
		t.Fatalf("encode image paste: %v", err)
	}
	if err := s1Client.WriteMessage(websocket.BinaryMessage, s1ImgMsg); err != nil {
		t.Fatalf("s1 write image paste: %v", err)
	}

	// Read from publisher — should only get s1's image paste
	pubClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := pubClient.ReadMessage()
	if err != nil {
		t.Fatalf("read from pubClient: %v", err)
	}
	if !bytes.Equal(data, s1ImgMsg) {
		t.Fatalf("expected s1 image paste, got different data")
	}
}

func TestTerminalChannel_AddSubscriber_CancelsKeepAliveTimer(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-ka-cancel", testChannelConfig(), nil, nil)

	s1Server, _ := createWSPair(t)
	ch.AddSubscriber("s1", s1Server)

	// Remove subscriber to start keep-alive timer
	ch.RemoveSubscriber("s1")

	if ch.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers after removal, got %d", ch.SubscriberCount())
	}

	// Add new subscriber — should cancel the keep-alive timer
	s2Server, _ := createWSPair(t)
	ch.AddSubscriber("s2", s2Server)

	if ch.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber after re-add, got %d", ch.SubscriberCount())
	}

	// Wait longer than KeepAliveDuration to verify timer was cancelled
	// If timer wasn't cancelled, the channel would trigger onAllSubscribersGone
	time.Sleep(300 * time.Millisecond)

	// Channel should still have the subscriber
	if ch.SubscriberCount() != 1 {
		t.Fatalf("expected 1 subscriber after waiting, got %d", ch.SubscriberCount())
	}
}

func TestTerminalChannel_ControlRequest_InvalidPayload(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-ctrl-bad", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	subServer, subClient := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	// Send a Control message with invalid JSON payload
	// Build a raw control message: [0x07][invalid-json]
	invalidControlMsg := protocol.EncodeMessage(protocol.MsgTypeControl, []byte("not-valid-json"))
	if err := subClient.WriteMessage(websocket.BinaryMessage, invalidControlMsg); err != nil {
		t.Fatalf("write invalid control: %v", err)
	}

	// Send a valid input after to verify the subscriber goroutine is still alive
	inputMsg := protocol.EncodeInput([]byte("still alive"))
	if err := subClient.WriteMessage(websocket.BinaryMessage, inputMsg); err != nil {
		t.Fatalf("write input after invalid control: %v", err)
	}

	// Verify publisher gets the input (i.e. the invalid control was silently skipped)
	pubClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := pubClient.ReadMessage()
	if err != nil {
		t.Fatalf("read from pubClient: %v", err)
	}
	if !bytes.Equal(data, inputMsg) {
		t.Fatalf("expected input message, got %v", data)
	}
}

func TestTerminalChannel_AddSubscriber_PubDisconnected(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-sub-disc", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	ch.SetPublisher(pubServer)

	// Close the publisher client side to trigger disconnect
	pubClient.Close()

	// Wait for publisher disconnect to be detected
	waitFor(t, func() bool {
		return ch.IsPublisherDisconnected()
	}, 2*time.Second)

	// Add a new subscriber AFTER publisher is disconnected
	subServer, subClient := createWSPair(t)
	ch.AddSubscriber("s1", subServer)

	// Read RunnerDisconnected notification from subClient
	subClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := subClient.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read RunnerDisconnected: %v", err)
	}
	msg, err := protocol.DecodeMessage(data)
	if err != nil {
		t.Fatalf("decode RunnerDisconnected: %v", err)
	}
	if msg.Type != protocol.MsgTypeRunnerDisconnected {
		t.Fatalf("expected MsgTypeRunnerDisconnected (0x%02x), got 0x%02x", protocol.MsgTypeRunnerDisconnected, msg.Type)
	}
}

func TestTerminalChannel_RemoveSubscriber_OnAllSubscribersGoneCallback(t *testing.T) {
	callbackCalled := make(chan string, 1)
	onGone := func(podKey string) {
		callbackCalled <- podKey
	}

	cfg := testChannelConfig()
	cfg.KeepAliveDuration = 50 * time.Millisecond
	ch := NewTerminalChannelWithConfig("pod-gone-cb", cfg, onGone, nil)

	subServer, _ := createWSPair(t)
	ch.AddSubscriber("s1", subServer)

	// Remove subscriber — triggers keep-alive timer
	ch.RemoveSubscriber("s1")

	// Wait for the callback to fire
	select {
	case key := <-callbackCalled:
		if key != "pod-gone-cb" {
			t.Fatalf("callback podKey: got %q, want %q", key, "pod-gone-cb")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("onAllSubscribersGone callback was not called within timeout")
	}
}

func TestTerminalChannel_ForwardSubToPub_InvalidMessage(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-inv-msg", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	subServer, subClient := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	// Send an empty binary message (will fail DecodeMessage → continue)
	if err := subClient.WriteMessage(websocket.BinaryMessage, []byte{}); err != nil {
		t.Fatalf("write empty message: %v", err)
	}

	// Follow up with a valid input to verify the goroutine is still alive
	inputMsg := protocol.EncodeInput([]byte("after-invalid"))
	if err := subClient.WriteMessage(websocket.BinaryMessage, inputMsg); err != nil {
		t.Fatalf("write input: %v", err)
	}

	// Verify publisher gets the valid input (invalid message was silently skipped)
	pubClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := pubClient.ReadMessage()
	if err != nil {
		t.Fatalf("read from pubClient: %v", err)
	}
	if !bytes.Equal(data, inputMsg) {
		t.Fatalf("expected input message, got %v", data)
	}
}

func TestTerminalChannel_AddSubscriber_BufferedOutputWriteError(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-buf-err", testChannelConfig(), nil, nil)

	// Buffer some output
	ch.bufferOutput(protocol.EncodeOutput([]byte("buffered-data")))

	// Create a subscriber WS pair and close the SERVER-side conn immediately
	// so that WriteMessage during AddSubscriber will fail
	subServer, _ := createWSPair(t)
	subServer.Close() // Close server-side, writes will definitely fail

	// AddSubscriber should handle the write error gracefully (not panic)
	ch.AddSubscriber("s1", subServer)

	// The subscriber goroutine will also exit quickly since the conn is closed
	time.Sleep(100 * time.Millisecond)
}

func TestTerminalChannel_ForwardSubToPub_PublisherWriteError(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-pub-werr", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	subServer, subClient := createWSPair(t)

	ch.SetPublisher(pubServer)
	ch.AddSubscriber("s1", subServer)

	// Close pubClient to break the connection (make writes to pubServer fail)
	pubClient.Close()

	// Wait for publisher disconnect to be detected so the forwardPublisherToSubscribers loop breaks
	waitFor(t, func() bool {
		return ch.IsPublisherDisconnected()
	}, 2*time.Second)

	// Set a new publisher that is already closed (so WriteMessage will fail)
	brokenPubServer, brokenPubClient := createWSPair(t)
	brokenPubServer.Close() // Close server-side so writes fail immediately
	brokenPubClient.Close()

	// Directly set publisher to the broken conn (bypassing SetPublisher's goroutine start)
	ch.publisherMu.Lock()
	ch.publisher = brokenPubServer
	ch.publisherDisconnected = false
	ch.publisherMu.Unlock()

	// Now subscriber sends input — publisher WriteMessage should fail (covered error branch)
	inputMsg := protocol.EncodeInput([]byte("will-fail-to-forward"))
	if err := subClient.WriteMessage(websocket.BinaryMessage, inputMsg); err != nil {
		t.Fatalf("write input: %v", err)
	}

	// Give the goroutine time to process the message and hit the error
	time.Sleep(200 * time.Millisecond)
}

func TestTerminalChannel_AddSubscriber_PubDisconnectedWriteError(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-disc-err", testChannelConfig(), nil, nil)

	pubServer, pubClient := createWSPair(t)
	ch.SetPublisher(pubServer)

	// Close the publisher to trigger disconnect
	pubClient.Close()
	waitFor(t, func() bool {
		return ch.IsPublisherDisconnected()
	}, 2*time.Second)

	// Create a subscriber WS pair and close the SERVER-side conn
	// so that the WriteMessage for RunnerDisconnected will fail
	subServer, _ := createWSPair(t)
	subServer.Close()

	// AddSubscriber should handle the write error gracefully (not panic)
	ch.AddSubscriber("s1", subServer)

	// The subscriber goroutine will also exit quickly since the conn is closed
	time.Sleep(100 * time.Millisecond)
}

func TestTerminalChannel_ForwardPubToSub_NilPublisher(t *testing.T) {
	// Test the conn == nil early exit path in forwardPublisherToSubscribers
	ch := NewTerminalChannelWithConfig("pod-nil-pub", testChannelConfig(), nil, nil)

	// Directly call forwardPublisherToSubscribers without setting a publisher
	// The publisher is nil, so the goroutine should exit immediately via the conn == nil check
	done := make(chan struct{})
	go func() {
		ch.forwardPublisherToSubscribers()
		close(done)
	}()

	select {
	case <-done:
		// Success - the goroutine exited because publisher was nil
	case <-time.After(2 * time.Second):
		t.Fatal("forwardPublisherToSubscribers did not exit when publisher is nil")
	}
}
