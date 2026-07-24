package relay

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type laneShutdownResponseWriter struct {
	header   http.Header
	lane     *localPodLane
	peer     net.Conn
	peerDone chan struct{}
}

func (w *laneShutdownResponseWriter) Header() http.Header { return w.header }
func (w *laneShutdownResponseWriter) Write(data []byte) (int, error) {
	return len(data), nil
}
func (w *laneShutdownResponseWriter) WriteHeader(int) {}

func (w *laneShutdownResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	server, peer := net.Pipe()
	w.peer = peer
	w.lane.shutdown()
	go func() {
		_, _ = io.Copy(io.Discard, peer)
		close(w.peerDone)
	}()
	return server, bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server)), nil
}

func startTestServer(t *testing.T) *LocalServer {
	t.Helper()
	srv := NewLocalServer(nil)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if _, err := srv.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(srv.Stop)
	return srv
}

func dialClient(t *testing.T, srv *LocalServer, podKey, token string) (*websocket.Conn, *http.Response) {
	t.Helper()
	u, err := url.Parse(srv.URL())
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	q := u.Query()
	q.Set("pod", podKey)
	q.Set("token", token)
	u.RawQuery = q.Encode()
	c, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil && resp == nil {
		t.Fatalf("dial: %v", err)
	}
	return c, resp
}

func TestLocalServer_RejectsUnknownPod(t *testing.T) {
	srv := startTestServer(t)
	_, resp := dialClient(t, srv, "missing", "any")
	if resp == nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %v", resp)
	}
}

func TestLocalServerRejectsMissingParametersAndUpgradeFailure(t *testing.T) {
	srv := NewLocalServer(nil)

	missing := httptest.NewRecorder()
	srv.handleBrowser(missing, httptest.NewRequest(http.MethodGet, "/", nil))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing parameters status = %d", missing.Code)
	}

	srv.RegisterPod("pod-1", "token")
	upgrade := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?pod=pod-1&token=token", nil)
	srv.handleBrowser(upgrade, req)
	if upgrade.Code != http.StatusBadRequest {
		t.Fatalf("non-hijacking upgrade status = %d", upgrade.Code)
	}

	if err := srv.Flush(nil, "missing"); err != nil {
		t.Fatalf("nil-context empty Flush: %v", err)
	}
}

