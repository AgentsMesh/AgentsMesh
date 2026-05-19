package runner

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

type RunnerStream interface {
	Send(msg *runnerv1.ServerMessage) error
	Recv() (*runnerv1.RunnerMessage, error)
	Context() context.Context
}

type GRPCConnection struct {
	RunnerID   int64
	Generation int64 // Unique monotonic ID for this connection instance
	NodeID     string
	OrgSlug    string
	Stream     RunnerStream

	ConnectedAt time.Time
	LastPing    time.Time
	lastPong    time.Time // Last downstream pong received time

	initialized     bool
	availableAgents []string

	localRelayURL string

	onlineEventSent atomic.Bool

	Send chan *runnerv1.ServerMessage

	closeOnce sync.Once
	closed    bool
	closeChan chan struct{}

	mu sync.RWMutex
}

func NewGRPCConnection(runnerID int64, generation int64, nodeID, orgSlug string, stream RunnerStream) *GRPCConnection {
	return &GRPCConnection{
		RunnerID:    runnerID,
		Generation:  generation,
		NodeID:      nodeID,
		OrgSlug:     orgSlug,
		Stream:      stream,
		ConnectedAt: time.Now(),
		LastPing:    time.Now(),
		Send:        make(chan *runnerv1.ServerMessage, 256),
		closeChan:   make(chan struct{}),
	}
}

func (c *GRPCConnection) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

func (c *GRPCConnection) SetInitialized(initialized bool, availableAgents []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.initialized = initialized
	c.availableAgents = availableAgents
}

func (c *GRPCConnection) GetAvailableAgents() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.availableAgents == nil {
		return nil
	}
	result := make([]string, len(c.availableAgents))
	copy(result, c.availableAgents)
	return result
}

func (c *GRPCConnection) SetLocalRelayURL(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.localRelayURL = url
}

func (c *GRPCConnection) GetLocalRelayURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.localRelayURL
}

func (c *GRPCConnection) UpdateLastPing() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastPing = time.Now()
}

func (c *GRPCConnection) GetLastPing() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LastPing
}

func (c *GRPCConnection) UpdateLastPong() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastPong = time.Now()
}

func (c *GRPCConnection) GetLastPong() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastPong
}

func (c *GRPCConnection) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

func (c *GRPCConnection) Close() {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()
		close(c.closeChan)
		close(c.Send)
		slog.Info("gRPC connection closed",
			"runner_id", c.RunnerID,
			"generation", c.Generation,
			"node_id", c.NodeID,
		)
	})
}

func (c *GRPCConnection) CloseChan() <-chan struct{} {
	return c.closeChan
}

func (c *GRPCConnection) SendMessage(msg *runnerv1.ServerMessage) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		slog.Warn("attempted send on closed connection",
			"runner_id", c.RunnerID,
			"generation", c.Generation,
		)
		return ErrConnectionClosed
	}

	select {
	case c.Send <- msg:
		return nil
	default:
		slog.Error("send buffer full, message dropped",
			"runner_id", c.RunnerID,
			"generation", c.Generation,
		)
		return ErrSendBufferFull
	}
}

func (c *GRPCConnection) IsOnlineEventSent() bool {
	return c.onlineEventSent.Load()
}

func (c *GRPCConnection) MarkOnlineEventSent() {
	c.onlineEventSent.Store(true)
}
