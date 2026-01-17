package grpc

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/service/runner"
)

// newTestLogger creates a test logger
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// mockRunnerService implements RunnerServiceInterface for testing
type mockRunnerService struct {
	runners            map[string]RunnerInfo
	revokedCerts       map[string]bool
	err                error
	revocationCheckErr error // Separate error for IsCertificateRevoked
}

func newMockRunnerService() *mockRunnerService {
	return &mockRunnerService{
		runners:      make(map[string]RunnerInfo),
		revokedCerts: make(map[string]bool),
	}
}

func (m *mockRunnerService) GetByNodeID(ctx context.Context, nodeID string) (RunnerInfo, error) {
	if m.err != nil {
		return RunnerInfo{}, m.err
	}
	if runner, ok := m.runners[nodeID]; ok {
		return runner, nil
	}
	return RunnerInfo{}, context.DeadlineExceeded
}

func (m *mockRunnerService) UpdateLastSeen(ctx context.Context, runnerID int64) error {
	return m.err
}

func (m *mockRunnerService) UpdateAvailableAgents(ctx context.Context, runnerID int64, agents []string) error {
	return m.err
}

func (m *mockRunnerService) IsCertificateRevoked(ctx context.Context, serialNumber string) (bool, error) {
	// Use separate error for revocation check to allow testing different error scenarios
	if m.revocationCheckErr != nil {
		return false, m.revocationCheckErr
	}
	// Check if serial is in revoked set
	if revoked, ok := m.revokedCerts[serialNumber]; ok {
		return revoked, nil
	}
	return false, nil
}

func (m *mockRunnerService) AddRunner(nodeID string, runner RunnerInfo) {
	m.runners[nodeID] = runner
}

func (m *mockRunnerService) SetCertificateRevoked(serialNumber string, revoked bool) {
	if m.revokedCerts == nil {
		m.revokedCerts = make(map[string]bool)
	}
	m.revokedCerts[serialNumber] = revoked
}

func (m *mockRunnerService) SetRevocationCheckError(err error) {
	m.revocationCheckErr = err
}

// mockOrgService implements OrganizationServiceInterface for testing
type mockOrgService struct {
	orgs map[string]OrganizationInfo
	err  error
}

func newMockOrgService() *mockOrgService {
	return &mockOrgService{
		orgs: make(map[string]OrganizationInfo),
	}
}

func (m *mockOrgService) GetBySlug(ctx context.Context, slug string) (OrganizationInfo, error) {
	if m.err != nil {
		return OrganizationInfo{}, m.err
	}
	if org, ok := m.orgs[slug]; ok {
		return org, nil
	}
	return OrganizationInfo{}, context.DeadlineExceeded
}

func (m *mockOrgService) AddOrg(slug string, org OrganizationInfo) {
	m.orgs[slug] = org
}

// NOTE: ConnectionEventHandler interface has been removed in favor of RunnerConnectionManager callbacks

func TestNewServer(t *testing.T) {
	logger := newTestLogger()
	cfg := &config.GRPCConfig{
		Address: ":0",
	}
	connMgr := runner.NewRunnerConnectionManager(logger)

	deps := &ServerDependencies{
		Logger:        logger,
		Config:        cfg,
		RunnerService: newMockRunnerService(),
		OrgService:    newMockOrgService(),
		ConnManager:   connMgr,
	}

	server, err := NewServer(deps)
	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.NotNil(t, server.grpcServer)
	assert.NotNil(t, server.runnerAdapter)
}

