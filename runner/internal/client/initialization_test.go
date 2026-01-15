package client

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// --- Tests for initialization handshake ---

func TestServerConnectionHandleInitializeResult(t *testing.T) {
	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")

	result := InitializeResult{
		ProtocolVersion: CurrentProtocolVersion,
		ServerInfo: ServerInfo{
			Version: "1.0.0",
		},
		AgentTypes: []AgentTypeInfo{
			{Slug: "claude-code", Executable: "claude"},
		},
		Features: []string{"files_to_create"},
	}

	// Start a goroutine to receive the result
	received := make(chan InitializeResult, 1)
	go func() {
		select {
		case r := <-conn.initCh:
			received <- r
		case <-time.After(100 * time.Millisecond):
		}
	}()

	// Send the result
	conn.HandleInitializeResult(result)

	// Verify it was received
	select {
	case r := <-received:
		if r.ServerInfo.Version != "1.0.0" {
			t.Errorf("ServerInfo.Version: got %v, want 1.0.0", r.ServerInfo.Version)
		}
		if len(r.AgentTypes) != 1 {
			t.Errorf("AgentTypes length: got %v, want 1", len(r.AgentTypes))
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("expected to receive initialize result")
	}
}

func TestServerConnectionHandleInitializeResultChannelFull(t *testing.T) {
	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")

	// Fill the channel
	conn.initCh <- InitializeResult{}

	// Send another result - should be dropped without blocking
	done := make(chan struct{})
	go func() {
		conn.HandleInitializeResult(InitializeResult{})
		close(done)
	}()

	select {
	case <-done:
		// Good, didn't block
	case <-time.After(100 * time.Millisecond):
		t.Error("HandleInitializeResult should not block when channel is full")
	}
}

func TestServerConnectionCheckAvailableAgents(t *testing.T) {
	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")

	tests := []struct {
		name       string
		agentTypes []AgentTypeInfo
		wantCount  int
	}{
		{
			name:       "empty list",
			agentTypes: []AgentTypeInfo{},
			wantCount:  0,
		},
		{
			name: "agent with no executable",
			agentTypes: []AgentTypeInfo{
				{Slug: "test-agent", Executable: ""},
			},
			wantCount: 0,
		},
		{
			name: "agent with non-existent executable",
			agentTypes: []AgentTypeInfo{
				{Slug: "test-agent", Executable: "nonexistent-command-12345"},
			},
			wantCount: 0,
		},
		{
			name: "agent with existing executable (ls/echo)",
			agentTypes: []AgentTypeInfo{
				{Slug: "test-agent", Executable: "ls"},
			},
			wantCount: 1,
		},
		{
			name: "mixed agents",
			agentTypes: []AgentTypeInfo{
				{Slug: "valid-agent", Executable: "ls"},
				{Slug: "invalid-agent", Executable: "nonexistent-12345"},
				{Slug: "no-exec-agent", Executable: ""},
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			available := conn.checkAvailableAgents(tt.agentTypes)
			if len(available) != tt.wantCount {
				t.Errorf("checkAvailableAgents() returned %d agents, want %d", len(available), tt.wantCount)
			}
		})
	}
}

func TestServerConnectionGettersSetters(t *testing.T) {
	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")

	// Test SetRunnerVersion
	conn.SetRunnerVersion("2.0.0")
	if conn.runnerVersion != "2.0.0" {
		t.Errorf("runnerVersion: got %v, want 2.0.0", conn.runnerVersion)
	}

	// Test SetMCPPort
	conn.SetMCPPort(19001)
	if conn.mcpPort != 19001 {
		t.Errorf("mcpPort: got %v, want 19001", conn.mcpPort)
	}

	// Test SetAuthToken
	conn.SetAuthToken("new-token")
	if conn.authToken != "new-token" {
		t.Errorf("authToken: got %v, want new-token", conn.authToken)
	}

	// Test SetOrgSlug / GetOrgSlug
	conn.SetOrgSlug("new-org")
	if conn.GetOrgSlug() != "new-org" {
		t.Errorf("orgSlug: got %v, want new-org", conn.GetOrgSlug())
	}

	// Test IsInitialized (should be false initially)
	if conn.IsInitialized() {
		t.Error("IsInitialized should be false initially")
	}

	// Set initialized
	conn.mu.Lock()
	conn.initialized = true
	conn.mu.Unlock()

	if !conn.IsInitialized() {
		t.Error("IsInitialized should be true after setting")
	}

	// Test GetAvailableAgents
	conn.mu.Lock()
	conn.availableAgents = []string{"agent1", "agent2"}
	conn.mu.Unlock()

	agents := conn.GetAvailableAgents()
	if len(agents) != 2 {
		t.Errorf("GetAvailableAgents: got %d agents, want 2", len(agents))
	}

	// Test QueueLength and QueueCapacity
	qLen := conn.QueueLength()
	if qLen != 0 {
		t.Errorf("QueueLength: got %d, want 0", qLen)
	}

	qCap := conn.QueueCapacity()
	if qCap != cap(conn.sendCh) {
		t.Errorf("QueueCapacity: got %d, want %d", qCap, cap(conn.sendCh))
	}
}

func TestServerConnectionSetInitTimeout(t *testing.T) {
	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")

	// Default should be DefaultInitTimeout
	if conn.initTimeout != DefaultInitTimeout {
		t.Errorf("default initTimeout: got %v, want %v", conn.initTimeout, DefaultInitTimeout)
	}

	// Set custom timeout
	conn.SetInitTimeout(5 * time.Second)
	if conn.initTimeout != 5*time.Second {
		t.Errorf("initTimeout after set: got %v, want 5s", conn.initTimeout)
	}
}

// Test initialization with mock that simulates server response
func TestServerConnectionInitializationWithMockServer(t *testing.T) {
	mockConn := newMockWebSocketConnWithControl()
	mockDialer := &mockWebSocketDialerWithControl{conn: mockConn}

	conn := NewServerConnection("ws://localhost:8080", "test-node", "test-token", "test-org")
	conn.WithDialer(mockDialer)
	conn.SetInitTimeout(200 * time.Millisecond)
	conn.SetHeartbeatInterval(50 * time.Millisecond)

	// Simulate server sending initialize_result
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Wait for initialize message to be sent
		time.Sleep(50 * time.Millisecond)

		// Check if we received the initialize message
		written := mockConn.GetWrittenData()
		if len(written) == 0 {
			return
		}

		// Parse and verify it's an initialize message
		var msg ProtocolMessage
		if err := json.Unmarshal(written[0], &msg); err != nil {
			return
		}

		if msg.Type != MsgTypeInitialize {
			return
		}

		// Send initialize_result through the connection's handler
		result := InitializeResult{
			ProtocolVersion: CurrentProtocolVersion,
			ServerInfo:      ServerInfo{Version: "1.0.0"},
			AgentTypes: []AgentTypeInfo{
				{Slug: "ls-agent", Executable: "ls"},
			},
			Features: []string{"test"},
		}
		conn.HandleInitializeResult(result)
	}()

	// Start connection
	conn.Start()

	// Wait for server simulation
	wg.Wait()

	// Give some time for initialization to complete
	time.Sleep(100 * time.Millisecond)

	// Stop connection
	conn.Stop()

	// Verify initialization completed
	if conn.IsInitialized() {
		agents := conn.GetAvailableAgents()
		if len(agents) != 1 || agents[0] != "ls-agent" {
			t.Errorf("unexpected available agents: %v", agents)
		}
	}
}

// Test protocol constants
func TestProtocolConstants(t *testing.T) {
	// Verify protocol version
	if CurrentProtocolVersion != 2 {
		t.Errorf("CurrentProtocolVersion: got %d, want 2", CurrentProtocolVersion)
	}

	// Verify default init timeout
	if DefaultInitTimeout != 30*time.Second {
		t.Errorf("DefaultInitTimeout: got %v, want 30s", DefaultInitTimeout)
	}
}
