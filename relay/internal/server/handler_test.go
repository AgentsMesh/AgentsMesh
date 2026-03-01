package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/relay/internal/auth"
	"github.com/AgentsMesh/AgentsMesh/relay/internal/channel"
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
func validToken(podKey string) string {
	t, _ := auth.GenerateToken(testSecret, testIssuer, podKey, 1, 2, 3, time.Hour)
	return t
}

func TestItoa(t *testing.T) {
	tests := []struct {
		in  int
		out string
	}{
		{0, "0"}, {1, "1"}, {123, "123"}, {-1, "-1"}, {-123, "-123"}, {1000000, "1000000"},
	}
	for _, tt := range tests {
		if got := itoa(tt.in); got != tt.out {
			t.Errorf("itoa(%d) = %q, want %q", tt.in, got, tt.out)
		}
	}
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
	// Use valid token for runner authentication
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "?token=" + validToken("pod-1")
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
	// Runner first - using token for authentication (podKey = "pod-1")
	runnerToken := validToken("pod-1")
	rc, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/runner?token="+runnerToken, nil)
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
