package runner

import (
	"context"
	"sync"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// RunnerStream defines a type-safe interface for gRPC bidirectional stream
// between Backend and Runner.
//
// Send: Backend → Runner (*runnerv1.ServerMessage)
// Recv: Runner → Backend (*runnerv1.RunnerMessage)
type RunnerStream interface {
	// Send sends a ServerMessage to the Runner
	Send(msg *runnerv1.ServerMessage) error
	// Recv receives a RunnerMessage from the Runner
	Recv() (*runnerv1.RunnerMessage, error)
	// Context returns the stream context
	Context() context.Context
}

// GRPCConnection represents an active gRPC connection to a runner.
type GRPCConnection struct {
	RunnerID int64
	NodeID   string
	OrgSlug  string
	Stream   RunnerStream

	// Connection timestamps
	ConnectedAt time.Time
	LastPing    time.Time
	lastPong    time.Time // Last downstream pong received time

	// Initialization state
	initialized     bool
	availableAgents []string

	// Send channel for outgoing messages (type-safe)
	Send chan *runnerv1.ServerMessage

	// Close handling
	closeOnce sync.Once
	closed    bool
	closeChan chan struct{}

	mu sync.RWMutex
}

// NewGRPCConnection creates a new gRPC connection wrapper.
func NewGRPCConnection(runnerID int64, nodeID, orgSlug string, stream RunnerStream) *GRPCConnection {
	return &GRPCConnection{
		RunnerID:    runnerID,
		NodeID:      nodeID,
		OrgSlug:     orgSlug,
		Stream:      stream,
		ConnectedAt: time.Now(),
		LastPing:    time.Now(),
		Send:        make(chan *runnerv1.ServerMessage, 256),
		closeChan:   make(chan struct{}),
	}
}

// IsInitialized returns whether the connection has completed initialization.
func (c *GRPCConnection) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// SetInitialized sets the initialization state and available agents.
func (c *GRPCConnection) SetInitialized(initialized bool, availableAgents []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.initialized = initialized
	c.availableAgents = availableAgents
}

// GetAvailableAgents returns a copy of the available agents list.
// Returns a copy to prevent external modification of internal state.
func (c *GRPCConnection) GetAvailableAgents() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.availableAgents == nil {
		return nil
	}
	// Return a copy to prevent external modification
	result := make([]string, len(c.availableAgents))
	copy(result, c.availableAgents)
	return result
}

// UpdateLastPing updates the last ping time.
func (c *GRPCConnection) UpdateLastPing() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastPing = time.Now()
}

// GetLastPing returns the last ping time.
func (c *GRPCConnection) GetLastPing() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LastPing
}

// UpdateLastPong updates the last downstream pong time.
func (c *GRPCConnection) UpdateLastPong() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastPong = time.Now()
}

// GetLastPong returns the last downstream pong time.
func (c *GRPCConnection) GetLastPong() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastPong
}

// IsClosed returns whether the connection is closed.
func (c *GRPCConnection) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// Close closes the connection.
func (c *GRPCConnection) Close() {
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()
		close(c.closeChan)
		close(c.Send)
	})
}

// CloseChan returns the close channel for select.
func (c *GRPCConnection) CloseChan() <-chan struct{} {
	return c.closeChan
}

// SendMessage sends a message through the gRPC stream.
// This is non-blocking; message is queued to the Send channel.
// Holds RLock during the entire check-and-send to prevent Close() from
// closing the Send channel between the IsClosed check and the channel write.
func (c *GRPCConnection) SendMessage(msg *runnerv1.ServerMessage) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return ErrConnectionClosed
	}

	select {
	case c.Send <- msg:
		return nil
	default:
		return ErrSendBufferFull
	}
}

// Note: All connection errors are now defined in errors.go for consistency.
// Use ErrConnectionClosed, ErrSendBufferFull, ErrRunnerNotConnected, ErrCommandSenderNotSet from there.
