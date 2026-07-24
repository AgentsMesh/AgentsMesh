package relay

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

func TestLocalConnClosedAndQueueAdmissionFailures(t *testing.T) {
	closed := bareLocalConn(1)
	closed.closed = true
	if closed.enqueue([]byte("frame")) {
		t.Fatal("closed connection accepted a frame")
	}
	//nolint:staticcheck // Explicit nil exercises local Flush context normalization.
	if err := closed.flush(nil); !errors.Is(err, errLocalConnClosed) {
		t.Fatalf("closed Flush = %v", err)
	}

	full := bareLocalConn(1)
	full.queue <- localWriteItem{frame: []byte("unaccounted")}
	if full.enqueue([]byte("frame")) {
		t.Fatal("physically full queue accepted a frame")
	}
	if !full.isClosed() {
		t.Fatal("physically full queue did not isolate its connection")
	}
}

func TestLocalConnFlushCancellationStages(t *testing.T) {
	t.Run("while queue is full", func(t *testing.T) {
		conn := bareLocalConn(1)
		conn.queue <- localWriteItem{frame: []byte("full")}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := conn.flush(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("Flush = %v, want context canceled", err)
		}
	})

	t.Run("connection closes while queue is full", func(t *testing.T) {
		conn := bareLocalConn(1)
		conn.queue <- localWriteItem{frame: []byte("full")}
		close(conn.done)
		if err := conn.flush(context.Background()); !errors.Is(err, errLocalConnClosed) {
			t.Fatalf("Flush = %v, want connection closed", err)
		}
	})

	for _, tt := range []struct {
		name   string
		want   error
		signal func(context.CancelFunc, *localConn)
	}{
		{name: "context after marker", want: context.Canceled, signal: func(cancel context.CancelFunc, _ *localConn) { cancel() }},
		{name: "close after marker", want: errLocalConnClosed, signal: func(_ context.CancelFunc, conn *localConn) { close(conn.done) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			conn := bareLocalConn(1)
			ctx, cancel := context.WithCancel(context.Background())
			result := make(chan error, 1)
			go func() { result <- conn.flush(ctx) }()
			waitForLocalQueueDepth(t, conn.queue, 1)
			tt.signal(cancel, conn)
			if err := <-result; !errors.Is(err, tt.want) {
				t.Fatalf("Flush = %v, want %v", err, tt.want)
			}
		})
	}

	ack := make(chan error, 1)
	ack <- nil
	if err := preferLocalFlushAck(ack, errLocalConnClosed); err != nil {
		t.Fatalf("ready flush ack lost to fallback: %v", err)
	}
}

func TestLocalConnWriterObservesClosedItemAndLoggedFailure(t *testing.T) {
	conn := bareLocalConn(1)
	conn.closed = true
	ack := make(chan error, 1)
	conn.queue <- localWriteItem{flush: ack}
	conn.writeLoop()
	if err := <-ack; !errors.Is(err, errLocalConnClosed) {
		t.Fatalf("closed writer marker = %v", err)
	}

	socket := newFakeLocalSocket(false)
	logged := newLocalConn(socket, nil, slog.Default())
	logged.fail(errors.New("test failure"))
	waitForWriterExit(t, logged)
}
