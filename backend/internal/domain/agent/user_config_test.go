package agent

import (
	"testing"
	"time"
)

func TestUserAgentConfig_TableName(t *testing.T) {
	config := UserAgentConfig{}
	if got := config.TableName(); got != "user_agent_configs" {
		t.Errorf("TableName() = %v, want %v", got, "user_agent_configs")
	}
}

func TestUserAgentConfig_ToResponse(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		config *UserAgentConfig
		check  func(*testing.T, *UserAgentConfigResponse)
	}{
		{
			name: "basic config without agent",
			config: &UserAgentConfig{
				ID: 1,
				UserID:      100,
				AgentSlug: "claude-code",
				ConfigValues: ConfigValues{
					"model":     "opus",
					"perm_mode": "plan",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			check: func(t *testing.T, resp *UserAgentConfigResponse) {
				if resp.ID != 1 {
					t.Errorf("ID = %d, want 1", resp.ID)
				}
				if resp.UserID != 100 {
					t.Errorf("UserID = %d, want 100", resp.UserID)
				}
				if resp.AgentSlug != "claude-code" {
					t.Errorf("AgentSlug = %s, want claude-code", resp.AgentSlug)
				}
				if resp.ConfigValues["model"] != "opus" {
					t.Errorf("ConfigValues[model] = %v, want opus", resp.ConfigValues["model"])
				}
				if resp.ConfigValues["perm_mode"] != "plan" {
					t.Errorf("ConfigValues[perm_mode] = %v, want plan", resp.ConfigValues["perm_mode"])
				}
				if resp.AgentName != "" {
					t.Errorf("AgentName should be empty, got %s", resp.AgentName)
				}
			},
		},
		{
			name: "config with agent",
			config: &UserAgentConfig{
				ID:           2,
				UserID:       200,
				AgentSlug:    "codex-cli",
				ConfigValues: ConfigValues{"key": "value"},
				CreatedAt:    now,
				UpdatedAt:    now,
				Agent: &Agent{
					Slug: "codex-cli",
					Name: "Codex CLI",
				},
			},
			check: func(t *testing.T, resp *UserAgentConfigResponse) {
				if resp.AgentName != "Codex CLI" {
					t.Errorf("AgentName = %s, want Codex CLI", resp.AgentName)
				}
			},
		},
		{
			name: "empty config values",
			config: &UserAgentConfig{
				ID:           3,
				UserID:       300,
				AgentSlug:    "aider",
				ConfigValues: ConfigValues{},
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			check: func(t *testing.T, resp *UserAgentConfigResponse) {
				if len(resp.ConfigValues) != 0 {
					t.Errorf("ConfigValues should be empty, got %v", resp.ConfigValues)
				}
			},
		},
		{
			name: "nil config values",
			config: &UserAgentConfig{
				ID:           4,
				UserID:       400,
				AgentSlug: "opencode",
				ConfigValues: nil,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			check: func(t *testing.T, resp *UserAgentConfigResponse) {
				if resp.ConfigValues != nil && len(resp.ConfigValues) != 0 {
					t.Errorf("ConfigValues should be nil or empty, got %v", resp.ConfigValues)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.config.ToResponse()
			if resp == nil {
				t.Fatal("ToResponse() returned nil")
			}
			tt.check(t, resp)
		})
	}
}

func TestUserAgentConfig_ToResponse_TimeFormat(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	config := &UserAgentConfig{
		ID: 1,
		UserID: 1,
		AgentSlug: "claude-code",
		ConfigValues: ConfigValues{},
		CreatedAt:    testTime,
		UpdatedAt:    testTime,
	}

	resp := config.ToResponse()

	expectedFormat := testTime.Format(time.RFC3339)
	if resp.CreatedAt != expectedFormat {
		t.Errorf("CreatedAt = %s, want %s", resp.CreatedAt, expectedFormat)
	}
	if resp.UpdatedAt != expectedFormat {
		t.Errorf("UpdatedAt = %s, want %s", resp.UpdatedAt, expectedFormat)
	}
}

func TestUserAgentConfigResponse_Structure(t *testing.T) {
	// Test that the response struct has all expected fields
	resp := UserAgentConfigResponse{
		ID:           1,
		UserID:       100,
		AgentSlug:    "claude-code",
		AgentName:    "Claude Code",
		ConfigValues: map[string]interface{}{"model": "opus"},
		CreatedAt:    "2025-01-15T10:30:00Z",
		UpdatedAt:    "2025-01-15T10:30:00Z",
	}

	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
	if resp.UserID != 100 {
		t.Errorf("UserID = %d, want 100", resp.UserID)
	}
	if resp.AgentSlug != "claude-code" {
		t.Errorf("AgentSlug = %s, want claude-code", resp.AgentSlug)
	}
	if resp.AgentName != "Claude Code" {
		t.Errorf("AgentName = %s, want Claude Code", resp.AgentName)
	}
}
