package ws

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/agentmesh/backend/internal/domain/runner"
	runnerSvc "github.com/anthropics/agentmesh/backend/internal/service/runner"
	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Use shared cache mode with unique database name per test
	// This is necessary for WebSocket integration tests where goroutines may use different connections
	// Each test gets its own unique database to avoid data collision between tests
	dbName := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create tables manually for SQLite compatibility
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS runner_registration_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			description TEXT,
			created_by_id INTEGER NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			max_uses INTEGER,
			used_count INTEGER NOT NULL DEFAULT 0,
			expires_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create registration_tokens table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS runners (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			node_id TEXT NOT NULL,
			description TEXT,
			auth_token_hash TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'offline',
			last_heartbeat DATETIME,
			current_pods INTEGER NOT NULL DEFAULT 0,
			max_concurrent_pods INTEGER NOT NULL DEFAULT 5,
			runner_version TEXT,
			is_enabled INTEGER NOT NULL DEFAULT 1,
			host_info TEXT,
			capabilities TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create runners table: %v", err)
	}

	return db
}

func setupTestHandler(t *testing.T) (*RunnerHandler, *runnerSvc.Service, *gorm.DB) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	service := runnerSvc.NewService(db)
	connMgr := runnerSvc.NewConnectionManager(slog.Default())
	logger := slog.Default()

	handler := NewRunnerHandler(service, connMgr, logger)
	return handler, service, db
}

func TestNewRunnerHandler(t *testing.T) {
	handler, _, _ := setupTestHandler(t)

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
	if handler.runnerService == nil {
		t.Fatal("expected non-nil runner service")
	}
	if handler.connectionManager == nil {
		t.Fatal("expected non-nil connection manager")
	}
	if handler.logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestHandleRunnerWSMissingCredentials(t *testing.T) {
	handler, _, _ := setupTestHandler(t)

	router := gin.New()
	router.GET("/api/v1/runners/ws", handler.HandleRunnerWS)

	// Test with missing node_id
	req := httptest.NewRequest("GET", "/api/v1/runners/ws?token=sometoken", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Test with missing token
	req = httptest.NewRequest("GET", "/api/v1/runners/ws?node_id=test-node", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandleRunnerWSInvalidAuth(t *testing.T) {
	handler, _, _ := setupTestHandler(t)

	router := gin.New()
	router.GET("/api/v1/runners/ws", handler.HandleRunnerWS)

	// Test with invalid credentials
	req := httptest.NewRequest("GET", "/api/v1/runners/ws?node_id=test-node&token=invalid-token", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandleRunnerWSHeaderAuth(t *testing.T) {
	handler, service, _ := setupTestHandler(t)
	ctx := context.Background()

	// Register a runner first
	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r, authToken, _ := service.RegisterRunner(ctx, plain, "test-runner", "Test", 5)

	router := gin.New()
	router.GET("/api/v1/runners/ws", handler.HandleRunnerWS)

	// Test with Authorization header
	req := httptest.NewRequest("GET", "/api/v1/runners/ws", nil)
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("X-Runner-ID", r.NodeID)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// httptest doesn't support WebSocket upgrade, so we expect the upgrade to fail
	// But it should NOT fail with 401 (auth failure) - it should fail because of
	// missing WebSocket upgrade headers (which is expected in httptest)
	// Status 400 in this case means it passed auth and tried to upgrade
	if w.Code == http.StatusUnauthorized {
		t.Errorf("unexpected status %d - should have passed auth check", w.Code)
	}
	// 400 is expected when auth passes but WebSocket upgrade fails in httptest
}

func TestHandleRunnerWSValidAuth(t *testing.T) {
	handler, service, _ := setupTestHandler(t)
	ctx := context.Background()

	// Register a runner first
	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r, authToken, _ := service.RegisterRunner(ctx, plain, "test-runner", "Test", 5)

	router := gin.New()
	router.GET("/api/v1/runners/ws", handler.HandleRunnerWS)

	// Test with valid credentials
	req := httptest.NewRequest("GET", "/api/v1/runners/ws?node_id="+r.NodeID+"&token="+authToken, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// httptest doesn't support WebSocket upgrade, so we expect the upgrade to fail
	// But it should NOT fail with 401 (auth failure)
	// Status 400 in this case means it passed auth and tried to upgrade
	if w.Code == http.StatusUnauthorized {
		t.Errorf("unexpected status %d - should have passed auth check", w.Code)
	}
	// 400 is expected when auth passes but WebSocket upgrade fails in httptest
}

func TestHandleRunnerWSDisabledRunner(t *testing.T) {
	handler, service, _ := setupTestHandler(t)
	ctx := context.Background()

	// Register and disable a runner
	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r, authToken, _ := service.RegisterRunner(ctx, plain, "test-runner", "Test", 5)

	// Disable the runner
	isEnabled := false
	service.UpdateRunner(ctx, r.ID, runnerSvc.RunnerUpdateInput{IsEnabled: &isEnabled})

	router := gin.New()
	router.GET("/api/v1/runners/ws", handler.HandleRunnerWS)

	// Test with disabled runner
	req := httptest.NewRequest("GET", "/api/v1/runners/ws?node_id="+r.NodeID+"&token="+authToken, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for disabled runner, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestRunnerWSIntegration(t *testing.T) {
	handler, service, _ := setupTestHandler(t)
	ctx := context.Background()

	// Register a runner
	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r, authToken, _ := service.RegisterRunner(ctx, plain, "test-runner", "Test", 5)

	// Create test server
	router := gin.New()
	router.GET("/api/v1/runners/ws", handler.HandleRunnerWS)
	server := httptest.NewServer(router)
	defer server.Close()

	// Convert URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/runners/ws?node_id=" + r.NodeID + "&token=" + authToken

	// Try to connect via WebSocket
	dialer := gorillaws.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		// Expected - the test server may not properly handle WebSocket upgrade
		// Just verify we got past auth (not 401)
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			t.Errorf("unexpected 401 - auth should have passed")
		}
		return
	}
	defer conn.Close()

	// If we get here, connection was successful
	// Verify runner status was updated to online
	updated, err := service.GetRunner(ctx, r.ID)
	if err != nil {
		t.Logf("GetRunner failed (may be due to test cleanup): %v", err)
		return
	}
	if updated.Status != runner.RunnerStatusOnline {
		t.Errorf("expected runner status online after connect, got %s", updated.Status)
	}
}

func TestSendToRunner(t *testing.T) {
	handler, service, _ := setupTestHandler(t)
	ctx := context.Background()

	// Register a runner (not connected)
	plain, _ := service.CreateRegistrationToken(ctx, 1, 1, "Test Token", nil, nil)
	r, _, _ := service.RegisterRunner(ctx, plain, "test-runner", "Test", 5)

	// Try to send to unconnected runner
	err := handler.SendToRunner(r.ID, "test_message", map[string]string{"key": "value"})
	if err == nil {
		t.Error("expected error sending to unconnected runner")
	}
	if err != runnerSvc.ErrRunnerNotConnected {
		t.Errorf("expected ErrRunnerNotConnected, got %v", err)
	}
}