func TestNewServer_NilDependencies(t *testing.T) {
	_, err := NewServer(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependencies are required")
}

func TestNewServer_NilConfig(t *testing.T) {
	deps := &ServerDependencies{
		Logger: newTestLogger(),
	}

	_, err := NewServer(deps)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gRPC config is required")
}

func TestNewServer_NilLogger(t *testing.T) {
	cfg := &config.GRPCConfig{Address: ":0"}
	connMgr := runner.NewRunnerConnectionManager(newTestLogger())

	deps := &ServerDependencies{
		Logger:        nil, // nil logger should use default
		Config:        cfg,
		RunnerService: newMockRunnerService(),
		OrgService:    newMockOrgService(),
		ConnManager:   connMgr,
	}

	server, err := NewServer(deps)
	require.NoError(t, err)
	assert.NotNil(t, server)
	assert.NotNil(t, server.logger) // Should have default logger
}

func TestServer_Start_DefaultAddress(t *testing.T) {
	logger := newTestLogger()
	cfg := &config.GRPCConfig{Address: ""} // Empty address triggers default path
	connMgr := runner.NewRunnerConnectionManager(logger)

	server, err := NewServer(&ServerDependencies{
		Logger:        logger,
		Config:        cfg,
		RunnerService: newMockRunnerService(),
		OrgService:    newMockOrgService(),
		ConnManager:   connMgr,
	})
	require.NoError(t, err)

	// The default is :9090, which might be in use. Use port 0 for testing but
	// first test with empty address to cover the default path.
	// Note: We directly test with empty address now to cover the default branch.
	// This will try to bind to :9090 which might fail if port is in use.
	err = server.Start()
	if err != nil {
		// If :9090 is in use, that's OK - we've still covered the code path
		assert.Contains(t, err.Error(), "failed to listen")
	} else {
		assert.NotNil(t, server.listener)
		server.Stop()
	}
}

func TestServer_Start_InvalidAddress(t *testing.T) {
	logger := newTestLogger()
	cfg := &config.GRPCConfig{Address: "invalid-address-format:::"}
	connMgr := runner.NewRunnerConnectionManager(logger)

	server, err := NewServer(&ServerDependencies{
		Logger:        logger,
		Config:        cfg,
		RunnerService: newMockRunnerService(),
		OrgService:    newMockOrgService(),
		ConnManager:   connMgr,
	})
	require.NoError(t, err)

	// Start should fail with invalid address
	err = server.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen")
}

func TestExtractClientIdentity(t *testing.T) {
	tests := []struct {
		name        string
		metadata    map[string]string
		wantErr     bool
		errContains string
		checkFunc   func(t *testing.T, identity *ClientIdentity)
	}{
		{
			name: "valid identity with RFC 2253 DN format",
			metadata: map[string]string{
				MetadataKeyClientCertDN:          "CN=test-node-123,O=AgentsMesh,OU=Runner",
				MetadataKeyOrgSlug:               "test-org",
				MetadataKeyClientCertSerial:      "ABCD1234",
				MetadataKeyClientCertFingerprint: "sha256:xyz",
				MetadataKeyRealIP:                "192.168.1.100",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, identity *ClientIdentity) {
				assert.Equal(t, "test-node-123", identity.NodeID)
				assert.Equal(t, "test-org", identity.OrgSlug)
				assert.Equal(t, "ABCD1234", identity.CertSerialNumber)
				assert.Equal(t, "sha256:xyz", identity.CertFingerprint)
				assert.Equal(t, "192.168.1.100", identity.RealIP)
			},
		},
		{
			name: "valid identity with OpenSSL DN format",
			metadata: map[string]string{
				MetadataKeyClientCertDN: "/CN=test-node-456/O=AgentsMesh/OU=Runner",
				MetadataKeyOrgSlug:      "test-org",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, identity *ClientIdentity) {
				assert.Equal(t, "test-node-456", identity.NodeID)
				assert.Equal(t, "test-org", identity.OrgSlug)
			},
		},
		{
			name: "missing node_id (empty DN)",
			metadata: map[string]string{
				MetadataKeyOrgSlug: "test-org",
			},
			wantErr:     true,
			errContains: "missing client certificate CN",
		},
		{
			name: "missing org_slug",
			metadata: map[string]string{
				MetadataKeyClientCertDN: "CN=test-node",
			},
			wantErr:     true,
			errContains: "missing org slug",
		},
		{
			name: "minimal valid identity",
			metadata: map[string]string{
				MetadataKeyClientCertDN: "CN=test-node",
				MetadataKeyOrgSlug:      "test-org",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, identity *ClientIdentity) {
				assert.Equal(t, "test-node", identity.NodeID)
				assert.Equal(t, "test-org", identity.OrgSlug)
				assert.Empty(t, identity.CertSerialNumber)
				assert.Empty(t, identity.CertFingerprint)
				assert.Empty(t, identity.RealIP)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with metadata
			md := metadata.New(tt.metadata)
			ctx := metadata.NewIncomingContext(context.Background(), md)

			identity, err := ExtractClientIdentity(ctx)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, identity)
			if tt.checkFunc != nil {
				tt.checkFunc(t, identity)
			}
		})
	}
}

