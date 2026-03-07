package runner

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/pki"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MockRunnerStream implements RunnerStream for testing with full type safety.
// Shared across all test files in the runner package.
type MockRunnerStream struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	mu         sync.Mutex
	SendCh     chan *runnerv1.ServerMessage
	RecvCh     chan *runnerv1.RunnerMessage
}

// Compile-time check: MockRunnerStream implements RunnerStream
var _ RunnerStream = (*MockRunnerStream)(nil)

// newMockRunnerStream creates a new MockRunnerStream without *testing.T dependency.
func newMockRunnerStream() *MockRunnerStream {
	ctx, cancel := context.WithCancel(context.Background())
	return &MockRunnerStream{
		ctx:        ctx,
		cancelFunc: cancel,
		SendCh:     make(chan *runnerv1.ServerMessage, 100),
		RecvCh:     make(chan *runnerv1.RunnerMessage, 100),
	}
}

// newMockRunnerStreamWithTesting creates a new MockRunnerStream with automatic cleanup.
func newMockRunnerStreamWithTesting(t *testing.T) *MockRunnerStream {
	stream := newMockRunnerStream()
	t.Cleanup(stream.Close)
	return stream
}

func (m *MockRunnerStream) Send(msg *runnerv1.ServerMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	select {
	case m.SendCh <- msg:
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
}

func (m *MockRunnerStream) Recv() (*runnerv1.RunnerMessage, error) {
	select {
	case msg := <-m.RecvCh:
		return msg, nil
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

func (m *MockRunnerStream) Context() context.Context {
	return m.ctx
}

func (m *MockRunnerStream) Close() {
	m.cancelFunc()
}

// newTestLogger creates a test logger that only logs errors
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// MockCommandSender implements RunnerCommandSender for testing.
// Shared across all test files in the runner package.
// Thread-safe for use with async goroutines.
type MockCommandSender struct {
	mu                        sync.Mutex
	CreatePodCalls            int
	TerminatePodCalls         int
	TerminalInputCalls        int
	TerminalResizeCalls       int
	TerminalRedrawCalls       int
	SendPromptCalls           int
	SubscribeTerminalCalls    int
	UnsubscribeTerminalCalls  int
}

func (m *MockCommandSender) SendCreatePod(ctx context.Context, runnerID int64, cmd *runnerv1.CreatePodCommand) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CreatePodCalls++
	return nil
}

func (m *MockCommandSender) SendTerminatePod(ctx context.Context, runnerID int64, podKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TerminatePodCalls++
	return nil
}

func (m *MockCommandSender) SendTerminalInput(ctx context.Context, runnerID int64, podKey string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TerminalInputCalls++
	return nil
}

func (m *MockCommandSender) SendTerminalResize(ctx context.Context, runnerID int64, podKey string, cols, rows int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TerminalResizeCalls++
	return nil
}

func (m *MockCommandSender) SendTerminalRedraw(ctx context.Context, runnerID int64, podKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TerminalRedrawCalls++
	return nil
}

func (m *MockCommandSender) SendPrompt(ctx context.Context, runnerID int64, podKey, prompt string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SendPromptCalls++
	return nil
}

func (m *MockCommandSender) SendSubscribeTerminal(ctx context.Context, runnerID int64, podKey, relayURL, runnerToken string, includeSnapshot bool, snapshotHistory int32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SubscribeTerminalCalls++
	return nil
}

func (m *MockCommandSender) SendUnsubscribeTerminal(ctx context.Context, runnerID int64, podKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UnsubscribeTerminalCalls++
	return nil
}

func (m *MockCommandSender) SendCreateAutopilot(runnerID int64, cmd *runnerv1.CreateAutopilotCommand) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil
}

func (m *MockCommandSender) SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return nil
}

