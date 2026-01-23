package session

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/relay/internal/protocol"
	"github.com/gorilla/websocket"
)

func TestSetRunnerConn(t *testing.T) {
	s := NewTerminalSession("s1", "p1", 30*time.Second, nil, nil)
	server, _, cleanup := newWSPair()
	defer cleanup()
	if server == nil { t.Skip("ws failed") }
	s.SetRunnerConn(server)
	if s.GetRunnerConn() == nil { t.Error("runner conn should be set") }
}

func TestSetRunnerConn_Reconnect(t *testing.T) {
	s := NewTerminalSession("s1", "p1", 30*time.Second, nil, nil)
	s.runnerMu.Lock()
	s.runnerDisconnected = true
	s.runnerReconnectTimer = time.AfterFunc(time.Hour, func() {})
	s.runnerMu.Unlock()
	server, _, cleanup := newWSPair()
	defer cleanup()
	if server == nil { t.Skip("ws failed") }
	s.SetRunnerConn(server)
	if s.IsRunnerDisconnected() { t.Error("should not be disconnected") }
}

func TestAddBrowser(t *testing.T) {
	s := NewTerminalSession("s1", "p1", 30*time.Second, nil, nil)
	s.bufferOutput(protocol.EncodeOutput([]byte("test")))
	s.runnerMu.Lock()
	s.runnerDisconnected = true
	s.runnerMu.Unlock()
	server, client, cleanup := newWSPair()
	defer cleanup()
	if server == nil { t.Skip("ws failed") }
	s.AddBrowser("b1", server)
	if s.BrowserCount() != 1 { t.Error("expected 1 browser") }
	client.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	for i := 0; i < 2; i++ { client.ReadMessage() }
}

func TestRemoveBrowser(t *testing.T) {
	s := NewTerminalSession("s1", "p1", 100*time.Millisecond, nil, nil)
	s.controllerMu.Lock()
	s.controllerID = "b1"
	s.controllerMu.Unlock()
	server, _, cleanup := newWSPair()
	defer cleanup()
	if server == nil { t.Skip("ws failed") }
	s.browsersMu.Lock()
	s.browsers["b1"] = &BrowserConn{ID: "b1", Conn: server}
	s.browsersMu.Unlock()
	s.RemoveBrowser("b1")
	if s.BrowserCount() != 0 { t.Error("browser should be removed") }
	s.controllerMu.RLock()
	if s.controllerID != "" { t.Error("control released") }
	s.controllerMu.RUnlock()
}

func TestRemoveBrowser_LastBrowser(t *testing.T) {
	var called int32
	s := NewTerminalSession("s1", "p1", 50*time.Millisecond, func(pk string) { atomic.StoreInt32(&called, 1) }, nil)
	server, _, cleanup := newWSPair()
	defer cleanup()
	if server == nil { t.Skip("ws failed") }
	s.browsersMu.Lock()
	s.browsers["b1"] = &BrowserConn{ID: "b1", Conn: server}
	s.browsersMu.Unlock()
	s.RemoveBrowser("b1")
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&called) == 0 { t.Error("onAllBrowsersGone called") }
}

func TestBroadcastToAllBrowsers(t *testing.T) {
	s := NewTerminalSession("s1", "p1", 30*time.Second, nil, nil)
	server, client, cleanup := newWSPair()
	defer cleanup()
	if server == nil { t.Skip("ws failed") }
	s.browsersMu.Lock()
	s.browsers["b1"] = &BrowserConn{ID: "b1", Conn: server}
	s.browsersMu.Unlock()
	s.BroadcastToAllBrowsers([]byte("test"))
	client.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	if _, data, _ := client.ReadMessage(); string(data) != "test" { t.Error("broadcast failed") }
}

func TestHandleRunnerDisconnect(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.RunnerReconnectTimeout = 50 * time.Millisecond
	s := NewTerminalSessionWithConfig("s1", "p1", cfg, nil, nil)
	server, _, cleanup := newWSPair()
	defer cleanup()
	if server == nil { t.Skip("ws failed") }
	s.runnerMu.Lock()
	s.runnerConn = server
	s.runnerMu.Unlock()
	s.handleRunnerDisconnect()
	if !s.IsRunnerDisconnected() || s.GetRunnerConn() != nil { t.Error("should be disconnected") }
}

func TestHandleRunnerDisconnect_Timeout(t *testing.T) {
	var closed int32
	cfg := DefaultSessionConfig()
	cfg.RunnerReconnectTimeout = 30 * time.Millisecond
	s := NewTerminalSessionWithConfig("s1", "p1", cfg, nil, func(id string) { atomic.StoreInt32(&closed, 1) })
	server, _, cleanup := newWSPair()
	defer cleanup()
	if server == nil { t.Skip("ws failed") }
	s.runnerMu.Lock()
	s.runnerConn = server
	s.runnerMu.Unlock()
	s.handleRunnerDisconnect()
	time.Sleep(80 * time.Millisecond)
	if atomic.LoadInt32(&closed) == 0 { t.Error("session should close") }
}

func TestClose_WithConnections(t *testing.T) {
	closed := false
	s := NewTerminalSession("s1", "p1", 30*time.Second, nil, func(id string) { closed = true })
	s1, _, c1 := newWSPair()
	defer c1()
	s2, _, c2 := newWSPair()
	defer c2()
	if s1 == nil || s2 == nil { t.Skip("ws failed") }
	s.runnerMu.Lock()
	s.runnerConn = s1
	s.runnerReconnectTimer = time.AfterFunc(time.Hour, func() {})
	s.runnerMu.Unlock()
	s.browsersMu.Lock()
	s.browsers["b1"] = &BrowserConn{ID: "b1", Conn: s2}
	s.browsersMu.Unlock()
	s.Close()
	if !s.IsClosed() || !closed || s.BrowserCount() != 0 { t.Error("close failed") }
}

func TestForwardBrowserToRunner_Input(t *testing.T) {
	s := NewTerminalSession("s1", "p1", 30*time.Second, nil, nil)
	rs, rc, c1 := newWSPair()
	defer c1()
	bs, bc, c2 := newWSPair()
	defer c2()
	if rs == nil || bs == nil { t.Skip("ws failed") }
	s.runnerMu.Lock()
	s.runnerConn = rs
	s.runnerMu.Unlock()
	s.browsersMu.Lock()
	s.browsers["b1"] = &BrowserConn{ID: "b1", Conn: bs}
	s.browsersMu.Unlock()
	go s.forwardBrowserToRunner("b1")
	bc.WriteMessage(websocket.BinaryMessage, protocol.EncodeInput([]byte("test")))
	rc.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	rc.ReadMessage()
}

func TestForwardBrowserToRunner_Ping(t *testing.T) {
	s := NewTerminalSession("s1", "p1", 30*time.Second, nil, nil)
	bs, bc, cleanup := newWSPair()
	defer cleanup()
	if bs == nil { t.Skip("ws failed") }
	s.browsersMu.Lock()
	s.browsers["b1"] = &BrowserConn{ID: "b1", Conn: bs}
	s.browsersMu.Unlock()
	go s.forwardBrowserToRunner("b1")
	bc.WriteMessage(websocket.BinaryMessage, protocol.EncodePing())
	bc.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	bc.ReadMessage()
}
