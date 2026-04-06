package console

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// newTestServer creates a minimal Server for handler testing.
func newTestServer() *Server {
	return New(&config.Config{
		ServerURL: "https://test.example.com",
		NodeID:    "test-node",
		OrgSlug:   "test-org",
	}, 0, "test")
}

func TestHandleRestart_Returns501(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/actions/restart", nil)
	rec := httptest.NewRecorder()

	s.handleRestart(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected status %d, got %d", http.StatusNotImplemented, rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["status"] != "error" {
		t.Errorf("expected status field %q, got %q", "error", body["status"])
	}
	if body["message"] != "Restart not implemented" {
		t.Errorf("expected message %q, got %q", "Restart not implemented", body["message"])
	}
}

func TestHandleStop_Returns501(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/actions/stop", nil)
	rec := httptest.NewRecorder()

	s.handleStop(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected status %d, got %d", http.StatusNotImplemented, rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["status"] != "error" {
		t.Errorf("expected status field %q, got %q", "error", body["status"])
	}
	if body["message"] != "Stop not implemented" {
		t.Errorf("expected message %q, got %q", "Stop not implemented", body["message"])
	}
}

func TestHandleRestart_GET_Returns405(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/actions/restart", nil)
	rec := httptest.NewRecorder()

	s.handleRestart(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestHandleStop_GET_Returns405(t *testing.T) {
	s := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/actions/stop", nil)
	rec := httptest.NewRecorder()

	s.handleStop(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}
