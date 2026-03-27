package agentpod

import (
	"testing"
	"time"
)

// --- Test Status Constants ---

func TestPodStatusConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{StatusInitializing, "initializing"},
		{StatusRunning, "running"},
		{StatusPaused, "paused"},
		{StatusDisconnected, "disconnected"},
		{StatusOrphaned, "orphaned"},
		{StatusCompleted, "completed"},
		{StatusTerminated, "terminated"},
		{StatusError, "error"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
		}
	}
}

func TestAgentStatusConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{AgentStatusExecuting, "executing"},
		{AgentStatusWaiting, "waiting"},
		{AgentStatusIdle, "idle"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
		}
	}
}

func TestPermissionModeConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{PermissionModePlan, "plan"},
		{PermissionModeDefault, "default"},
		{PermissionModeBypass, "bypassPermissions"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
		}
	}
}

// --- Test Pod ---

func TestPodTableName(t *testing.T) {
	p := Pod{}
	if p.TableName() != "pods" {
		t.Errorf("expected 'pods', got %s", p.TableName())
	}
}

func TestPodIsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"running is active", StatusRunning, true},
		{"initializing is active", StatusInitializing, true},
		{"paused is active", StatusPaused, true},
		{"disconnected is active", StatusDisconnected, true},
		{"completed not active", StatusCompleted, false},
		{"terminated not active", StatusTerminated, false},
		{"orphaned not active", StatusOrphaned, false},
		{"error not active", StatusError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pod{Status: tt.status}
			if p.IsActive() != tt.expected {
				t.Errorf("expected IsActive() = %v, got %v", tt.expected, p.IsActive())
			}
		})
	}
}

func TestPodIsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"terminated is terminal", StatusTerminated, true},
		{"orphaned is terminal", StatusOrphaned, true},
		{"error is terminal", StatusError, true},
		{"running not terminal", StatusRunning, false},
		{"paused not terminal", StatusPaused, false},
		{"completed not terminal", StatusCompleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pod{Status: tt.status}
			if p.IsTerminal() != tt.expected {
				t.Errorf("expected IsTerminal() = %v, got %v", tt.expected, p.IsTerminal())
			}
		})
	}
}

func TestPodCanReconnect(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"disconnected can reconnect", StatusDisconnected, true},
		{"running cannot reconnect", StatusRunning, false},
		{"terminated cannot reconnect", StatusTerminated, false},
		{"completed cannot reconnect", StatusCompleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pod{Status: tt.status}
			if p.CanReconnect() != tt.expected {
				t.Errorf("expected CanReconnect() = %v, got %v", tt.expected, p.CanReconnect())
			}
		})
	}
}

func TestPodStruct(t *testing.T) {
	now := time.Now()
	model := "opus"
	permMode := "default"
	branch := "feature/test"

	p := Pod{
		ID: 1,
		OrganizationID: 100,
		PodKey:         "pod-123",
		RunnerID:       5,
		CreatedByID:    50,
		Status:         StatusRunning,
		AgentStatus:    AgentStatusExecuting,
		InitialPrompt:  "Test prompt",
		BranchName:     &branch,
		Model:          &model,
		PermissionMode: &permMode,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if p.ID != 1 {
		t.Errorf("expected ID 1, got %d", p.ID)
	}
	if p.PodKey != "pod-123" {
		t.Errorf("expected PodKey 'pod-123', got %s", p.PodKey)
	}
	if *p.Model != "opus" {
		t.Errorf("expected Model 'opus', got %s", *p.Model)
	}
}

// --- Test IsACPMode ---

func TestPodIsACPMode(t *testing.T) {
	tests := []struct {
		name            string
		interactionMode string
		expected        bool
	}{
		{"acp mode returns true", InteractionModeACP, true},
		{"pty mode returns false", InteractionModePTY, false},
		{"empty mode returns false", "", false},
		{"unknown mode returns false", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pod{InteractionMode: tt.interactionMode}
			if p.IsACPMode() != tt.expected {
				t.Errorf("IsACPMode() = %v, want %v", p.IsACPMode(), tt.expected)
			}
		})
	}
}

// --- Test PreparationConfig ---

func TestPreparationConfigStruct(t *testing.T) {
	config := PreparationConfig{
		Script:  "npm install",
		Timeout: 300,
	}

	if config.Script != "npm install" {
		t.Errorf("expected Script 'npm install', got %s", config.Script)
	}
	if config.Timeout != 300 {
		t.Errorf("expected Timeout 300, got %d", config.Timeout)
	}
}

// --- Test CreatePodCommand ---

func TestCreatePodCommandStruct(t *testing.T) {
	cmd := CreatePodCommand{
		PodKey:           "pod-456",
		InitialCommand:   "bash",
		InitialPrompt:    "Start working",
		PermissionMode:   "bypassPermissions",
		TicketSlug: "TICKET-123",
		PodSuffix:        "v1",
		EnvVars:          map[string]string{"FOO": "bar"},
		PreparationConfig: &PreparationConfig{
			Script:  "npm ci",
			Timeout: 120,
		},
	}

	if cmd.PodKey != "pod-456" {
		t.Errorf("expected PodKey 'pod-456', got %s", cmd.PodKey)
	}
	if cmd.TicketSlug != "TICKET-123" {
		t.Errorf("expected TicketSlug 'TICKET-123', got %s", cmd.TicketSlug)
	}
	if cmd.EnvVars["FOO"] != "bar" {
		t.Error("expected EnvVars['FOO'] = 'bar'")
	}
	if cmd.PreparationConfig.Script != "npm ci" {
		t.Errorf("expected PreparationConfig.Script 'npm ci', got %s", cmd.PreparationConfig.Script)
	}
}

// --- Test TerminatePodCommand ---

func TestTerminatePodCommandStruct(t *testing.T) {
	cmd := TerminatePodCommand{
		PodKey: "pod-789",
	}

	if cmd.PodKey != "pod-789" {
		t.Errorf("expected PodKey 'pod-789', got %s", cmd.PodKey)
	}
}

// --- Benchmark Tests ---

func BenchmarkPodIsActive(b *testing.B) {
	p := &Pod{Status: StatusRunning}
	for i := 0; i < b.N; i++ {
		p.IsActive()
	}
}

func BenchmarkPodIsTerminal(b *testing.B) {
	p := &Pod{Status: StatusTerminated}
	for i := 0; i < b.N; i++ {
		p.IsTerminal()
	}
}

func BenchmarkPodCanReconnect(b *testing.B) {
	p := &Pod{Status: StatusDisconnected}
	for i := 0; i < b.N; i++ {
		p.CanReconnect()
	}
}
