package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// --- Mock implementations for loop testing ---

// mockWebSocketConnWithControl provides fine-grained control over read behavior
type mockWebSocketConnWithControl struct {
	readMessages  chan []byte
	writtenData   [][]byte
	closed        atomic.Bool
	readError     error
	writeError    error
	mu            sync.Mutex
	readDeadline  time.Time
	closeCallback func()
}

func newMockWebSocketConnWithControl() *mockWebSocketConnWithControl {
	return &mockWebSocketConnWithControl{
		readMessages: make(chan []byte, 10),
		writtenData:  make([][]byte, 0),
	}
}

func (m *mockWebSocketConnWithControl) ReadMessage() (int, []byte, error) {
	if m.readError != nil {
		return 0, nil, m.readError
	}

	// Check if closed
	if m.closed.Load() {
		return 0, nil, websocket.ErrCloseSent
	}

	select {
	case msg := <-m.readMessages:
		return websocket.TextMessage, msg, nil
	case <-time.After(100 * time.Millisecond):
		// Return error to exit read loop
		return 0, nil, websocket.ErrCloseSent
	}
}

func (m *mockWebSocketConnWithControl) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.writeError != nil {
		return m.writeError
	}

	if m.closed.Load() {
		return websocket.ErrCloseSent
	}

	m.writtenData = append(m.writtenData, data)
	return nil
}

func (m *mockWebSocketConnWithControl) Close() error {
	m.closed.Store(true)
	if m.closeCallback != nil {
		m.closeCallback()
	}
	return nil
}

func (m *mockWebSocketConnWithControl) SetReadDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readDeadline = t
	return nil
}

func (m *mockWebSocketConnWithControl) SetPingHandler(h func(appData string) error) {
	// No-op for mock
}

func (m *mockWebSocketConnWithControl) WriteControl(messageType int, data []byte, deadline time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writeError != nil {
		return m.writeError
	}
	return nil
}

func (m *mockWebSocketConnWithControl) GetWrittenData() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]byte, len(m.writtenData))
	copy(result, m.writtenData)
	return result
}

// mockWebSocketDialerWithControl provides control over dial behavior
type mockWebSocketDialerWithControl struct {
	conn          WebSocketConn
	dialError     error
	dialCalls     atomic.Int32
	dialDelay     time.Duration
	mu            sync.Mutex
	connSequence  []WebSocketConn
	errorSequence []error
}

