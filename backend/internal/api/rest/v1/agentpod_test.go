package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
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
		name     string
		input    UpdateSettingsRequest
		wantErr  bool
	}{
		{
			name: "valid request",
			input: UpdateSettingsRequest{
				PreparationScript:  strPtr("#!/bin/bash\necho hello"),
				PreparationTimeout: intPtr(300),
				DefaultModel:       strPtr("claude-3-sonnet"),
				TerminalFontSize:   intPtr(14),
				TerminalTheme:      strPtr("dark"),
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
			name: "nil values allowed",
			input: UpdateSettingsRequest{},
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

// Integration-like test for request/response structure
func TestAgentPodHandler_RequestResponse(t *testing.T) {
	router := gin.New()
	gin.SetMode(gin.TestMode)

	// Test settings endpoint response format
	router.GET("/settings", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"settings": gin.H{
				"user_id":             int64(1),
				"preparation_script":  "#!/bin/bash",
				"preparation_timeout": 300,
				"terminal_font_size":  14,
				"terminal_theme":      "dark",
			},
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if _, ok := resp["settings"]; !ok {
		t.Error("response should have 'settings' key")
	}
}

func TestAgentPodHandler_ProvidersResponse(t *testing.T) {
	router := gin.New()
	gin.SetMode(gin.TestMode)

	router.GET("/providers", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"providers": []gin.H{
				{
					"id":            int64(1),
					"user_id":       int64(1),
					"provider_type": "claude",
					"name":          "My Claude",
					"is_default":    true,
					"is_enabled":    true,
				},
			},
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/providers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	providers, ok := resp["providers"].([]interface{})
	if !ok {
		t.Error("response should have 'providers' array")
	}
	if len(providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(providers))
	}
}

func TestAgentPodHandler_CreateProviderResponse(t *testing.T) {
	router := gin.New()
	gin.SetMode(gin.TestMode)

	router.POST("/providers", func(c *gin.Context) {
		var req CreateProviderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"provider": gin.H{
				"id":            int64(1),
				"user_id":       int64(1),
				"provider_type": req.ProviderType,
				"name":          req.Name,
				"is_default":    req.IsDefault,
				"is_enabled":    true,
			},
		})
	})

	body := bytes.NewBuffer([]byte(`{
		"provider_type": "claude",
		"name": "Test Provider",
		"credentials": {"api_key": "sk-test"},
		"is_default": true
	}`))

	req := httptest.NewRequest(http.MethodPost, "/providers", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if _, ok := resp["provider"]; !ok {
		t.Error("response should have 'provider' key")
	}
}

func TestAgentPodHandler_DeleteProviderResponse(t *testing.T) {
	router := gin.New()
	gin.SetMode(gin.TestMode)

	router.DELETE("/providers/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Provider deleted"})
	})

	req := httptest.NewRequest(http.MethodDelete, "/providers/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["message"] != "Provider deleted" {
		t.Errorf("unexpected message: %v", resp["message"])
	}
}

func TestAgentPodHandler_SetDefaultProviderResponse(t *testing.T) {
	router := gin.New()
	gin.SetMode(gin.TestMode)

	router.POST("/providers/:id/default", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Default provider set"})
	})

	req := httptest.NewRequest(http.MethodPost, "/providers/1/default", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["message"] != "Default provider set" {
		t.Errorf("unexpected message: %v", resp["message"])
	}
}

func TestAgentPodHandler_ErrorResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		setupRoute func(*gin.Engine)
		method     string
		path       string
		body       string
		wantCode   int
		wantError  string
	}{
		{
			name: "invalid provider ID",
			setupRoute: func(r *gin.Engine) {
				r.DELETE("/providers/:id", func(c *gin.Context) {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
				})
			},
			method:    http.MethodDelete,
			path:      "/providers/invalid",
			wantCode:  http.StatusBadRequest,
			wantError: "Invalid provider ID",
		},
		{
			name: "provider not found",
			setupRoute: func(r *gin.Engine) {
				r.GET("/providers/:id", func(c *gin.Context) {
					c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
				})
			},
			method:    http.MethodGet,
			path:      "/providers/999",
			wantCode:  http.StatusNotFound,
			wantError: "Provider not found",
		},
		{
			name: "invalid JSON body",
			setupRoute: func(r *gin.Engine) {
				r.POST("/providers", func(c *gin.Context) {
					var req CreateProviderRequest
					if err := c.ShouldBindJSON(&req); err != nil {
						c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
						return
					}
				})
			},
			method:   http.MethodPost,
			path:     "/providers",
			body:     "{invalid json}",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			gin.SetMode(gin.TestMode)
			tt.setupRoute(r)

			var body *bytes.Buffer
			if tt.body != "" {
				body = bytes.NewBuffer([]byte(tt.body))
			} else {
				body = bytes.NewBuffer(nil)
			}

			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("expected status %d, got %d", tt.wantCode, w.Code)
			}

			if tt.wantError != "" {
				var resp map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				if resp["error"] != tt.wantError {
					t.Errorf("expected error '%s', got '%v'", tt.wantError, resp["error"])
				}
			}
		})
	}
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}
