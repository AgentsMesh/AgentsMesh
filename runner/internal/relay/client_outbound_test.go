package relay

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestClientFlushPreflightFailures(t *testing.T) {
	disconnected := NewClient(context.Background(), "ws://relay", "pod", "token", nil)
	if err := disconnected.Flush(nil); err == nil || err.Error() != "not connected" {
		t.Fatalf("disconnected Flush = %v", err)
	}

	tests := []struct {
		name string
		want error
		stop bool
		done bool
		exit bool
	}{
		{name: "stopped", want: errRelayClientStopped, stop: true},
		{name: "connection done", want: errRelayFlushDisconnected, done: true},
		{name: "writer exited", want: errRelayFlushDisconnected, exit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _, done, exit := newFlushStateClient(1)
			if tt.stop {
				close(client.stopCh)
			}
			if tt.done {
				close(done)
			}
			if tt.exit {
				close(exit)
			}
			if err := client.Flush(context.Background()); !errors.Is(err, tt.want) {
				t.Fatalf("Flush = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestClientFlushQueueAdmissionFailures(t *testing.T) {
	t.Run("context cancelled", func(t *testing.T) {
		client, queue, _, _ := newFlushStateClient(1)
		queue <- outboundItem{data: []byte("full"), generation: client.outboundGeneration}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := client.Flush(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("Flush = %v, want context canceled", err)
		}
	})

	for _, tt := range []struct {
		name   string
		want   error
		signal func(*Client, chan struct{}, chan struct{})
	}{
		{name: "client stopped", want: errRelayClientStopped, signal: func(c *Client, _, _ chan struct{}) { close(c.stopCh) }},
		{name: "connection done", want: errRelayFlushDisconnected, signal: func(_ *Client, done, _ chan struct{}) { close(done) }},
		{name: "writer exited", want: errRelayFlushDisconnected, signal: func(_ *Client, _, exit chan struct{}) { close(exit) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client, queue, done, exit := newFlushStateClient(1)
			queue <- outboundItem{data: []byte("full"), generation: client.outboundGeneration}
			result := make(chan error, 1)
			go func() { result <- client.Flush(context.Background()) }()
			waitForOutboundMutexHeld(t, client)
			tt.signal(client, done, exit)
			if err := <-result; !errors.Is(err, tt.want) {
				t.Fatalf("Flush = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestClientFlushMarkerWaitFailures(t *testing.T) {
	for _, tt := range []struct {
		name   string
		want   error
		signal func(*Client, chan struct{}, chan struct{}, context.CancelFunc)
	}{
		{name: "context cancelled", want: context.Canceled, signal: func(_ *Client, _, _ chan struct{}, cancel context.CancelFunc) { cancel() }},
		{name: "client stopped", want: errRelayClientStopped, signal: func(c *Client, _, _ chan struct{}, _ context.CancelFunc) { close(c.stopCh) }},
		{name: "connection done", want: errRelayFlushDisconnected, signal: func(_ *Client, done, _ chan struct{}, _ context.CancelFunc) { close(done) }},
		{name: "writer exited", want: errRelayFlushDisconnected, signal: func(_ *Client, _, exit chan struct{}, _ context.CancelFunc) { close(exit) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client, queue, done, exit := newFlushStateClient(2)
			ctx, cancel := context.WithCancel(context.Background())
			result := make(chan error, 1)
			go func() { result <- client.Flush(ctx) }()
			waitForOutboundQueueDepth(t, queue, 1)
			tt.signal(client, done, exit, cancel)
			if err := <-result; !errors.Is(err, tt.want) {
				t.Fatalf("Flush = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestOutboundLifecycleDefensiveHelpers(t *testing.T) {
	client := NewClient(context.Background(), "ws://relay", "pod", "token", nil)
	client.stopped.Store(true)
	if client.activateOutbound(make(chan outboundItem, 1), make(chan struct{}), make(chan struct{})) {
		t.Fatal("stopped client activated an outbound generation")
	}

	discardOutbound(nil, errRelayGenerationEnded)
	signalConnectionDone(nil, &sync.Once{})
	signalConnectionDone(make(chan struct{}), nil)

	queue := make(chan outboundItem, 2)
	ack := make(chan error, 1)
	queue <- outboundItem{data: []byte("frame")}
	queue <- outboundItem{flush: ack}
	discardOutbound(queue, errRelayGenerationEnded)
	if err := <-ack; !errors.Is(err, errRelayGenerationEnded) {
		t.Fatalf("discarded marker = %v", err)
	}

	for _, signal := range []func(*Client, chan struct{}, chan struct{}){
		func(c *Client, _, _ chan struct{}) { close(c.stopCh) },
		func(_ *Client, done, _ chan struct{}) { close(done) },
		func(_ *Client, _, exit chan struct{}) { close(exit) },
	} {
		client, _, done, exit := newFlushStateClient(1)
		signal(client, done, exit)
		if err := client.send([]byte("frame")); err == nil {
			t.Fatal("closed outbound signal accepted a frame")
		}
	}

	staleClient, staleQueue, _, staleExit := newFlushStateClient(2)
	staleAck := make(chan error, 1)
	staleQueue <- outboundItem{flush: staleAck, generation: staleClient.outboundGeneration - 1}
	staleClient.wg.Add(1)
	go staleClient.writeLoop()
	if err := <-staleAck; !errors.Is(err, errRelayGenerationEnded) {
		t.Fatalf("stale generation marker = %v", err)
	}
	close(staleClient.stopCh)
	select {
	case <-staleExit:
	case <-time.After(time.Second):
		t.Fatal("stale generation writer did not exit")
	}

	nilExitClient := NewClient(context.Background(), "ws://relay", "pod", "token", nil)
	nilExitQueue := make(chan outboundItem, 1)
	nilExitClient.activateOutbound(nilExitQueue, make(chan struct{}), nil)
	nilExitClient.finishOutboundWriter(nilExitQueue, nilExitClient.outboundGeneration, nil)

	closedExit := make(chan struct{})
	close(closedExit)
	closedExitClient := NewClient(context.Background(), "ws://relay", "pod", "token", nil)
	closedExitQueue := make(chan outboundItem, 1)
	closedExitClient.activateOutbound(closedExitQueue, make(chan struct{}), closedExit)
	closedExitClient.finishOutboundWriter(
		closedExitQueue,
		closedExitClient.outboundGeneration,
		closedExit,
	)
}

func TestWriteLoopConnectionFailurePathsUseCurrentGenerationSignals(t *testing.T) {
	t.Run("queued frame after socket removal", func(t *testing.T) {
		client, queue, done, exit := newFlushStateClient(1)
		ping := make(chan time.Time)
		client.wg.Add(1)
		go client.writeLoopWithPing(ping)
		queue <- outboundItem{data: []byte("frame"), generation: client.outboundGeneration}

		waitForClosedSignal(t, exit, "writer did not exit after socket removal")
		waitForClosedSignal(t, done, "writer did not close the connection generation")
		client.wg.Wait()
	})

	t.Run("ping after socket removal", func(t *testing.T) {
		client, _, done, exit := newFlushStateClient(1)
		ping := make(chan time.Time, 1)
		client.wg.Add(1)
		go client.writeLoopWithPing(ping)
		ping <- time.Now()

		waitForClosedSignal(t, exit, "writer did not exit after ping found no socket")
		waitForClosedSignal(t, done, "ping did not close the connection generation")
		client.wg.Wait()
	})

	t.Run("ping write error", func(t *testing.T) {
		socket := newFakeLocalSocket(false)
		socket.writeErr = errors.New("ping failed")
		client, _, done, exit := newFlushStateClient(1)
		client.connMu.Lock()
		client.conn = socket
		client.connMu.Unlock()
		ping := make(chan time.Time, 1)
		client.wg.Add(1)
		go client.writeLoopWithPing(ping)
		ping <- time.Now()

		waitForClosedSignal(t, exit, "writer did not exit after ping write error")
		waitForClosedSignal(t, done, "ping error did not close the connection generation")
		waitForClosedSignal(t, socket.closed, "ping error did not close the socket")
		client.wg.Wait()
	})
}

func waitForClosedSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal(message)
	}
}

func newFlushStateClient(capacity int) (*Client, chan outboundItem, chan struct{}, chan struct{}) {
	client := NewClient(context.Background(), "ws://relay", "pod", "token", nil)
	queue := make(chan outboundItem, capacity)
	done := make(chan struct{})
	exit := make(chan struct{})
	client.activateOutbound(queue, done, exit)
	return client, queue, done, exit
}

func waitForOutboundMutexHeld(t *testing.T, client *Client) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if !client.outboundMu.TryLock() {
			return
		}
		client.outboundMu.Unlock()
		time.Sleep(time.Millisecond)
	}
	t.Fatal("Flush did not block on full outbound queue")
}

func waitForOutboundQueueDepth(t *testing.T, queue chan outboundItem, depth int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(queue) >= depth {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("outbound queue depth = %d, want %d", len(queue), depth)
}

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
