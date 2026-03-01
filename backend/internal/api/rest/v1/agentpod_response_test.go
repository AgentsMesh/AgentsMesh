package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Response Format Tests
// ===========================================

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
			apierr.ValidationError(c, err.Error())
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
