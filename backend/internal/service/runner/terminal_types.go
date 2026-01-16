package runner

import (
	"context"
	"sync"

	"github.com/anthropics/agentsmesh/backend/internal/infra/terminal"
	"github.com/gorilla/websocket"
)

const (
	// DefaultScrollbackSize is the default scrollback buffer size (100KB)
	DefaultScrollbackSize = 100 * 1024

	// terminalShards is the number of shards for terminal data partitioning
	// 64 shards reduce lock contention for 500K pods
	terminalShards = 64

	// DefaultTerminalCols is the default terminal width
	DefaultTerminalCols = 80

	// DefaultTerminalRows is the default terminal height
	DefaultTerminalRows = 24

	// DefaultVirtualTerminalHistory is the default scrollback history lines
	DefaultVirtualTerminalHistory = 10000
)

// TerminalMessage represents a message to send to the frontend client
type TerminalMessage struct {
	Data   []byte
	IsJSON bool // true for JSON control messages, false for binary terminal output
}

// TerminalClient represents a frontend WebSocket client connected to a terminal
type TerminalClient struct {
	Conn   *websocket.Conn
	PodKey string
	Send   chan TerminalMessage
}

// PtySize represents the current PTY terminal size
type PtySize struct {
	Cols int
	Rows int
}

// PodInfoGetter interface for getting and updating pod information
type PodInfoGetter interface {
	GetPodOrganizationAndCreator(ctx context.Context, podKey string) (orgID, creatorID int64, err error)
	UpdatePodTitle(ctx context.Context, podKey, title string) error
}

// terminalShard holds a subset of terminal data with its own lock
type terminalShard struct {
	podRunnerMap      map[string]int64
	terminalClients   map[string]map[*TerminalClient]bool
	scrollbackBuffers map[string]*ScrollbackBuffer
	virtualTerminals  map[string]*terminal.VirtualTerminal
	ptySize           map[string]*PtySize
	mu                sync.RWMutex
}

// newTerminalShard creates a new terminal shard with initialized maps
func newTerminalShard() *terminalShard {
	return &terminalShard{
		podRunnerMap:      make(map[string]int64),
		terminalClients:   make(map[string]map[*TerminalClient]bool),
		scrollbackBuffers: make(map[string]*ScrollbackBuffer),
		virtualTerminals:  make(map[string]*terminal.VirtualTerminal),
		ptySize:           make(map[string]*PtySize),
	}
}
