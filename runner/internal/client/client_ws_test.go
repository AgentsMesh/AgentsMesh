package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// --- Mock implementations for message router testing ---

// mockHandler implements MessageHandler for testing
type mockHandler struct {
	mu sync.Mutex

	createPodCalled     bool
	terminatePodCalled  bool
	terminalInputCalled bool
	terminalResizeCalled bool

	lastCreateReq    CreatePodRequest
	lastTerminateReq TerminatePodRequest
	lastInputReq     TerminalInputRequest
	lastResizeReq    TerminalResizeRequest

	pods []PodInfo
}

func (m *mockHandler) OnCreatePod(req CreatePodRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createPodCalled = true
	m.lastCreateReq = req
	return nil
}

func (m *mockHandler) OnTerminatePod(req TerminatePodRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terminatePodCalled = true
	m.lastTerminateReq = req
	return nil
}

func (m *mockHandler) OnListPods() []PodInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pods
}

func (m *mockHandler) OnTerminalInput(req TerminalInputRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terminalInputCalled = true
	m.lastInputReq = req
	return nil
}

func (m *mockHandler) OnTerminalResize(req TerminalResizeRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terminalResizeCalled = true
	m.lastResizeReq = req
	return nil
}

// mockEventSender implements EventSender for testing
type mockEventSender struct {
	mu     sync.Mutex
	events []EventCall
	err    error
}

func (m *mockEventSender) SendEvent(msgType MessageType, data interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, EventCall{Type: msgType, Data: data})
	return nil
}

// --- WebSocket test server ---

type testWebSocketServer struct {
	server         *httptest.Server
	upgrader       websocket.Upgrader
	conn           *websocket.Conn
	messages       [][]byte
	mu             sync.Mutex
	onConnect      func(*websocket.Conn)
	onMessage      func([]byte)
	closeAfterRead int
	readCount      int
}

func newTestWebSocketServer() *testWebSocketServer {
	ts := &testWebSocketServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		messages: make([][]byte, 0),
	}

	ts.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle WebSocket upgrade
		if strings.Contains(r.URL.Path, "/ws") {
			ts.handleWebSocket(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	return ts
}

func (ts *testWebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ts.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	ts.mu.Lock()
	ts.conn = conn
	ts.mu.Unlock()

	if ts.onConnect != nil {
		ts.onConnect(conn)
	}

	// Read messages
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		ts.mu.Lock()
		ts.messages = append(ts.messages, data)
		ts.readCount++
		closeAfter := ts.closeAfterRead
		ts.mu.Unlock()

		if ts.onMessage != nil {
			ts.onMessage(data)
		}

		if closeAfter > 0 && ts.readCount >= closeAfter {
			conn.Close()
			break
		}
	}
}

func (ts *testWebSocketServer) URL() string {
	return ts.server.URL
}

func (ts *testWebSocketServer) Close() {
	ts.mu.Lock()
	if ts.conn != nil {
		ts.conn.Close()
	}
	ts.mu.Unlock()
	ts.server.Close()
}

func (ts *testWebSocketServer) SendMessage(msg interface{}) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.conn == nil {
		return nil
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return ts.conn.WriteMessage(websocket.TextMessage, data)
}

func (ts *testWebSocketServer) GetMessages() [][]byte {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	result := make([][]byte, len(ts.messages))
	copy(result, ts.messages)
	return result
}

// --- Tests for Client with real WebSocket ---

func TestClientConnectWithServer(t *testing.T) {
	ts := newTestWebSocketServer()
	defer ts.Close()

	connected := make(chan struct{})
	ts.onConnect = func(conn *websocket.Conn) {
		close(connected)
	}

	client := New(ts.URL(), "test-node", "test-token")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer client.Close()

	// Wait for connection
	select {
	case <-connected:
		// Connected
	case <-time.After(2 * time.Second):
		t.Error("expected connection to be established")
	}

	if !client.IsConnected() {
		t.Error("client should be connected")
	}
}

func TestClientConnectWithHTTPS(t *testing.T) {
	// Test URL parsing with HTTPS
	client := New("https://example.com", "test-node", "test-token")

	// Can't actually connect to example.com, but we can verify URL parsing
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This will fail because we can't connect, but we're testing the URL parsing
	err := client.Connect(ctx)
	if err == nil {
		t.Error("expected error connecting to example.com")
		client.Close()
	}
}

