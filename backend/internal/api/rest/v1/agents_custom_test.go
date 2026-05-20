package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// CreateCustomAgent slug validate runs before tenant/auth checks, so we
// can drive it with a bare handler and nil services. The bug class this
// guards: pre-Phase-1, callers could POST any string as the custom agent
// slug and the service would happily INSERT it, breaking downstream
// /agents/{slug} routes.

func TestCreateCustomAgent_RejectsSlugWithDot(t *testing.T) {
	h := &AgentHandler{}
	router := gin.New()
	router.POST("/agents/custom", h.CreateCustomAgent)

	body, _ := json.Marshal(map[string]string{
		"slug":           "my.bad.agent",
		"name":           "Test",
		"launch_command": "echo hi",
	})
	req := httptest.NewRequest(http.MethodPost, "/agents/custom", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body: %s)", err, w.Body.String())
	}
	if resp["code"] != "VALIDATION_FAILED" {
		t.Errorf("code = %v, want VALIDATION_FAILED (body: %s)", resp["code"], w.Body.String())
	}
	if resp["field"] != "slug" {
		t.Errorf("field = %v, want slug (body: %s)", resp["field"], w.Body.String())
	}
	if resp["suggestion"] != "my-bad-agent" {
		t.Errorf("suggestion = %v, want my-bad-agent (body: %s)", resp["suggestion"], w.Body.String())
	}
}

func TestCreateCustomAgent_RejectsUppercase(t *testing.T) {
	h := &AgentHandler{}
	router := gin.New()
	router.POST("/agents/custom", h.CreateCustomAgent)

	body, _ := json.Marshal(map[string]string{
		"slug":           "MyAgent",
		"name":           "Test",
		"launch_command": "echo hi",
	})
	req := httptest.NewRequest(http.MethodPost, "/agents/custom", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateCustomAgent_RejectsEmpty(t *testing.T) {
	h := &AgentHandler{}
	router := gin.New()
	router.POST("/agents/custom", h.CreateCustomAgent)

	body, _ := json.Marshal(map[string]string{
		"slug":           "",
		"name":           "Test",
		"launch_command": "echo hi",
	})
	req := httptest.NewRequest(http.MethodPost, "/agents/custom", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
