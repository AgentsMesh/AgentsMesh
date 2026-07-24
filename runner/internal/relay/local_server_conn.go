package relay

import (
	"errors"
	"log/slog"
	"sync"
	"time"
)

const (
	localWriteQueueItems      = 256
	localWriteQueueByteBudget = 1 << 20
)

var (
	errLocalWriteQueueOverflow = errors.New("local relay connection write queue overflow")
	errLocalConnClosed         = errors.New("local relay connection closed before output flush completed")
)

type localWriteItem struct {
	frame []byte
	flush chan error
}

// localSocket is the one-reader/one-writer WebSocket surface used by a local
// browser connection. Only localConn.writeLoop calls WriteMessage.
type localSocket interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	SetReadLimit(limit int64)
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	SetPongHandler(h func(appData string) error)
	Close() error
}

// localConn owns the bounded outbound lane for one browser connection.
// queuedBytes and queuedItems include the frame currently being written, so a
// blocked writer cannot make its queue budget appear available again.
type localConn struct {
	socket localSocket
	lane   *localPodLane
	logger *slog.Logger

	queue      chan localWriteItem
	done       chan struct{}
	writerDone chan struct{}
	closeOnce  sync.Once
	queueMu    sync.Mutex

	mu          sync.Mutex
	closed      bool
	queuedBytes int
	queuedItems int
}

func newLocalConn(socket localSocket, lane *localPodLane, logger *slog.Logger) *localConn {
	c := &localConn{
		socket:     socket,
		lane:       lane,
		logger:     logger,
		queue:      make(chan localWriteItem, localWriteQueueItems),
		done:       make(chan struct{}),
		writerDone: make(chan struct{}),
	}
	go c.writeLoop()
	return c
}