func (m *mockWebSocketDialerWithControl) Dial(urlStr string, requestHeader http.Header) (WebSocketConn, *http.Response, error) {
	m.dialCalls.Add(1)

	if m.dialDelay > 0 {
		time.Sleep(m.dialDelay)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Use sequence if provided
	if len(m.errorSequence) > 0 {
		err := m.errorSequence[0]
		m.errorSequence = m.errorSequence[1:]
		if err != nil {
			return nil, nil, err
		}
	} else if m.dialError != nil {
		return nil, nil, m.dialError
	}

	if len(m.connSequence) > 0 {
		conn := m.connSequence[0]
		m.connSequence = m.connSequence[1:]
		return conn, nil, nil
	}

	return m.conn, nil, nil
}

// --- Tests for connection loops ---

func TestServerConnectionStartStop(t *testing.T) {
	mockConn := newMockWebSocketConnWithControl()
	mockDialer := &mockWebSocketDialerWithControl{conn: mockConn}

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.WithDialer(mockDialer)
	conn.SetHeartbeatInterval(50 * time.Millisecond)

	// Start should launch the connection loop
	conn.Start()

	// Give it time to connect and start loops
	time.Sleep(150 * time.Millisecond)

	// Stop should terminate the loops
	conn.Stop()

	// Connection should be closed
	if !mockConn.closed.Load() {
		t.Error("connection should be closed after Stop")
	}
}

func TestServerConnectionConnectionLoopWithDialError(t *testing.T) {
	mockDialer := &mockWebSocketDialerWithControl{
		dialError: errors.New("connection refused"),
	}

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.WithDialer(mockDialer)
	// Use very short reconnect strategy for testing
	conn.reconnectStrategy = NewReconnectStrategy(10*time.Millisecond, 50*time.Millisecond)

	conn.Start()

	// Wait for a few reconnect attempts
	time.Sleep(100 * time.Millisecond)

	conn.Stop()

	// Should have attempted to connect multiple times
	if mockDialer.dialCalls.Load() < 2 {
		t.Errorf("expected at least 2 dial attempts, got %d", mockDialer.dialCalls.Load())
	}
}

func TestServerConnectionConnectionLoopReconnectOnError(t *testing.T) {
	// First connection succeeds, then fails, then succeeds again
	mockConn1 := newMockWebSocketConnWithControl()
	mockConn2 := newMockWebSocketConnWithControl()

	mockDialer := &mockWebSocketDialerWithControl{
		connSequence: []WebSocketConn{mockConn1, mockConn2},
	}

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.WithDialer(mockDialer)
	conn.SetHeartbeatInterval(50 * time.Millisecond)
	conn.reconnectStrategy = NewReconnectStrategy(10*time.Millisecond, 50*time.Millisecond)
	// Use short init timeout for testing - initialization will timeout and trigger reconnect
	conn.SetInitTimeout(50 * time.Millisecond)

	conn.Start()

	// Wait for first connection and init timeout
	time.Sleep(100 * time.Millisecond)

	// Close first connection to trigger reconnect
	mockConn1.Close()

	// Wait for reconnect (after init timeout)
	time.Sleep(200 * time.Millisecond)

	conn.Stop()

	// Should have connected at least twice (first connect + reconnect after timeout)
	if mockDialer.dialCalls.Load() < 2 {
		t.Errorf("expected at least 2 dial calls, got %d", mockDialer.dialCalls.Load())
	}
}

func TestServerConnectionReadLoop(t *testing.T) {
	mockConn := newMockWebSocketConnWithControl()
	handler := &mockHandler{}

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.conn = mockConn
	conn.SetHandler(handler)

	// Send a create pod message
	reqData, _ := json.Marshal(CreatePodRequest{
		PodKey:        "test-pod",
		LaunchCommand: "claude-code",
	})
	msg := ProtocolMessage{
		Type: MsgTypeCreatePod,
		Data: reqData,
	}
	msgBytes, _ := json.Marshal(msg)
	mockConn.readMessages <- msgBytes

	// Run read loop in a goroutine
	done := make(chan struct{})
	go func() {
		conn.readLoop()
		close(done)
	}()

	// Wait for message to be processed
	select {
	case <-done:
		// Read loop exited
	case <-time.After(500 * time.Millisecond):
		t.Error("read loop should have exited")
	}

	// Handler should have been called
	handler.mu.Lock()
	called := handler.createPodCalled
	podKey := handler.lastCreateReq.PodKey
	handler.mu.Unlock()

	if !called {
		t.Error("OnCreatePod should have been called")
	}

	if podKey != "test-pod" {
		t.Errorf("PodKey: got %v, want test-pod", podKey)
	}
}

func TestServerConnectionReadLoopInvalidJSON(t *testing.T) {
	mockConn := newMockWebSocketConnWithControl()
	handler := &mockHandler{}

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.conn = mockConn
	conn.SetHandler(handler)

	// Send invalid JSON
	mockConn.readMessages <- []byte("invalid json{")

	// Run read loop
	done := make(chan struct{})
	go func() {
		conn.readLoop()
		close(done)
	}()

	select {
	case <-done:
		// Read loop exited
	case <-time.After(500 * time.Millisecond):
		t.Error("read loop should have exited")
	}

	// Handler should not have been called
	handler.mu.Lock()
	called := handler.createPodCalled
	handler.mu.Unlock()

	if called {
		t.Error("OnCreatePod should not have been called for invalid JSON")
	}
}

func TestServerConnectionWriteLoop(t *testing.T) {
	mockConn := newMockWebSocketConnWithControl()

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.conn = mockConn

	done := make(chan struct{})

	// Start write loop
	go conn.writeLoop(done)

	// Send a message
	conn.Send(ProtocolMessage{
		Type: MsgTypeHeartbeat,
		Data: json.RawMessage(`{"node_id": "test"}`),
	})

	// Wait for message to be written
	time.Sleep(50 * time.Millisecond)

	// Stop the write loop
	close(done)

	// Check that message was written
	written := mockConn.GetWrittenData()
	if len(written) == 0 {
		t.Error("expected at least one message to be written")
	}
}

func TestServerConnectionWriteLoopStopChannel(t *testing.T) {
	mockConn := newMockWebSocketConnWithControl()

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.conn = mockConn

	done := make(chan struct{})

	// Start write loop
	loopDone := make(chan struct{})
	go func() {
		conn.writeLoop(done)
		close(loopDone)
	}()

	// Stop via stopCh
	conn.Stop()

	select {
	case <-loopDone:
		// Write loop exited
	case <-time.After(500 * time.Millisecond):
		t.Error("write loop should have exited")
	}
}

func TestServerConnectionWriteLoopWriteError(t *testing.T) {
	mockConn := newMockWebSocketConnWithControl()
	mockConn.writeError = errors.New("write error")

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.conn = mockConn

	done := make(chan struct{})

	// Start write loop
	loopDone := make(chan struct{})
	go func() {
		conn.writeLoop(done)
		close(loopDone)
	}()

	// Send a message that will fail
	conn.Send(ProtocolMessage{Type: MsgTypeHeartbeat})

	select {
	case <-loopDone:
		// Write loop should exit on error
	case <-time.After(500 * time.Millisecond):
		t.Error("write loop should have exited on write error")
	}

	// Connection should be closed
	if !mockConn.closed.Load() {
		t.Error("connection should be closed on write error")
	}
}

func TestServerConnectionWriteLoopNilConnection(t *testing.T) {
	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	// conn.conn is nil

	done := make(chan struct{})

	// Start write loop
	loopDone := make(chan struct{})
	go func() {
		conn.writeLoop(done)
		close(loopDone)
	}()

	// Send a message - should not panic
	conn.Send(ProtocolMessage{Type: MsgTypeHeartbeat})

	// Wait a bit for processing
	time.Sleep(50 * time.Millisecond)

	// Stop the loop
	close(done)

	select {
	case <-loopDone:
		// Write loop exited
	case <-time.After(500 * time.Millisecond):
		t.Error("write loop should have exited")
	}
}

func TestServerConnectionHeartbeatLoop(t *testing.T) {
	mockConn := newMockWebSocketConnWithControl()
	handler := &mockHandler{
		pods: []PodInfo{
			{PodKey: "pod-1", Status: "running"},
		},
	}

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.conn = mockConn
	conn.SetHandler(handler)
	conn.SetHeartbeatInterval(30 * time.Millisecond)

	done := make(chan struct{})

	// Start heartbeat loop
	go conn.heartbeatLoop(done)

	// Wait for at least 2 heartbeats
	time.Sleep(100 * time.Millisecond)

	// Stop the loop
	close(done)

	// Check that heartbeats were sent
	// Note: messages are sent to sendCh, then writeLoop writes them
	// Since we don't have writeLoop running, messages are in sendCh
	heartbeatCount := 0
	for {
		select {
		case <-conn.sendCh:
			heartbeatCount++
		default:
			goto done
		}
	}
done:
	// Should have at least 2 heartbeats (initial + at least 1 from ticker)
	if heartbeatCount < 2 {
		t.Errorf("expected at least 2 heartbeats, got %d", heartbeatCount)
	}
}

func TestServerConnectionHeartbeatLoopStopChannel(t *testing.T) {
	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.SetHeartbeatInterval(100 * time.Millisecond)

	done := make(chan struct{})

	// Start heartbeat loop
	loopDone := make(chan struct{})
	go func() {
		conn.heartbeatLoop(done)
		close(loopDone)
	}()

	// Stop via stopCh
	conn.Stop()

	select {
	case <-loopDone:
		// Heartbeat loop exited
	case <-time.After(500 * time.Millisecond):
		t.Error("heartbeat loop should have exited")
	}
}

func TestServerConnectionSendHeartbeat(t *testing.T) {
	handler := &mockHandler{
		pods: []PodInfo{
			{PodKey: "pod-1", Status: "running", Pid: 12345},
		},
	}

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.SetHandler(handler)

	conn.sendHeartbeat()

	// Check the message in sendCh
	select {
	case msg := <-conn.sendCh:
		if msg.Type != MsgTypeHeartbeat {
			t.Errorf("Type: got %v, want %v", msg.Type, MsgTypeHeartbeat)
		}

		var data HeartbeatData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			t.Fatalf("failed to unmarshal heartbeat data: %v", err)
		}

		if data.NodeID != "test-node" {
			t.Errorf("NodeID: got %v, want test-node", data.NodeID)
		}

		if len(data.Pods) != 1 {
			t.Errorf("Pods length: got %v, want 1", len(data.Pods))
		}
	default:
		t.Error("expected heartbeat message in sendCh")
	}
}

