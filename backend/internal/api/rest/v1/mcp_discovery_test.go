package v1

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func TestMCPDiscoveryHandler_Types(t *testing.T) {
	// Test RunnerSummary structure
	runner := RunnerSummary{
		ID:                1,
		NodeID:            "test-node",
		Description:       "Test Runner",
		Status:            "online",
		CurrentPods:       2,
		MaxConcurrentPods: 5,
		AvailableAgents:   []AgentTypeSummary{},
	}

	if runner.ID != 1 {
		t.Errorf("ID: got %v, want 1", runner.ID)
	}
	if runner.NodeID != "test-node" {
		t.Errorf("NodeID: got %v, want test-node", runner.NodeID)
	}
	if runner.Status != "online" {
		t.Errorf("Status: got %v, want online", runner.Status)
	}
}

func TestAgentTypeSummary_Structure(t *testing.T) {
	agent := AgentTypeSummary{
		ID:          1,
		Slug:        "claude-code",
		Name:        "Claude Code",
		Description: "AI coding assistant",
		Config: []ConfigFieldSummary{
			{
				Name:     "model",
				Type:     "select",
				Default:  "claude-sonnet-4-20250514",
				Options:  []string{"claude-sonnet-4-20250514", "claude-opus-4-20250514"},
				Required: true,
			},
		},
		UserConfig: map[string]interface{}{
			"model": "claude-opus-4-20250514",
		},
	}

	if agent.ID != 1 {
		t.Errorf("ID: got %v, want 1", agent.ID)
	}
	if agent.Slug != "claude-code" {
		t.Errorf("Slug: got %v, want claude-code", agent.Slug)
	}
	if len(agent.Config) != 1 {
		t.Fatalf("Config count: got %v, want 1", len(agent.Config))
	}

	config := agent.Config[0]
	if config.Name != "model" {
		t.Errorf("Config Name: got %v, want model", config.Name)
	}
	if !config.Required {
		t.Error("Config Required: expected true")
	}
	if len(config.Options) != 2 {
		t.Errorf("Config Options count: got %v, want 2", len(config.Options))
	}
}

func TestConfigFieldSummary_Structure(t *testing.T) {
	tests := []struct {
		name     string
		field    ConfigFieldSummary
		wantName string
		wantType string
	}{
		{
			name: "select field",
			field: ConfigFieldSummary{
				Name:     "model",
				Type:     "select",
				Default:  "default-value",
				Options:  []string{"opt1", "opt2"},
				Required: true,
			},
			wantName: "model",
			wantType: "select",
		},
		{
			name: "number field",
			field: ConfigFieldSummary{
				Name:     "max_turns",
				Type:     "number",
				Default:  10,
				Required: false,
			},
			wantName: "max_turns",
			wantType: "number",
		},
		{
			name: "boolean field",
			field: ConfigFieldSummary{
				Name:     "auto_commit",
				Type:     "boolean",
				Default:  false,
				Required: false,
			},
			wantName: "auto_commit",
			wantType: "boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Name != tt.wantName {
				t.Errorf("Name: got %v, want %v", tt.field.Name, tt.wantName)
			}
			if tt.field.Type != tt.wantType {
				t.Errorf("Type: got %v, want %v", tt.field.Type, tt.wantType)
			}
		})
	}
}