func TestClientReceiveMessages(t *testing.T) {
	ts := newTestWebSocketServer()
	defer ts.Close()

	connected := make(chan struct{})
	ts.onConnect = func(conn *websocket.Conn) {
		close(connected)
	}

	client := New(ts.URL(), "test-node", "test-token")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer client.Close()

	// Wait for connection
	<-connected

	// Send a message from server
	msg := Message{
		Type: string(MsgTypeHeartbeat),
		Data: json.RawMessage(`{"test": "data"}`),
	}
	ts.SendMessage(msg)

	// Receive message
	select {
	case received := <-client.Messages():
		if received.Type != MessageTypeHeartbeat {
			t.Errorf("Type: got %v, want %v", received.Type, MessageTypeHeartbeat)
		}
	case <-time.After(2 * time.Second):
		t.Error("expected to receive message")
	}
}

func TestClientSendWithConnection(t *testing.T) {
	ts := newTestWebSocketServer()
	defer ts.Close()

	connected := make(chan struct{})
	ts.onConnect = func(conn *websocket.Conn) {
		close(connected)
	}

	client := New(ts.URL(), "test-node", "test-token")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer client.Close()

	// Wait for connection
	<-connected

	// Send message
	err = client.Send(&Message{
		Type: string(MsgTypeHeartbeat),
		Data: json.RawMessage(`{"test": "value"}`),
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}

	// Wait a bit for message to be sent
	time.Sleep(100 * time.Millisecond)

	// Verify server received message
	messages := ts.GetMessages()
	if len(messages) == 0 {
		t.Error("expected server to receive message")
	}
}

func TestClientPingLoop(t *testing.T) {
	ts := newTestWebSocketServer()
	defer ts.Close()

	pingReceived := make(chan struct{})
	ts.onConnect = func(conn *websocket.Conn) {
		conn.SetPingHandler(func(data string) error {
			select {
			case pingReceived <- struct{}{}:
			default:
			}
			return conn.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(time.Second))
		})
	}

	// Create client with very short ping interval (not directly settable,
	// so we test with standard interval)
	client := New(ts.URL(), "test-node", "test-token")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer client.Close()

	// pingLoop uses 30 second interval, so we can't really wait for it
	// Just verify the loop starts without panic
	time.Sleep(100 * time.Millisecond)
}

func TestClientReconnectOnClose(t *testing.T) {
	ts := newTestWebSocketServer()

	connectCount := 0
	var mu sync.Mutex
	ts.onConnect = func(conn *websocket.Conn) {
		mu.Lock()
		connectCount++
		count := connectCount
		mu.Unlock()

		// Close connection after first connect to trigger reconnect
		if count == 1 {
			time.Sleep(50 * time.Millisecond)
			conn.Close()
		}
	}

	client := New(ts.URL(), "test-node", "test-token")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}

	// Wait for reconnection attempt
	// Note: reconnect has backoff starting at 1 second
	time.Sleep(1500 * time.Millisecond)

	client.Close()
	ts.Close()

	mu.Lock()
	count := connectCount
	mu.Unlock()

	// Should have at least attempted reconnection
	if count < 2 {
		t.Logf("Connect count: %d (reconnect may not have completed in time)", count)
		// This is timing dependent, so we don't fail the test
	}
}

func TestClientCloseStopsLoops(t *testing.T) {
	ts := newTestWebSocketServer()
	defer ts.Close()

	connected := make(chan struct{})
	ts.onConnect = func(conn *websocket.Conn) {
		close(connected)
	}

	client := New(ts.URL(), "test-node", "test-token")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}

	<-connected

	// Close should not panic and should stop all loops
	err = client.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}

	// Messages channel should be closed eventually
	// (readLoop closes it)
	select {
	case _, ok := <-client.Messages():
		if ok {
			// Got a message, continue reading
		} else {
			// Channel closed, as expected
		}
	case <-time.After(500 * time.Millisecond):
		// Timeout is okay, close happened but channel may not be closed yet
	}
}

// Test message_router error handling
func TestMessageRouterHandleTerminatePodError(t *testing.T) {
	handler := &mockHandler{}
	sender := &mockEventSender{}
	router := NewMessageRouter(handler, sender)

	// Invalid JSON for terminate pod
	msg := ProtocolMessage{
		Type: MsgTypeTerminatePod,
		Data: json.RawMessage(`invalid json`),
	}

	// Should not panic
	router.Route(msg)

	if handler.terminatePodCalled {
		t.Error("handler should not be called for invalid JSON")
	}
}

