package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/relay/internal/auth"
	"github.com/anthropics/agentsmesh/relay/internal/channel"
	"github.com/gorilla/websocket"
)

const testSecret = "test-secret-key"
const testIssuer = "test-issuer"

func createTestHandler() *Handler {
	cm := channel.NewChannelManager(30*time.Second, 10, nil)
	tv := auth.NewTokenValidator(testSecret, testIssuer)
	return NewHandler(cm, tv)
}

func TestNewHandler(t *testing.T) {
	h := createTestHandler()
	if h == nil || h.channelManager == nil || h.tokenValidator == nil || h.logger == nil {
		t.Error("handler init failed")
	}
}

func TestHandler_HandleHealth(t *testing.T) {
	h := createTestHandler()
	w := httptest.NewRecorder()
	h.HandleHealth(w, httptest.NewRequest("GET", "/health", nil))
	if w.Code != http.StatusOK || w.Body.String() != `{"status":"ok"}` {
		t.Errorf("health: %d %s", w.Code, w.Body.String())
	}
}

func TestHandler_HandleStats(t *testing.T) {
	h := createTestHandler()
	w := httptest.NewRecorder()
	h.HandleStats(w, httptest.NewRequest("GET", "/stats", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "active_channels") {
		t.Errorf("stats: %d %s", w.Code, w.Body.String())
	}
}

func TestHandler_HandleRunnerWS_MissingToken(t *testing.T) {
	h := createTestHandler()
	// Without token should return 401 Unauthorized
	w := httptest.NewRecorder()
	h.HandleRunnerWS(w, httptest.NewRequest("GET", "/runner/terminal", nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandler_HandleRunnerWS_InvalidToken(t *testing.T) {
	h := createTestHandler()
	w := httptest.NewRecorder()
	h.HandleRunnerWS(w, httptest.NewRequest("GET", "/runner/terminal?token=invalid", nil))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandler_HandleBrowserWS_Errors(t *testing.T) {
	h := createTestHandler()
	tests := []struct {
		name  string
		query string
		code  int
	}{
		{"no_token", "", http.StatusUnauthorized},
		{"invalid_token", "token=invalid", http.StatusUnauthorized},
		{"expired", "token=" + expiredToken(), http.StatusUnauthorized},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		h.HandleBrowserWS(w, httptest.NewRequest("GET", "/browser/terminal?"+tt.query, nil))
		if w.Code != tt.code {
			t.Errorf("%s: expected %d, got %d", tt.name, tt.code, w.Code)
		}
	}
}

func expiredToken() string {
	t, _ := auth.GenerateToken(testSecret, testIssuer, "p1", 1, 2, 3, -time.Hour)
	return t
}

// validToken generates a browser token (userID=2, non-zero) for browser endpoint tests.
func validToken(podKey string) string {
	t, _ := auth.GenerateToken(testSecret, testIssuer, podKey, 1, 2, 3, time.Hour)
	return t
}

// runnerToken generates a runner token (userID=0) for runner endpoint tests.
func runnerToken(podKey string) string {
	t, _ := auth.GenerateToken(testSecret, testIssuer, podKey, 1, 0, 3, time.Hour)
	return t
}

func TestUpgraderSettings(t *testing.T) {
	if upgrader.ReadBufferSize != 1024*64 || upgrader.WriteBufferSize != 1024*64 {
		t.Error("upgrader buffer sizes wrong")
	}
	if !upgrader.CheckOrigin(httptest.NewRequest("GET", "/", nil)) {
		t.Error("CheckOrigin should allow all")
	}
}

func TestHandler_HandleRunnerWS_Success(t *testing.T) {
	h := createTestHandler()
	srv := httptest.NewServer(http.HandlerFunc(h.HandleRunnerWS))
	defer srv.Close()
	// Use runner token (userID=0) for runner authentication
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "?token=" + runnerToken("pod-1")
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer c.Close()
	// Wait for connection to be registered in channel manager
	time.Sleep(50 * time.Millisecond)
	if h.channelManager.Stats().PendingPublishers != 1 {
		t.Error("should have pending publisher")
	}
}

func TestHandler_HandleBrowserWS_Success(t *testing.T) {
	h := createTestHandler()
	srv := httptest.NewServer(http.HandlerFunc(h.HandleBrowserWS))
	defer srv.Close()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "?token=" + validToken("pod-1")
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer c.Close()
	// Wait for connection to be registered in channel manager
	time.Sleep(50 * time.Millisecond)
	if h.channelManager.Stats().PendingSubscribers != 1 {
		t.Error("should have pending subscriber")
	}
}

func TestHandler_HandleRunnerWS_EmptyPodKey(t *testing.T) {
	h := createTestHandler()
	token, _ := auth.GenerateToken(testSecret, testIssuer, "", 1, 0, 3, time.Hour)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/runner/terminal?token="+token, nil)
	h.HandleRunnerWS(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_HandleBrowserWS_EmptyPodKey(t *testing.T) {
	h := createTestHandler()
	token, _ := auth.GenerateToken(testSecret, testIssuer, "", 1, 2, 3, time.Hour)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/browser/terminal?token="+token, nil)
	h.HandleBrowserWS(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandler_HandleBrowserWS_MaxSubscribers(t *testing.T) {
	cm := channel.NewChannelManager(30*time.Second, 1, nil)
	tv := auth.NewTokenValidator(testSecret, testIssuer)
	h := NewHandler(cm, tv)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/runner") {
			h.HandleRunnerWS(w, r)
		} else {
			h.HandleBrowserWS(w, r)
		}
	}))
	defer srv.Close()

	podKey := "pod-max-sub"
	browserToken := validToken(podKey)
	runnerTok := runnerToken(podKey)
	wsBase := "ws" + strings.TrimPrefix(srv.URL, "http")

	// Connect runner (publisher)
	rc, _, err := websocket.DefaultDialer.Dial(wsBase+"/runner?token="+runnerTok, nil)
	if err != nil {
		t.Fatalf("runner dial: %v", err)
	}
	defer rc.Close()

	// Wait for publisher to be registered
	time.Sleep(50 * time.Millisecond)

	// Connect first browser (subscriber) - should succeed
	bc1, _, err := websocket.DefaultDialer.Dial(wsBase+"/browser?token="+browserToken, nil)
	if err != nil {
		t.Fatalf("browser1 dial: %v", err)
	}
	defer bc1.Close()

	// Wait for subscriber to be registered
	time.Sleep(50 * time.Millisecond)

	// Connect second browser - should hit max subscribers
	bc2, _, err := websocket.DefaultDialer.Dial(wsBase+"/browser?token="+browserToken, nil)
	if err != nil {
		t.Fatalf("browser2 dial: %v", err)
	}
	defer bc2.Close()

	// Read close message from bc2
	_ = bc2.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = bc2.ReadMessage()
	if err == nil {
		t.Fatal("expected close error from max subscribers")
	}
	closeErr, ok := err.(*websocket.CloseError)
	if !ok {
		// Connection was closed, that's expected behavior
		return
	}
	if closeErr.Code != websocket.ClosePolicyViolation {
		t.Errorf("expected ClosePolicyViolation (%d), got %d", websocket.ClosePolicyViolation, closeErr.Code)
	}
}

func TestHandler_HandleRunnerWS_UpgradeError(t *testing.T) {
	// Test the WebSocket upgrade error path: valid token + non-WebSocket request
	h := createTestHandler()
	token := runnerToken("pod-1")
	w := httptest.NewRecorder()
	// This is NOT a WebSocket upgrade request — no Upgrade/Connection headers
	r := httptest.NewRequest("GET", "/runner/terminal?token="+token, nil)
	h.HandleRunnerWS(w, r)
	// The upgrader writes 400 Bad Request when upgrade headers are missing
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 from upgrade failure, got %d", w.Code)
	}
}

func TestHandler_HandleBrowserWS_UpgradeError(t *testing.T) {
	// Test the WebSocket upgrade error path: valid token + non-WebSocket request
	h := createTestHandler()
	token := validToken("pod-1")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/browser/terminal?token="+token, nil)
	h.HandleBrowserWS(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 from upgrade failure, got %d", w.Code)
	}
}

func TestHandler_ChannelCreation(t *testing.T) {
	h := createTestHandler()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check path to determine which handler to use
		if strings.HasPrefix(r.URL.Path, "/runner") {
			h.HandleRunnerWS(w, r)
		} else {
			h.HandleBrowserWS(w, r)
		}
	}))
	defer srv.Close()
	// Runner first - using runner token (userID=0) for authentication (podKey = "pod-1")
	rToken := runnerToken("pod-1")
	rc, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/runner?token="+rToken, nil)
	if err != nil {
		t.Fatalf("runner dial failed: %v", err)
	}
	defer rc.Close()
	// Then browser with same podKey
	bc, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/browser?token="+validToken("pod-1"), nil)
	if err != nil {
		t.Fatalf("browser dial failed: %v", err)
	}
	defer bc.Close()
	time.Sleep(50 * time.Millisecond)
	if h.channelManager.Stats().ActiveChannels != 1 {
		t.Error("should have active channel")
	}
}