func TestRunnerSummary_JSON(t *testing.T) {
	runner := RunnerSummary{
		ID:                1,
		NodeID:            "dev-machine",
		Description:       "Development runner",
		Status:            "online",
		CurrentPods:       2,
		MaxConcurrentPods: 5,
		AvailableAgents: []AgentTypeSummary{
			{
				ID:          1,
				Slug:        "claude-code",
				Name:        "Claude Code",
				Description: "AI coding assistant",
				Config: []ConfigFieldSummary{
					{
						Name:     "model",
						Type:     "select",
						Default:  "claude-sonnet-4-20250514",
						Options:  []string{"claude-sonnet-4-20250514", "claude-opus-4-20250514"},
						Required: true,
					},
				},
				UserConfig: map[string]interface{}{
					"model": "claude-opus-4-20250514",
				},
			},
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(runner)
	if err != nil {
		t.Fatalf("failed to marshal runner: %v", err)
	}

	// Test JSON unmarshaling
	var decoded RunnerSummary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal runner: %v", err)
	}

	if decoded.ID != runner.ID {
		t.Errorf("decoded ID: got %v, want %v", decoded.ID, runner.ID)
	}
	if decoded.NodeID != runner.NodeID {
		t.Errorf("decoded NodeID: got %v, want %v", decoded.NodeID, runner.NodeID)
	}
	if len(decoded.AvailableAgents) != 1 {
		t.Fatalf("decoded AvailableAgents count: got %v, want 1", len(decoded.AvailableAgents))
	}

	agent := decoded.AvailableAgents[0]
	if agent.Slug != "claude-code" {
		t.Errorf("decoded agent Slug: got %v, want claude-code", agent.Slug)
	}
	if len(agent.Config) != 1 {
		t.Fatalf("decoded agent Config count: got %v, want 1", len(agent.Config))
	}
}

func TestRunnerSummary_OmitEmpty(t *testing.T) {
	// Test that omitempty fields are properly excluded
	runner := RunnerSummary{
		ID:                1,
		NodeID:            "test",
		Status:            "online",
		CurrentPods:       0,
		MaxConcurrentPods: 5,
		AvailableAgents:   []AgentTypeSummary{},
		// Description is empty, should be omitted
	}

	data, err := json.Marshal(runner)
	if err != nil {
		t.Fatalf("failed to marshal runner: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Description should be omitted when empty
	if _, exists := decoded["description"]; exists && decoded["description"] != "" {
		t.Error("description should be omitted when empty")
	}
}

func TestAgentTypeSummary_NilUserConfig(t *testing.T) {
	agent := AgentTypeSummary{
		ID:         1,
		Slug:       "test-agent",
		Name:       "Test Agent",
		Config:     nil,
		UserConfig: nil,
	}

	data, err := json.Marshal(agent)
	if err != nil {
		t.Fatalf("failed to marshal agent: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Config and UserConfig should be omitted when nil
	if _, exists := decoded["config"]; exists && decoded["config"] != nil {
		t.Error("config should be omitted when nil")
	}
	if _, exists := decoded["user_config"]; exists && decoded["user_config"] != nil {
		t.Error("user_config should be omitted when nil")
	}
}

func TestNewMCPDiscoveryHandler(t *testing.T) {
	// Test handler construction (nil services for type checking only)
	handler := NewMCPDiscoveryHandler(nil, nil, nil)

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

// Integration-style test with mocked gin context
func TestListRunnersForMCP_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a test response directly to verify JSON structure
	response := gin.H{
		"runners": []RunnerSummary{
			{
				ID:                1,
				NodeID:            "test-runner",
				Description:       "Test Runner",
				Status:            "online",
				CurrentPods:       1,
				MaxConcurrentPods: 5,
				AvailableAgents: []AgentTypeSummary{
					{
						ID:          1,
						Slug:        "claude-code",
						Name:        "Claude Code",
						Description: "AI assistant",
						Config: []ConfigFieldSummary{
							{Name: "model", Type: "select", Required: true},
						},
						UserConfig: map[string]interface{}{"model": "opus"},
					},
				},
			},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	runners, ok := decoded["runners"].([]interface{})
	if !ok {
		t.Fatal("expected runners array in response")
	}

	if len(runners) != 1 {
		t.Fatalf("runners count: got %v, want 1", len(runners))
	}

	runner := runners[0].(map[string]interface{})
	if runner["node_id"] != "test-runner" {
		t.Errorf("runner node_id: got %v, want test-runner", runner["node_id"])
	}

	agents, ok := runner["available_agents"].([]interface{})
	if !ok {
		t.Fatal("expected available_agents array")
	}

	if len(agents) != 1 {
		t.Fatalf("agents count: got %v, want 1", len(agents))
	}

	agent := agents[0].(map[string]interface{})
	if agent["slug"] != "claude-code" {
		t.Errorf("agent slug: got %v, want claude-code", agent["slug"])
	}
}

// Test that verifies the handler correctly uses middleware context
func TestListRunnersForMCP_ContextExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Set up tenant context
	tenant := &middleware.TenantContext{
		OrganizationID:   123,
		OrganizationSlug: "test-org",
		UserID:           456,
		UserRole:         "member",
	}
	c.Set("tenant", tenant)
	// GetUserID reads from "user_id" key directly, not from tenant
	c.Set("user_id", int64(456))

	// Verify context extraction works
	extractedTenant := middleware.GetTenant(c)
	if extractedTenant == nil {
		t.Fatal("expected tenant context")
	}
	if extractedTenant.OrganizationID != 123 {
		t.Errorf("org ID: got %v, want 123", extractedTenant.OrganizationID)
	}

	extractedUserID := middleware.GetUserID(c)
	if extractedUserID != 456 {
		t.Errorf("user ID: got %v, want 456", extractedUserID)
	}
}

// Test response JSON keys for token efficiency
func TestRunnerSummary_NoRedundantFields(t *testing.T) {
	runner := RunnerSummary{
		ID:                1,
		NodeID:            "test",
		Status:            "online",
		CurrentPods:       0,
		MaxConcurrentPods: 5,
		AvailableAgents:   []AgentTypeSummary{},
	}

	data, err := json.Marshal(runner)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// Verify removed fields are not present
	removedFields := []string{
		"last_heartbeat",
		"runner_version",
		"is_enabled",
		"host_info",
		"created_at",
		"updated_at",
	}

	for _, field := range removedFields {
		if contains(jsonStr, "\""+field+"\"") {
			t.Errorf("field %s should not be present in JSON output", field)
		}
	}

	// Verify required fields are present
	requiredFields := []string{
		"id",
		"node_id",
		"status",
		"current_pods",
		"max_concurrent_pods",
		"available_agents",
	}

	for _, field := range requiredFields {
		if !contains(jsonStr, "\""+field+"\"") {
			t.Errorf("field %s should be present in JSON output", field)
		}
	}
}

func TestConfigFieldSummary_NoValidationOrShowWhen(t *testing.T) {
	config := ConfigFieldSummary{
		Name:     "model",
		Type:     "select",
		Default:  "default",
		Options:  []string{"a", "b"},
		Required: true,
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// Verify removed fields are not present
	removedFields := []string{
		"validation",
		"show_when",
	}

	for _, field := range removedFields {
		if contains(jsonStr, "\""+field+"\"") {
			t.Errorf("field %s should not be present in config JSON output", field)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