func TestServerConnectionSendHeartbeatNilHandler(t *testing.T) {
	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	// handler is nil

	// Should not panic
	conn.sendHeartbeat()

	// Check the message
	select {
	case msg := <-conn.sendCh:
		var data HeartbeatData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			t.Fatalf("failed to unmarshal heartbeat data: %v", err)
		}

		// Pods should be nil when handler is nil
		if data.Pods != nil {
			t.Errorf("Pods should be nil, got %v", data.Pods)
		}
	default:
		t.Error("expected heartbeat message")
	}
}

func TestServerConnectionRunConnection(t *testing.T) {
	mockConn := newMockWebSocketConnWithControl()
	handler := &mockHandler{}

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.conn = mockConn
	conn.SetHandler(handler)
	conn.SetHeartbeatInterval(20 * time.Millisecond)
	// Use short init timeout - runConnection will exit after init timeout or connection close
	conn.SetInitTimeout(50 * time.Millisecond)

	// Run connection in a goroutine
	done := make(chan struct{})
	go func() {
		conn.runConnection()
		close(done)
	}()

	// Wait for init timeout to trigger exit
	time.Sleep(100 * time.Millisecond)

	// Close connection to ensure exit
	mockConn.Close()

	select {
	case <-done:
		// runConnection exited
	case <-time.After(500 * time.Millisecond):
		t.Error("runConnection should have exited")
	}
}

