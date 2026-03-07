package channel

import (
	"bytes"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/agentsmesh/relay/internal/protocol"
)

// ==================== Publisher → Subscriber Forwarding ====================

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
	_ = subClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := subClient.ReadMessage()
	if err != nil {
		t.Fatalf("read from subClient: %v", err)
	}
	if !bytes.Equal(data, outMsg) {
		t.Fatalf("forwarded data mismatch: got %v, want %v", data, outMsg)
	}
}

// ==================== Subscriber → Publisher Forwarding ====================

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

	_ = pubClient.SetReadDeadline(time.Now().Add(2 * time.Second))
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

	_ = subClient.SetReadDeadline(time.Now().Add(2 * time.Second))
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
	_ = pubClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := pubClient.ReadMessage()
	if err != nil {
		t.Fatalf("read image paste from pubClient: %v", err)
	}
	if !bytes.Equal(data, imgData) {
		t.Fatalf("image paste data mismatch: got %v, want %v", data, imgData)
	}
}

// ==================== Input Rejection (Control Enforcement) ====================

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
	_ = pubClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := pubClient.ReadMessage()
	if err != nil {
		t.Fatalf("read from pubClient: %v", err)
	}
	if !bytes.Equal(data, s1InputMsg) {
		t.Fatalf("expected s1 input, got %v", data)
	}

	// Verify publisher doesn't receive s2's rejected message within a short window
	_ = pubClient.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
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
	_ = pubClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := pubClient.ReadMessage()
	if err != nil {
		t.Fatalf("read from pubClient: %v", err)
	}
	if !bytes.Equal(data, s1ImgMsg) {
		t.Fatalf("expected s1 image paste, got different data")
	}
}

// ==================== Control Request Forwarding ====================

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

	_ = s1Client.SetReadDeadline(time.Now().Add(2 * time.Second))
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

	_ = s2Client.SetReadDeadline(time.Now().Add(2 * time.Second))
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

	_ = s1Client.SetReadDeadline(time.Now().Add(2 * time.Second))
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

	_ = s1Client.SetReadDeadline(time.Now().Add(2 * time.Second))
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
	_ = pubClient.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := pubClient.ReadMessage()
	if err != nil {
		t.Fatalf("read from pubClient: %v", err)
	}
	if !bytes.Equal(data, inputMsg) {
		t.Fatalf("expected input message, got %v", data)
	}
}
