package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/runner"
)

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
