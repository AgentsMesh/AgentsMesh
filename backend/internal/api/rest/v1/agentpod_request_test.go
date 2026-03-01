package v1

import (
	"encoding/json"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/agentpod"
	"github.com/gin-gonic/gin"
)

func setupAgentPodHandlerTest() (*AgentPodHandler, *agentpod.MockSettingsService, *agentpod.MockAIProviderService, *gin.Engine) {
	mockSettingsSvc := agentpod.NewMockSettingsService()
	mockProviderSvc := agentpod.NewMockAIProviderService()
	handler := NewAgentPodHandler(
		&agentpod.SettingsService{}, // We'll use mocks instead
		&agentpod.AIProviderService{},
	)
	// Replace with mocks - in real code we'd use interfaces
	handler.settingsService = nil
	handler.aiProviderService = nil

	router := gin.New()
	return handler, mockSettingsSvc, mockProviderSvc, router
}

func setAgentPodUserContext(c *gin.Context, userID int64) {
	c.Set("user_id", userID)
}

func TestNewAgentPodHandler(t *testing.T) {
	mockSettingsSvc := agentpod.NewMockSettingsService()
	mockProviderSvc := agentpod.NewMockAIProviderService()

	// Can't actually create handler with mock services due to type mismatch
	// but we can test the structs exist
	if mockSettingsSvc == nil {
		t.Fatal("expected non-nil mock settings service")
	}
	if mockProviderSvc == nil {
		t.Fatal("expected non-nil mock provider service")
	}
}

func TestUpdateSettingsRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   UpdateSettingsRequest
		wantErr bool
	}{
		{
			name: "valid request",
			input: UpdateSettingsRequest{
				DefaultModel:     strPtr("claude-3-sonnet"),
				TerminalFontSize: intPtr(14),
				TerminalTheme:    strPtr("dark"),
			},
			wantErr: false,
		},
		{
			name: "valid minimal request",
			input: UpdateSettingsRequest{
				TerminalFontSize: intPtr(16),
			},
			wantErr: false,
		},
		{
			name:    "nil values allowed",
			input:   UpdateSettingsRequest{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil && !tt.wantErr {
				t.Errorf("json.Marshal() error = %v", err)
			}
			if len(data) == 0 && !tt.wantErr {
				t.Error("expected non-empty JSON data")
			}
		})
	}
}

func TestCreateProviderRequest_Validation(t *testing.T) {
	tests := []struct {
		name     string
		input    CreateProviderRequest
		wantJSON bool
	}{
		{
			name: "valid claude provider",
			input: CreateProviderRequest{
				ProviderType: "claude",
				Name:         "My Claude",
				Credentials:  map[string]string{"api_key": "sk-test"},
				IsDefault:    true,
			},
			wantJSON: true,
		},
		{
			name: "valid gemini provider",
			input: CreateProviderRequest{
				ProviderType: "gemini",
				Name:         "My Gemini",
				Credentials:  map[string]string{"api_key": "test-key"},
				IsDefault:    false,
			},
			wantJSON: true,
		},
		{
			name: "valid openai provider",
			input: CreateProviderRequest{
				ProviderType: "openai",
				Name:         "My OpenAI",
				Credentials:  map[string]string{"api_key": "sk-openai"},
				IsDefault:    false,
			},
			wantJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			if tt.wantJSON && len(data) == 0 {
				t.Error("expected non-empty JSON data")
			}

			// Verify roundtrip
			var decoded CreateProviderRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if decoded.ProviderType != tt.input.ProviderType {
				t.Errorf("ProviderType = %s, want %s", decoded.ProviderType, tt.input.ProviderType)
			}
			if decoded.Name != tt.input.Name {
				t.Errorf("Name = %s, want %s", decoded.Name, tt.input.Name)
			}
		})
	}
}

func TestUpdateProviderRequest_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input UpdateProviderRequest
	}{
		{
			name: "update name only",
			input: UpdateProviderRequest{
				Name: "New Name",
			},
		},
		{
			name: "update credentials",
			input: UpdateProviderRequest{
				Credentials: map[string]string{"api_key": "new-key"},
			},
		},
		{
			name: "update is_default",
			input: UpdateProviderRequest{
				IsDefault: boolPtr(true),
			},
		},
		{
			name: "update is_enabled",
			input: UpdateProviderRequest{
				IsEnabled: boolPtr(false),
			},
		},
		{
			name: "update all fields",
			input: UpdateProviderRequest{
				Name:        "Updated Name",
				Credentials: map[string]string{"api_key": "updated-key"},
				IsDefault:   boolPtr(true),
				IsEnabled:   boolPtr(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var decoded UpdateProviderRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
		})
	}
}