// GetTerminalRedrawCalls returns the number of SendTerminalRedraw calls (thread-safe).
func (m *MockCommandSender) GetTerminalRedrawCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.TerminalRedrawCalls
}

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Create runners table (auth_token_hash removed - using mTLS certificates)
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS runners (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			node_id TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'offline',
			last_heartbeat DATETIME,
			current_pods INTEGER NOT NULL DEFAULT 0,
			max_concurrent_pods INTEGER NOT NULL DEFAULT 5,
			runner_version TEXT,
			is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
			host_info TEXT,
			available_agents TEXT DEFAULT '[]',
			agent_versions TEXT DEFAULT '[]',
			visibility TEXT NOT NULL DEFAULT 'organization',
			registered_by_user_id INTEGER,
			cert_serial_number TEXT,
			cert_fingerprint TEXT,
			cert_expires_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create runners table: %v", err)
	}

	// Create indexes
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_runners_organization_id ON runners(organization_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_runners_status ON runners(status)`)

	// Create organizations table for gRPC registration tests
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create organizations table: %v", err)
	}

	// Create runner_pending_auths table for Tailscale-style registration
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS runner_pending_auths (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			auth_key TEXT NOT NULL UNIQUE,
			machine_key TEXT NOT NULL,
			node_id TEXT,
			labels TEXT,
			authorized BOOLEAN NOT NULL DEFAULT FALSE,
			organization_id INTEGER,
			runner_id INTEGER,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create runner_pending_auths table: %v", err)
	}

	// Create runner_grpc_registration_tokens table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS runner_grpc_registration_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			token_hash TEXT NOT NULL UNIQUE,
			organization_id INTEGER NOT NULL,
			name TEXT,
			labels TEXT,
			single_use BOOLEAN NOT NULL DEFAULT TRUE,
			max_uses INTEGER NOT NULL DEFAULT 1,
			used_count INTEGER NOT NULL DEFAULT 0,
			expires_at DATETIME NOT NULL,
			created_by INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create runner_grpc_registration_tokens table: %v", err)
	}

	// Create runner_certificates table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS runner_certificates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			runner_id INTEGER NOT NULL,
			serial_number TEXT NOT NULL UNIQUE,
			fingerprint TEXT NOT NULL,
			issued_at DATETIME NOT NULL,
			expires_at DATETIME NOT NULL,
			revoked_at DATETIME,
			revocation_reason TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create runner_certificates table: %v", err)
	}

	// Create runner_reactivation_tokens table
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS runner_reactivation_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			runner_id INTEGER NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			used_at DATETIME,
			created_by INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create runner_reactivation_tokens table: %v", err)
	}

	// Create loops table (referenced by DeleteRunner for application-level RESTRICT check)
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS loops (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			repository_id INTEGER,
			runner_id INTEGER,
			custom_agent_type_id INTEGER
		)
	`).Error
	if err != nil {
		t.Fatalf("failed to create loops table: %v", err)
	}

	return db
}

// testOrg represents a test organization
type testOrg struct {
	ID   int64
	Name string
	Slug string
}

// createTestOrg creates a test organization in the database
func createTestOrg(t *testing.T, db *gorm.DB, slug string) *testOrg {
	result := db.Exec(`
		INSERT INTO organizations (name, slug) VALUES (?, ?)
	`, "Test Org "+slug, slug)
	if result.Error != nil {
		t.Fatalf("failed to create test org: %v", result.Error)
	}

	var org testOrg
	err := db.Raw(`SELECT id, name, slug FROM organizations WHERE slug = ?`, slug).Scan(&org).Error
	if err != nil {
		t.Fatalf("failed to get test org: %v", err)
	}
	return &org
}

// createTestCA creates a self-signed CA certificate for testing
func createTestCA(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	// Generate CA key
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}

	// Create CA certificate template
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("failed to generate serial: %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "Test CA",
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// Self-sign the CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA cert: %v", err)
	}

	// Encode to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	keyDER, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		t.Fatalf("failed to marshal CA key: %v", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM
}

// setupTestPKI creates a test PKI service with temporary CA files
func setupTestPKI(t *testing.T) (*pki.Service, string) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "pki-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create test CA
	certPEM, keyPEM := createTestCA(t)

	// Write CA files
	certFile := filepath.Join(tmpDir, "ca.crt")
	keyFile := filepath.Join(tmpDir, "ca.key")
	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	// Create service
	cfg := &pki.Config{
		CACertFile:   certFile,
		CAKeyFile:    keyFile,
		ValidityDays: 365,
	}

	service, err := pki.NewService(cfg)
	if err != nil {
		t.Fatalf("failed to create PKI service: %v", err)
	}

	return service, tmpDir
}
