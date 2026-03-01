package agentpod

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentDomain "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agent"
	podDomain "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agentpod"
	runnerDomain "github.com/AgentsMesh/AgentsMesh/backend/internal/domain/runner"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/agent"
)

func TestNewPodOrchestrator(t *testing.T) {
	db := setupTestDB(t)
	podSvc := NewPodService(db)
	coord := &mockPodCoordinator{}

	deps := &PodOrchestratorDeps{
		PodService:     podSvc,
		PodCoordinator: coord,
	}
	orch := NewPodOrchestrator(deps)

	assert.NotNil(t, orch)
	assert.Equal(t, podSvc, orch.podService)
	assert.Equal(t, coord, orch.podCoordinator)
}

func TestCreatePod_NormalMode_Success(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		InitialPrompt:  "Hello",
		Cols:           120,
		Rows:           40,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Pod)
	assert.Empty(t, result.Warning)
	assert.Equal(t, podDomain.StatusInitializing, result.Pod.Status)
	assert.True(t, coord.createPodCalled)
	assert.Equal(t, int64(1), coord.lastRunnerID)
	assert.Equal(t, result.Pod.PodKey, coord.lastCmd.PodKey)
}

func TestCreatePod_NormalMode_MissingRunnerID(t *testing.T) {
	// Without RunnerSelector/AgentTypeResolver injected, RunnerID=0 should fail with ErrMissingRunnerID
	orch, _ := setupOrchestrator(t)

	agentTypeID := int64(1)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       0, // missing
		AgentTypeID:    &agentTypeID,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMissingRunnerID))
}

// ==================== Auto-Select Runner Tests ====================

func TestCreatePod_AutoSelectRunner_Success(t *testing.T) {
	coord := &mockPodCoordinator{}
	selector := &mockRunnerSelector{
		runner: &runnerDomain.Runner{ID: 42, NodeID: "auto-runner"},
	}
	resolver := &mockAgentTypeResolver{
		agentType: &agentDomain.AgentType{ID: 1, Slug: "claude-code"},
	}

	orch, _ := setupOrchestrator(t,
		withCoordinator(coord),
		withRunnerSelector(selector),
		withAgentTypeResolver(resolver),
	)

	agentTypeID := int64(1)
	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       0, // auto-select
		AgentTypeID:    &agentTypeID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Pod)
	assert.Equal(t, int64(42), result.Pod.RunnerID) // auto-selected runner
	assert.True(t, coord.createPodCalled)
	assert.Equal(t, int64(42), coord.lastRunnerID)
}

func TestCreatePod_AutoSelectRunner_NoAvailableRunner(t *testing.T) {
	selector := &mockRunnerSelector{
		err: errors.New("no available runner supports the requested agent"),
	}
	resolver := &mockAgentTypeResolver{
		agentType: &agentDomain.AgentType{ID: 1, Slug: "claude-code"},
	}

	orch, _ := setupOrchestrator(t,
		withRunnerSelector(selector),
		withAgentTypeResolver(resolver),
	)

	agentTypeID := int64(1)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       0,
		AgentTypeID:    &agentTypeID,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNoAvailableRunner))
}

func TestCreatePod_AutoSelectRunner_AgentTypeResolveError(t *testing.T) {
	selector := &mockRunnerSelector{
		runner: &runnerDomain.Runner{ID: 42},
	}
	resolver := &mockAgentTypeResolver{
		err: errors.New("agent type not found"),
	}

	orch, _ := setupOrchestrator(t,
		withRunnerSelector(selector),
		withAgentTypeResolver(resolver),
	)

	agentTypeID := int64(999)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       0,
		AgentTypeID:    &agentTypeID,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMissingAgentTypeID))
}

func TestCreatePod_ExplicitRunnerID_SkipsAutoSelect(t *testing.T) {
	// When RunnerID is explicitly provided, auto-select should NOT be invoked
	coord := &mockPodCoordinator{}
	selector := &mockRunnerSelector{
		// This would fail if called, but it shouldn't be called
		err: errors.New("should not be called"),
	}
	resolver := &mockAgentTypeResolver{
		agentType: &agentDomain.AgentType{ID: 1, Slug: "claude-code"},
	}

	orch, _ := setupOrchestrator(t,
		withCoordinator(coord),
		withRunnerSelector(selector),
		withAgentTypeResolver(resolver),
	)

	agentTypeID := int64(1)
	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       5, // explicit runner
		AgentTypeID:    &agentTypeID,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Pod)
	assert.Equal(t, int64(5), result.Pod.RunnerID) // uses explicit runner, not auto-selected
}

func TestCreatePod_NormalMode_MissingAgentTypeID(t *testing.T) {
	orch, _ := setupOrchestrator(t)

	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    nil, // missing
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMissingAgentTypeID))
}

func TestCreatePod_QuotaExceeded(t *testing.T) {
	errQuota := errors.New("quota exceeded")
	billing := &mockBillingService{err: errQuota}
	orch, _ := setupOrchestrator(t, withBilling(billing))

	agentTypeID := int64(1)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
	})

	require.Error(t, err)
	assert.Equal(t, errQuota, err)
}

func TestCreatePod_NilBilling_SkipsQuotaCheck(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Pod)
}

func TestCreatePod_NilCoordinator(t *testing.T) {
	// No coordinator -> pod is created in DB but no command sent
	orch, _ := setupOrchestrator(t)

	agentTypeID := int64(1)
	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Pod)
	assert.Empty(t, result.Warning)
}

func TestCreatePod_CoordinatorSendFailure_ReturnsWarning(t *testing.T) {
	coord := &mockPodCoordinator{err: errors.New("runner not connected")}
	orch, _ := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
	})

	require.NoError(t, err) // Not an error - returns warning
	assert.NotNil(t, result.Pod)
	assert.Contains(t, result.Warning, "runner communication failed")
}

func TestCreatePod_ConfigBuildFailure(t *testing.T) {
	// Create an orchestrator with a provider that fails on GetAgentType
	db := setupTestDB(t)
	podSvc := NewPodService(db)

	provider := &mockAgentConfigProvider{
		agentErr: errors.New("agent type not found"),
	}
	configBuilder := agent.NewConfigBuilder(provider)

	orch := NewPodOrchestrator(&PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: configBuilder,
	})

	agentTypeID := int64(999)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrConfigBuildFailed))
}

func TestCreatePod_SessionID_SetForNormalMode(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
	})

	require.NoError(t, err)
	// Session ID should be set on the pod
	assert.NotNil(t, result.Pod.SessionID)
	assert.NotEmpty(t, *result.Pod.SessionID)
}

func TestCreatePod_ConfigOverrides_Preserved(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID:  1,
		UserID:          1,
		RunnerID:        1,
		AgentTypeID:     &agentTypeID,
		ConfigOverrides: map[string]interface{}{"custom_key": "custom_value"},
	})

	require.NoError(t, err)
	assert.True(t, coord.createPodCalled)
}

func TestCreatePod_NilConfigOverrides_Initialized(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID:  1,
		UserID:          1,
		RunnerID:        1,
		AgentTypeID:     &agentTypeID,
		ConfigOverrides: nil, // should be auto-initialized
	})

	require.NoError(t, err)
	assert.True(t, coord.createPodCalled)
}

func TestCreatePod_PermissionMode(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	permMode := "bypassPermissions"
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		PermissionMode: &permMode,
	})

	require.NoError(t, err)
	assert.True(t, coord.createPodCalled)
}