func TestExtractClientIdentity_NoMetadata(t *testing.T) {
	ctx := context.Background()

	_, err := ExtractClientIdentity(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no metadata in context")
}

func TestServer_GRPCServerAccessor(t *testing.T) {
	logger := newTestLogger()
	cfg := &config.GRPCConfig{Address: ":0"}
	connMgr := runner.NewRunnerConnectionManager(logger)

	server, err := NewServer(&ServerDependencies{
		Logger:        logger,
		Config:        cfg,
		RunnerService: newMockRunnerService(),
		OrgService:    newMockOrgService(),
		ConnManager:   connMgr,
	})
	require.NoError(t, err)

	assert.NotNil(t, server.GRPCServer())
	assert.NotNil(t, server.RunnerAdapter())
}

func TestServer_StartStop(t *testing.T) {
	logger := newTestLogger()
	cfg := &config.GRPCConfig{Address: "127.0.0.1:0"} // Use port 0 for random available port
	connMgr := runner.NewRunnerConnectionManager(logger)

	server, err := NewServer(&ServerDependencies{
		Logger:        logger,
		Config:        cfg,
		RunnerService: newMockRunnerService(),
		OrgService:    newMockOrgService(),
		ConnManager:   connMgr,
	})
	require.NoError(t, err)

	// Start server
	err = server.Start()
	require.NoError(t, err)

	// Verify listener is set
	assert.NotNil(t, server.listener)

	// Stop server
	server.Stop()
}

func TestLoggingUnaryInterceptor(t *testing.T) {
	logger := newTestLogger()
	interceptor := loggingUnaryInterceptor(logger)

	// Create mock handler
	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	// Create mock server info
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// Call interceptor
	resp, err := interceptor(context.Background(), "request", info, handler)

	require.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, "response", resp)
}

func TestLoggingStreamInterceptor(t *testing.T) {
	logger := newTestLogger()
	interceptor := loggingStreamInterceptor(logger)

	// Create mock handler
	handlerCalled := false
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		handlerCalled = true
		return nil
	}

	// Create mock server info
	info := &grpc.StreamServerInfo{
		FullMethod: "/test.Service/StreamMethod",
	}

	// Call interceptor
	err := interceptor("server", nil, info, handler)

	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestExtractCNFromDN_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		dn       string
		expected string
	}{
		{
			name:     "empty DN",
			dn:       "",
			expected: "",
		},
		{
			name:     "CN only",
			dn:       "CN=test",
			expected: "test",
		},
		{
			name:     "CN with spaces",
			dn:       "CN=test node",
			expected: "test node",
		},
		{
			name:     "CN in middle",
			dn:       "O=Org,CN=middle,OU=Unit",
			expected: "middle",
		},
		{
			name:     "OpenSSL format CN in middle",
			dn:       "/O=Org/CN=middle/OU=Unit",
			expected: "middle",
		},
		{
			name:     "no CN",
			dn:       "O=Org,OU=Unit",
			expected: "",
		},
		{
			name:     "OpenSSL format no CN",
			dn:       "/O=Org/OU=Unit",
			expected: "",
		},
		{
			name:     "CN= at end",
			dn:       "O=Org,CN=",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCNFromDN(tt.dn)
			assert.Equal(t, tt.expected, result)
		})
	}
}
