package relay

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

type fakeLocalSocket struct {
	mu           sync.Mutex
	writes       [][]byte
	writeErr     error
	blockWrites  bool
	writeStarted chan struct{}
	startedOnce  sync.Once
	writeRelease chan struct{}
	releaseOnce  sync.Once
	closed       chan struct{}
	closeOnce    sync.Once
}

func newFakeLocalSocket(blockWrites bool) *fakeLocalSocket {
	return &fakeLocalSocket{
		blockWrites:  blockWrites,
		writeStarted: make(chan struct{}),
		writeRelease: make(chan struct{}),
		closed:       make(chan struct{}),
	}
}

func (s *fakeLocalSocket) ReadMessage() (int, []byte, error) {
	<-s.closed
	return 0, nil, net.ErrClosed
}

func (s *fakeLocalSocket) WriteMessage(_ int, data []byte) error {
	s.startedOnce.Do(func() { close(s.writeStarted) })
	if s.blockWrites {
		select {
		case <-s.writeRelease:
		case <-s.closed:
			return net.ErrClosed
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writeErr != nil {
		return s.writeErr
	}
	s.writes = append(s.writes, append([]byte(nil), data...))
	return nil
}

func (s *fakeLocalSocket) releaseWrites() {
	s.releaseOnce.Do(func() { close(s.writeRelease) })
}

func (s *fakeLocalSocket) SetReadLimit(int64)                {}
func (s *fakeLocalSocket) SetReadDeadline(time.Time) error   { return nil }
func (s *fakeLocalSocket) SetWriteDeadline(time.Time) error  { return nil }
func (s *fakeLocalSocket) SetPongHandler(func(string) error) {}
func (s *fakeLocalSocket) Close() error {
	s.closeOnce.Do(func() { close(s.closed) })
	return nil
}

func (s *fakeLocalSocket) writtenFrames() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	frames := make([][]byte, len(s.writes))
	for i := range s.writes {
		frames[i] = append([]byte(nil), s.writes[i]...)
	}
	return frames
}

func newLocalConnTestServer(t *testing.T, sockets ...*fakeLocalSocket) (*LocalServer, []*localConn) {
	t.Helper()
	lane := &localPodLane{
		expectedTokens: make(map[string]struct{}),
		handlers:       make(map[byte]func([]byte)),
		reqHandlers:    make(map[byte]RequestHandler),
		conns:          make(map[*localConn]struct{}),
	}
	server := NewLocalServer(nil)
	server.pods["pod-1"] = lane

	clients := make([]*localConn, 0, len(sockets))
	for _, socket := range sockets {
		client := newLocalConn(socket, lane, nil)
		lane.conns[client] = struct{}{}
		clients = append(clients, client)
	}
	t.Cleanup(func() {
		for _, client := range clients {
			client.close()
		}
		for _, client := range clients {
			client.waitWriter()
		}
	})
	return server, clients
}

func TestLocalServer_SlowConnectionDoesNotBlockHealthyPeer(t *testing.T) {
	slow := newFakeLocalSocket(true)
	healthy := newFakeLocalSocket(false)
	server, clients := newLocalConnTestServer(t, slow, healthy)

	if err := server.Send("pod-1", MsgTypeOutput, []byte("frame")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	select {
	case <-slow.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("slow connection writer did not start")
	}
	waitForFakeWrites(t, healthy, 1)

	frames := healthy.writtenFrames()
	if len(frames) != 1 || string(frames[0]) != string(EncodeMessage(MsgTypeOutput, []byte("frame"))) {
		t.Fatalf("healthy peer got %q", frames)
	}
	waitForQueueUsage(t, clients[1], 0, 0)
}

func TestLocalServer_ConnectionOverflowClosesOnlyThatConnection(t *testing.T) {
	slow := newFakeLocalSocket(true)
	server, clients := newLocalConnTestServer(t, slow)
	client := clients[0]
	payload := make([]byte, localWriteQueueByteBudget/2)

	if err := server.Send("pod-1", MsgTypeOutput, payload); err != nil {
		t.Fatalf("first Send: %v", err)
	}
	select {
	case <-slow.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("blocked writer did not start")
	}
	if err := server.Send("pod-1", MsgTypeOutput, payload); err != nil {
		t.Fatalf("overflow must remain connection-local, got Send error: %v", err)
	}

	waitForWriterExit(t, client)
	if server.IsPodConnected("pod-1") {
		t.Fatal("overflowed connection remained registered")
	}
	if bytes, items := client.queueUsage(); bytes != 0 || items != 0 {
		t.Fatalf("queue accounting leaked after overflow: bytes=%d items=%d", bytes, items)
	}
}

func TestLocalServer_WriteErrorDoesNotCloseHealthyPeer(t *testing.T) {
	broken := newFakeLocalSocket(false)
	broken.writeErr = errors.New("write failed")
	healthy := newFakeLocalSocket(false)
	server, clients := newLocalConnTestServer(t, broken, healthy)

	if err := server.Send("pod-1", MsgTypeOutput, []byte("frame")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitForWriterExit(t, clients[0])
	waitForFakeWrites(t, healthy, 1)

	if clients[1].isClosed() {
		t.Fatal("healthy peer closed after another connection's write error")
	}
	if got := len(server.lookupLane("pod-1").snapshotConns()); got != 1 {
		t.Fatalf("got %d registered connections after isolated write error, want 1", got)
	}
}

func TestLocalServer_FlushWaitsForSocketWriter(t *testing.T) {
	socket := newFakeLocalSocket(true)
	server, _ := newLocalConnTestServer(t, socket)

	if err := server.Send("pod-1", MsgTypeOutput, []byte("tail")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	select {
	case <-socket.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("socket writer did not start")
	}

	flushDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		flushDone <- server.Flush(ctx, "pod-1")
	}()
	select {
	case err := <-flushDone:
		t.Fatalf("Flush returned before WriteMessage completed: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	socket.releaseWrites()
	select {
	case err := <-flushDone:
		if err != nil {
			t.Fatalf("Flush: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Flush did not complete after socket writer was released")
	}
	frames := socket.writtenFrames()
	if len(frames) != 1 || string(frames[0]) != string(EncodeMessage(MsgTypeOutput, []byte("tail"))) {
		t.Fatalf("socket frames = %q", frames)
	}
}

func TestLocalServer_FlushWaitsForHealthyPeerAfterAnotherCloses(t *testing.T) {
	closing := newFakeLocalSocket(true)
	healthy := newFakeLocalSocket(true)
	server, clients := newLocalConnTestServer(t, closing, healthy)
	if err := server.Send("pod-1", MsgTypeOutput, []byte("tail")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	for name, started := range map[string]<-chan struct{}{
		"closing": closing.writeStarted,
		"healthy": healthy.writeStarted,
	} {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatalf("%s socket writer did not start", name)
		}
	}

	flushDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		flushDone <- server.Flush(ctx, "pod-1")
	}()
	waitForQueueDepth(t, clients[0], 1)
	waitForQueueDepth(t, clients[1], 1)
	clients[0].close()
	select {
	case err := <-flushDone:
		t.Fatalf("failed peer made Flush skip the healthy peer: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	healthy.releaseWrites()
	select {
	case err := <-flushDone:
		if err == nil {
			t.Fatal("Flush must report the connection that closed before its marker")
		}
	case <-time.After(time.Second):
		t.Fatal("Flush did not finish after healthy peer completed")
	}
	if frames := healthy.writtenFrames(); len(frames) != 1 {
		t.Fatalf("healthy peer wrote %d frames, want 1", len(frames))
	}
}

func bareLocalConn(capacity int) *localConn {
	return &localConn{
		socket: newFakeLocalSocket(false), queue: make(chan localWriteItem, capacity),
		done: make(chan struct{}), writerDone: make(chan struct{}),
	}
}

func waitForLocalQueueDepth(t *testing.T, queue chan localWriteItem, depth int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(queue) >= depth {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("local queue depth = %d, want %d", len(queue), depth)
}

func waitForFakeWrites(t *testing.T, socket *fakeLocalSocket, count int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(socket.writtenFrames()) >= count {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("socket received %d frames, want at least %d", len(socket.writtenFrames()), count)
}

func waitForQueueUsage(t *testing.T, client *localConn, bytes, items int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		gotBytes, gotItems := client.queueUsage()
		if gotBytes == bytes && gotItems == items {
			return
		}
		time.Sleep(time.Millisecond)
	}
	gotBytes, gotItems := client.queueUsage()
	t.Fatalf("queue usage=(%d,%d), want (%d,%d)", gotBytes, gotItems, bytes, items)
}

func waitForWriterExit(t *testing.T, client *localConn) {
	t.Helper()
	select {
	case <-client.writerDone:
	case <-time.After(time.Second):
		t.Fatal("connection writer did not exit")
	}
}

func waitForQueueDepth(t *testing.T, client *localConn, depth int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(client.queue) >= depth {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("connection queue depth=%d, want at least %d", len(client.queue), depth)
}