func TestServerConnectionSendWithBackpressureBlocking(t *testing.T) {
	// Create connection with a very small buffer
	conn := &ServerConnection{
		sendCh: make(chan ProtocolMessage, 1),
		stopCh: make(chan struct{}),
	}

	// Fill the buffer
	conn.sendCh <- ProtocolMessage{Type: MsgTypeHeartbeat}

	// SendWithBackpressure should block until there's room
	done := make(chan bool)
	go func() {
		result := conn.SendWithBackpressure(ProtocolMessage{Type: MsgTypeHeartbeat})
		done <- result
	}()

	// Verify it's blocking
	select {
	case <-done:
		t.Error("SendWithBackpressure should be blocking")
	case <-time.After(50 * time.Millisecond):
		// Expected: still blocking
	}

	// Make room in the buffer
	<-conn.sendCh

	// Now it should complete
	select {
	case result := <-done:
		if !result {
			t.Error("SendWithBackpressure should return true")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("SendWithBackpressure should have completed")
	}
}

func TestServerConnectionSendWithBackpressureStopped(t *testing.T) {
	// Create connection with a very small buffer
	conn := &ServerConnection{
		sendCh: make(chan ProtocolMessage, 1),
		stopCh: make(chan struct{}),
	}

	// Fill the buffer
	conn.sendCh <- ProtocolMessage{Type: MsgTypeHeartbeat}

	// SendWithBackpressure should return false when stopped
	done := make(chan bool)
	go func() {
		result := conn.SendWithBackpressure(ProtocolMessage{Type: MsgTypeHeartbeat})
		done <- result
	}()

	// Stop the connection
	close(conn.stopCh)

	// Should return false
	select {
	case result := <-done:
		if result {
			t.Error("SendWithBackpressure should return false when stopped")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("SendWithBackpressure should have completed")
	}
}

func TestServerConnectionIntegration(t *testing.T) {
	// This test verifies the full connection lifecycle:
	// Start -> Connect -> ReadLoop processes message -> Stop
	//
	// Use a separate test for message routing since integration
	// with the connection loop has timing complexities.
	mockConn := newMockWebSocketConnWithControl()
	mockDialer := &mockWebSocketDialerWithControl{conn: mockConn}
	handler := &mockHandler{
		pods: []PodInfo{
			{PodKey: "pod-1", Status: "running"},
		},
	}

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.WithDialer(mockDialer)
	conn.SetHandler(handler)
	conn.SetHeartbeatInterval(30 * time.Millisecond)

	// Start connection
	conn.Start()

	// Wait for connection to be established
	time.Sleep(50 * time.Millisecond)

	// Verify connection was established
	if mockDialer.dialCalls.Load() < 1 {
		t.Error("dialer should have been called")
	}

	// Stop connection
	conn.Stop()

	// Connection should be closed
	if !mockConn.closed.Load() {
		t.Error("connection should be closed after Stop")
	}
}

// Test that multiple message types are routed correctly
func TestServerConnectionMultipleMessageTypes(t *testing.T) {
	mockConn := newMockWebSocketConnWithControl()
	handler := &mockHandler{}

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.conn = mockConn
	conn.SetHandler(handler)

	// Queue multiple messages
	messages := []ProtocolMessage{
		{
			Type: MsgTypeCreatePod,
			Data: mustMarshal(CreatePodRequest{PodKey: "pod-1"}),
		},
		{
			Type: MsgTypeTerminatePod,
			Data: mustMarshal(TerminatePodRequest{PodKey: "pod-1"}),
		},
		{
			Type: MsgTypeTerminalInput,
			Data: mustMarshal(TerminalInputRequest{PodKey: "pod-1", Data: "test"}),
		},
		{
			Type: MsgTypeTerminalResize,
			Data: mustMarshal(TerminalResizeRequest{PodKey: "pod-1", Cols: 80, Rows: 24}),
		},
	}

	for _, msg := range messages {
		msgBytes, _ := json.Marshal(msg)
		mockConn.readMessages <- msgBytes
	}

	// Run read loop
	done := make(chan struct{})
	go func() {
		conn.readLoop()
		close(done)
	}()

	// Wait for processing
	<-done

	// Verify all handlers were called
	handler.mu.Lock()
	defer handler.mu.Unlock()

	if !handler.createPodCalled {
		t.Error("OnCreatePod should have been called")
	}
	if !handler.terminatePodCalled {
		t.Error("OnTerminatePod should have been called")
	}
	if !handler.terminalInputCalled {
		t.Error("OnTerminalInput should have been called")
	}
	if !handler.terminalResizeCalled {
		t.Error("OnTerminalResize should have been called")
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
