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
// Error Response Tests
// ===========================================

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
					apierr.InvalidInput(c, "Invalid provider ID")
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
					apierr.ResourceNotFound(c, "Provider not found")
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
						apierr.ValidationError(c, err.Error())
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
				if _, ok := resp["code"]; !ok {
					t.Error("expected 'code' field in error response")
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