func TestLocalServerClosesUpgradedBrowserWhenLaneWasUnregistered(t *testing.T) {
	srv := NewLocalServer(nil)
	srv.RegisterPod("pod-1", "token")
	lane := srv.lookupLane("pod-1")
	writer := &laneShutdownResponseWriter{
		header:   make(http.Header),
		lane:     lane,
		peerDone: make(chan struct{}),
	}
	req := httptest.NewRequest(http.MethodGet, "/?pod=pod-1&token=token", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	srv.handleBrowser(writer, req)
	if writer.peer == nil {
		t.Fatal("request was not upgraded before lane shutdown")
	}
	defer writer.peer.Close()
	waitForClosedSignal(t, writer.peerDone, "upgraded socket was not closed after lane shutdown")
	if lane.conns != nil {
		t.Fatal("unregistered lane accepted an upgraded browser")
	}
}

func TestLocalServerPongRefreshesReadDeadline(t *testing.T) {
	srv := startTestServer(t)
	srv.RegisterPod("pod-1", "tok")
	processed := make(chan struct{})
	srv.SetMessageHandler("pod-1", MsgTypeInput, func([]byte) { close(processed) })
	c, _ := dialClient(t, srv, "pod-1", "tok")
	defer c.Close()

	if err := c.WriteControl(websocket.PongMessage, []byte("keepalive"), time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := c.WriteMessage(websocket.BinaryMessage, EncodeMessage(MsgTypeInput, nil)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Fatal("server did not continue reading after pong")
	}
}

func TestLocalServer_RejectsBadToken(t *testing.T) {
	srv := startTestServer(t)
	srv.RegisterPod("pod-1", "expected-token")
	_, resp := dialClient(t, srv, "pod-1", "wrong-token")
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %v", resp)
	}
}

func TestLocalServer_AcceptsMatchingTokenAndBroadcasts(t *testing.T) {
	srv := startTestServer(t)
	srv.RegisterPod("pod-1", "tok")
	c, _ := dialClient(t, srv, "pod-1", "tok")
	defer c.Close()

	if !waitFor(srv, "pod-1") {
		t.Fatal("server did not record the connection")
	}
	if err := srv.Send("pod-1", MsgTypeOutput, []byte("hello")); err != nil {
		t.Fatalf("Send: %v", err)
	}

	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	mt, payload, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if mt != websocket.BinaryMessage {
		t.Fatalf("expected binary, got %d", mt)
	}
	if len(payload) < 1 || payload[0] != MsgTypeOutput {
		t.Fatalf("unexpected message type byte: %v", payload)
	}
	if string(payload[1:]) != "hello" {
		t.Fatalf("payload mismatch: %q", payload[1:])
	}
}

func TestLocalServer_DispatchesIncomingToHandler(t *testing.T) {
	srv := startTestServer(t)
	srv.RegisterPod("pod-1", "tok")

	got := make(chan []byte, 1)
	srv.SetMessageHandler("pod-1", MsgTypeInput, func(payload []byte) {
		got <- append([]byte(nil), payload...)
	})

	c, _ := dialClient(t, srv, "pod-1", "tok")
	defer c.Close()

	frame := EncodeMessage(MsgTypeInput, []byte("ls\n"))
	if err := c.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	select {
	case payload := <-got:
		if string(payload) != "ls\n" {
			t.Fatalf("payload mismatch: %q", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler never fired")
	}
}

func TestLocalServer_IgnoresNonBinaryBrowserFrame(t *testing.T) {
	srv := startTestServer(t)
	srv.RegisterPod("pod-1", "tok")
	got := make(chan struct{}, 1)
	srv.SetMessageHandler("pod-1", MsgTypeInput, func([]byte) { got <- struct{}{} })
	c, _ := dialClient(t, srv, "pod-1", "tok")
	defer c.Close()

	if err := c.WriteMessage(websocket.TextMessage, []byte("ignored")); err != nil {
		t.Fatal(err)
	}
	if err := c.WriteMessage(websocket.BinaryMessage, EncodeMessage(MsgTypeInput, nil)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-got:
	case <-time.After(time.Second):
		t.Fatal("reader did not continue after non-binary frame")
	}
}

func TestLocalServer_UnregisterClosesActiveConns(t *testing.T) {
	srv := startTestServer(t)
	srv.RegisterPod("pod-1", "tok")
	c, _ := dialClient(t, srv, "pod-1", "tok")
	defer c.Close()
	if !waitFor(srv, "pod-1") {
		t.Fatal("client never connected")
	}
	lane := srv.lookupLane("pod-1")
	clients := lane.snapshotConns()
	srv.UnregisterPod("pod-1")
	for _, client := range clients {
		select {
		case <-client.writerDone:
		default:
			t.Fatal("UnregisterPod returned before connection writer exited")
		}
	}
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := c.ReadMessage(); err == nil {
		t.Fatal("expected read to fail after unregister")
	}
	if srv.IsPodConnected("pod-1") {
		t.Fatal("expected IsPodConnected=false after unregister")
	}
}

func TestLocalServer_BroadcastsToMultipleClients(t *testing.T) {
	srv := startTestServer(t)
	srv.RegisterPod("pod-1", "tok")
	c1, _ := dialClient(t, srv, "pod-1", "tok")
	defer c1.Close()
	c2, _ := dialClient(t, srv, "pod-1", "tok")
	defer c2.Close()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if connCount(srv, "pod-1") == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	_ = srv.Send("pod-1", MsgTypeOutput, []byte("x"))

	var wg sync.WaitGroup
	wg.Add(2)
	read := func(c *websocket.Conn) {
		defer wg.Done()
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, payload, err := c.ReadMessage()
		if err != nil {
			t.Errorf("ReadMessage: %v", err)
			return
		}
		if !strings.HasSuffix(string(payload), "x") {
			t.Errorf("payload mismatch: %q", payload)
		}
	}
	go read(c1)
	go read(c2)
	wg.Wait()
}

func TestLocalServer_RequestHandlerRepliesToRequesterOnly(t *testing.T) {
	srv := startTestServer(t)
	srv.RegisterPod("pod-1", "tok")
	srv.SetRequestHandler("pod-1", MsgTypeSnapshotRequest, func(_ []byte, reply ReplyFunc) {
		reply(MsgTypeAcpSnapshot, []byte("snap"))
		reply(MsgTypeAcpEvent, []byte("event"))
	})

	c1, _ := dialClient(t, srv, "pod-1", "tok")
	defer c1.Close()
	c2, _ := dialClient(t, srv, "pod-1", "tok")
	defer c2.Close()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if connCount(srv, "pod-1") == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := c1.WriteMessage(websocket.BinaryMessage, EncodeMessage(MsgTypeSnapshotRequest, nil)); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	// The requester receives the reply...
	c1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, payload, err := c1.ReadMessage()
	if err != nil {
		t.Fatalf("requester read: %v", err)
	}
	if len(payload) < 1 || payload[0] != MsgTypeAcpSnapshot || string(payload[1:]) != "snap" {
		t.Fatalf("requester got %v, want snapshot 'snap'", payload)
	}
	_, payload, err = c1.ReadMessage()
	if err != nil {
		t.Fatalf("requester second read: %v", err)
	}
	if len(payload) < 1 || payload[0] != MsgTypeAcpEvent || string(payload[1:]) != "event" {
		t.Fatalf("requester got %v, want ordered event reply", payload)
	}

	// ...and the already-synced peer must NOT — the reply is per-connection, so a
	// late joiner's request can't re-apply state to browsers already in sync.
	c2.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	if _, _, err := c2.ReadMessage(); err == nil {
		t.Fatal("non-requesting browser received the snapshot reply (must be per-connection)")
	}
}

func waitFor(srv *LocalServer, podKey string) bool {
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if srv.IsPodConnected(podKey) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func connCount(srv *LocalServer, podKey string) int {
	lane := srv.lookupLane(podKey)
	if lane == nil {
		return 0
	}
	return len(lane.snapshotConns())
}