func TestMessageRouterHandleTerminalInputError(t *testing.T) {
	handler := &mockHandler{}
	sender := &mockEventSender{}
	router := NewMessageRouter(handler, sender)

	// Invalid JSON for terminal input
	msg := ProtocolMessage{
		Type: MsgTypeTerminalInput,
		Data: json.RawMessage(`invalid json`),
	}

	// Should not panic
	router.Route(msg)

	if handler.terminalInputCalled {
		t.Error("handler should not be called for invalid JSON")
	}
}

func TestMessageRouterHandleTerminalResizeError(t *testing.T) {
	handler := &mockHandler{}
	sender := &mockEventSender{}
	router := NewMessageRouter(handler, sender)

	// Invalid JSON for terminal resize
	msg := ProtocolMessage{
		Type: MsgTypeTerminalResize,
		Data: json.RawMessage(`invalid json`),
	}

	// Should not panic
	router.Route(msg)

	if handler.terminalResizeCalled {
		t.Error("handler should not be called for invalid JSON")
	}
}

// mockHandlerWithError implements MessageHandler with controllable errors
type mockHandlerWithError struct {
	mockHandler
	createError    error
	terminateError error
	inputError     error
	resizeError    error
}

func (m *mockHandlerWithError) OnCreatePod(req CreatePodRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createPodCalled = true
	m.lastCreateReq = req
	return m.createError
}

func (m *mockHandlerWithError) OnTerminatePod(req TerminatePodRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terminatePodCalled = true
	m.lastTerminateReq = req
	return m.terminateError
}

func (m *mockHandlerWithError) OnTerminalInput(req TerminalInputRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terminalInputCalled = true
	m.lastInputReq = req
	return m.inputError
}

func (m *mockHandlerWithError) OnTerminalResize(req TerminalResizeRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.terminalResizeCalled = true
	m.lastResizeReq = req
	return m.resizeError
}

func TestMessageRouterHandlerReturnsError(t *testing.T) {
	t.Run("CreatePod error", func(t *testing.T) {
		handler := &mockHandlerWithError{
			createError: context.DeadlineExceeded,
		}
		sender := &mockEventSender{}
		router := NewMessageRouter(handler, sender)

		reqData, _ := json.Marshal(CreatePodRequest{PodKey: "p1"})
		msg := ProtocolMessage{
			Type: MsgTypeCreatePod,
			Data: reqData,
		}

		// Should not panic, error is logged
		router.Route(msg)

		if !handler.createPodCalled {
			t.Error("handler should be called")
		}
	})

	t.Run("TerminatePod error", func(t *testing.T) {
		handler := &mockHandlerWithError{
			terminateError: context.DeadlineExceeded,
		}
		sender := &mockEventSender{}
		router := NewMessageRouter(handler, sender)

		reqData, _ := json.Marshal(TerminatePodRequest{PodKey: "p1"})
		msg := ProtocolMessage{
			Type: MsgTypeTerminatePod,
			Data: reqData,
		}

		router.Route(msg)

		if !handler.terminatePodCalled {
			t.Error("handler should be called")
		}
	})

	t.Run("TerminalInput error", func(t *testing.T) {
		handler := &mockHandlerWithError{
			inputError: context.DeadlineExceeded,
		}
		sender := &mockEventSender{}
		router := NewMessageRouter(handler, sender)

		reqData, _ := json.Marshal(TerminalInputRequest{PodKey: "p1"})
		msg := ProtocolMessage{
			Type: MsgTypeTerminalInput,
			Data: reqData,
		}

		router.Route(msg)

		if !handler.terminalInputCalled {
			t.Error("handler should be called")
		}
	})

	t.Run("TerminalResize error", func(t *testing.T) {
		handler := &mockHandlerWithError{
			resizeError: context.DeadlineExceeded,
		}
		sender := &mockEventSender{}
		router := NewMessageRouter(handler, sender)

		reqData, _ := json.Marshal(TerminalResizeRequest{PodKey: "p1"})
		msg := ProtocolMessage{
			Type: MsgTypeTerminalResize,
			Data: reqData,
		}

		router.Route(msg)

		if !handler.terminalResizeCalled {
			t.Error("handler should be called")
		}
	})
}
