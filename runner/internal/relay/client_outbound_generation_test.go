package relay

import (
	"context"
	"testing"
	"time"
)

func TestClientFlushWaitsForSocketWriter(t *testing.T) {
	socket := newFakeLocalSocket(true)
	c := NewClient(context.Background(), "ws://localhost:8080", "pod-1", "test-token", nil)
	c.connMu.Lock()
	c.conn = socket
	c.connMu.Unlock()
	c.activateOutbound(c.sendCh, c.connDoneCh, c.writeExitCh)
	if !c.Start() {
		t.Fatal("Start returned false")
	}
	defer c.Stop()

	if err := c.Send(MsgTypeOutput, []byte("tail")); err != nil {
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
		flushDone <- c.Flush(ctx)
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

func TestClientStopFailsPendingFlush(t *testing.T) {
	socket := newFakeLocalSocket(true)
	c := NewClient(context.Background(), "ws://localhost:8080", "pod-1", "test-token", nil)
	c.connMu.Lock()
	c.conn = socket
	c.connMu.Unlock()
	c.activateOutbound(c.sendCh, c.connDoneCh, c.writeExitCh)
	if !c.Start() {
		t.Fatal("Start returned false")
	}

	if err := c.Send(MsgTypeOutput, []byte("tail")); err != nil {
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
		flushDone <- c.Flush(ctx)
	}()
	waitForClientQueueDepth(t, c, 1)

	stopDone := make(chan struct{})
	go func() {
		c.Stop()
		close(stopDone)
	}()
	select {
	case err := <-flushDone:
		if err == nil {
			t.Fatal("pending Flush succeeded after Stop")
		}
	case <-time.After(time.Second):
		t.Fatal("pending Flush did not fail after Stop")
	}
	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("Stop did not finish after interrupting the socket writer")
	}
}

func TestClientOutboundGenerationDoesNotCrossReconnect(t *testing.T) {
	oldSocket := newFakeLocalSocket(true)
	c := NewClient(context.Background(), "ws://localhost:8080", "pod-1", "test-token", nil)
	c.connMu.Lock()
	c.conn = oldSocket
	c.connMu.Unlock()
	c.activateOutbound(c.sendCh, c.connDoneCh, c.writeExitCh)
	c.wg.Add(1)
	go c.writeLoop()

	if err := c.Send(MsgTypeOutput, []byte("old-writing")); err != nil {
		t.Fatalf("first old Send: %v", err)
	}
	select {
	case <-oldSocket.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("old socket writer did not start")
	}
	if err := c.Send(MsgTypeOutput, []byte("old-queued")); err != nil {
		t.Fatalf("queued old Send: %v", err)
	}
	oldFlushDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		oldFlushDone <- c.Flush(ctx)
	}()
	waitForClientQueueDepth(t, c, 2)
	oldSocket.Close()
	select {
	case err := <-oldFlushDone:
		if err == nil {
			t.Fatal("old generation flush succeeded after its socket closed")
		}
	case <-time.After(time.Second):
		t.Fatal("old generation flush did not fail")
	}
	select {
	case <-c.writeExitCh:
	case <-time.After(time.Second):
		t.Fatal("old generation writer did not exit")
	}

	newSocket := newFakeLocalSocket(false)
	newDone := make(chan struct{})
	newExit := make(chan struct{})
	newQueue := make(chan outboundItem, 256)
	c.connMu.Lock()
	c.conn = newSocket
	c.connDoneCh = newDone
	c.writeExitCh = newExit
	c.connMu.Unlock()
	if !c.activateOutbound(newQueue, newDone, newExit) {
		t.Fatal("new outbound generation was rejected")
	}
	c.wg.Add(1)
	go c.writeLoop()
	if err := c.Send(MsgTypeOutput, []byte("fresh")); err != nil {
		t.Fatalf("fresh Send: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.Flush(ctx); err != nil {
		t.Fatalf("fresh Flush: %v", err)
	}
	frames := newSocket.writtenFrames()
	if len(frames) != 1 || string(frames[0]) != string(EncodeMessage(MsgTypeOutput, []byte("fresh"))) {
		t.Fatalf("new generation received stale frames: %q", frames)
	}
	c.Stop()
}

func TestMockClientFlushAndResetPreserveSynchronousContract(t *testing.T) {
	client := NewMockClient("ws://relay")
	if err := client.Connect(); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if !client.Start() {
		t.Fatal("Start failed")
	}
	if err := client.Send(MsgTypeOutput, []byte("accepted")); err != nil {
		t.Fatalf("Send: %v", err)
	}
	client.UpdateToken("rotated")
	if err := client.Flush(context.Background()); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if client.FlushCalls != 1 {
		t.Fatalf("FlushCalls = %d, want 1", client.FlushCalls)
	}

	client.Stop()
	client.Reset()
	if client.ConnectCalled || client.StartCalled || client.StopCalled || client.FlushCalls != 0 {
		t.Fatalf("Reset retained lifecycle tracking: %+v", client)
	}
	if len(client.UpdateTokenCalls) != 0 || len(client.SentMessages) != 0 {
		t.Fatalf("Reset retained outbound tracking: tokens=%v messages=%v", client.UpdateTokenCalls, client.SentMessages)
	}
}

func waitForClientQueueDepth(t *testing.T, c *Client, depth int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		c.outboundMu.Lock()
		queue := c.sendCh
		got := len(queue)
		c.outboundMu.Unlock()
		if got >= depth {
			return
		}
		time.Sleep(time.Millisecond)
	}
	c.outboundMu.Lock()
	got := len(c.sendCh)
	c.outboundMu.Unlock()
	t.Fatalf("client queue depth=%d, want at least %d", got, depth)
}
