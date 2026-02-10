package agentpod

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"gorm.io/gorm"
)

// ==================== Mock Definitions ====================

// mockPodCoordinator implements PodCoordinatorForOrchestrator.
type mockPodCoordinator struct {
	createPodCalled bool
	lastRunnerID    int64
	lastCmd         *runnerv1.CreatePodCommand
	err             error
}

func (m *mockPodCoordinator) CreatePod(_ context.Context, runnerID int64, cmd *runnerv1.CreatePodCommand) error {
	m.createPodCalled = true
	m.lastRunnerID = runnerID
	m.lastCmd = cmd
	return m.err
}

// mockBillingService implements BillingServiceForOrchestrator.
type mockBillingService struct {
	err error
}

func (m *mockBillingService) CheckQuota(_ context.Context, _ int64, _ string, _ int) error {
	return m.err
}

// mockUserServiceForOrch implements UserServiceForOrchestrator.
type mockUserServiceForOrch struct {
	defaultCred    *user.GitCredential
	defaultCredErr error
	decryptedCred  *userService.DecryptedCredential
	decryptedErr   error
}

func (m *mockUserServiceForOrch) GetDefaultGitCredential(_ context.Context, _ int64) (*user.GitCredential, error) {
	return m.defaultCred, m.defaultCredErr
}

func (m *mockUserServiceForOrch) GetDecryptedCredentialToken(_ context.Context, _, _ int64) (*userService.DecryptedCredential, error) {
	return m.decryptedCred, m.decryptedErr
}

// mockRepoService implements RepositoryServiceForOrchestrator.
type mockRepoService struct {
	repo *gitprovider.Repository
	err  error
}

func (m *mockRepoService) GetByID(_ context.Context, _ int64) (*gitprovider.Repository, error) {
	return m.repo, m.err
}

// mockTicketServiceForOrch implements TicketServiceForOrchestrator.
type mockTicketServiceForOrch struct {
	ticket *ticket.Ticket
	err    error
}

func (m *mockTicketServiceForOrch) GetTicket(_ context.Context, _ int64) (*ticket.Ticket, error) {
	return m.ticket, m.err
}

// mockAgentConfigProvider implements agent.AgentConfigProvider for ConfigBuilder.
type mockAgentConfigProvider struct {
	agentType *agentDomain.AgentType
	agentErr  error
	config    agentDomain.ConfigValues
	creds     agentDomain.EncryptedCredentials
	isRunner  bool
	credsErr  error
}

func (m *mockAgentConfigProvider) GetAgentType(_ context.Context, _ int64) (*agentDomain.AgentType, error) {
	return m.agentType, m.agentErr
}

func (m *mockAgentConfigProvider) GetUserEffectiveConfig(_ context.Context, _, _ int64, overrides agentDomain.ConfigValues) agentDomain.ConfigValues {
	if m.config != nil {
		return m.config
	}
	return overrides
}

func (m *mockAgentConfigProvider) GetEffectiveCredentialsForPod(_ context.Context, _, _ int64, _ *int64) (agentDomain.EncryptedCredentials, bool, error) {
	return m.creds, m.isRunner, m.credsErr
}

// ==================== Helper Functions ====================

// setupOrchestratorTestDB extends setupTestDB with additional tables required
// by GORM Preload in GetPod (agent_types, repositories).
// We keep setupTestDB unchanged to avoid breaking existing tests.
func setupOrchestratorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupTestDB(t)

	// agent_types table — needed by Preload("AgentType") when AgentTypeID is set
	db.Exec(`CREATE TABLE IF NOT EXISTS agent_types (
		id INTEGER PRIMARY KEY,
		slug TEXT,
		name TEXT,
		launch_command TEXT,
		description TEXT,
		config_schema TEXT DEFAULT '{}',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	// repositories table — needed by Preload("Repository") when RepositoryID is set
	db.Exec(`CREATE TABLE IF NOT EXISTS repositories (
		id INTEGER PRIMARY KEY,
		organization_id INTEGER,
		provider_type TEXT,
		provider_base_url TEXT,
		clone_url TEXT,
		external_id TEXT,
		name TEXT,
		full_path TEXT,
		default_branch TEXT DEFAULT 'main',
		preparation_script TEXT,
		preparation_timeout INTEGER DEFAULT 300,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	return db
}

func newTestProvider() *mockAgentConfigProvider {
	return &mockAgentConfigProvider{
		agentType: &agentDomain.AgentType{
			ID:            1,
			Slug:          "claude-code",
			Name:          "Claude Code",
			LaunchCommand: "claude",
		},
		config:   agentDomain.ConfigValues{},
		creds:    agentDomain.EncryptedCredentials{},
		isRunner: true,
	}
}

func setupOrchestrator(t *testing.T, opts ...func(*PodOrchestratorDeps)) (*PodOrchestrator, *PodService) {
	t.Helper()
	db := setupOrchestratorTestDB(t)
	podSvc := NewPodService(db)

	provider := newTestProvider()
	configBuilder := agent.NewConfigBuilder(provider)

	deps := &PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: configBuilder,
	}

	for _, opt := range opts {
		opt(deps)
	}

	return NewPodOrchestrator(deps), podSvc
}

func withCoordinator(coord PodCoordinatorForOrchestrator) func(*PodOrchestratorDeps) {
	return func(d *PodOrchestratorDeps) { d.PodCoordinator = coord }
}

func withBilling(b BillingServiceForOrchestrator) func(*PodOrchestratorDeps) {
	return func(d *PodOrchestratorDeps) { d.BillingService = b }
}

func withUserSvc(u UserServiceForOrchestrator) func(*PodOrchestratorDeps) {
	return func(d *PodOrchestratorDeps) { d.UserService = u }
}

func withRepoSvc(r RepositoryServiceForOrchestrator) func(*PodOrchestratorDeps) {
	return func(d *PodOrchestratorDeps) { d.RepoService = r }
}

func withTicketSvc(ts TicketServiceForOrchestrator) func(*PodOrchestratorDeps) {
	return func(d *PodOrchestratorDeps) { d.TicketService = ts }
}

// ==================== Tests ====================

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

// ==================== Resume Mode Tests ====================

func TestCreatePod_ResumeMode_Success(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc := setupOrchestrator(t, withCoordinator(coord))

	// Create source pod (terminated)
	agentTypeID := int64(1)
	sessionID := "existing-session-123"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		CreatedByID:    1,
		SessionID:      sessionID,
	})
	require.NoError(t, err)

	// Terminate the source pod (use raw SQL to avoid GREATEST() SQLite incompatibility)
	podSvc.db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Pod)
	// Should inherit runner_id and agent_type_id from source pod
	assert.Equal(t, int64(1), result.Pod.RunnerID)
	assert.Equal(t, &agentTypeID, result.Pod.AgentTypeID)
}

func TestCreatePod_ResumeMode_SourcePodNotFound(t *testing.T) {
	orch, _ := setupOrchestrator(t)

	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   "non-existent-pod",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourcePodNotFound))
}

func TestCreatePod_ResumeMode_AccessDenied(t *testing.T) {
	orch, podSvc := setupOrchestrator(t)

	agentTypeID := int64(1)
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 999, // Different org
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		CreatedByID:    1,
	})
	require.NoError(t, err)
	podSvc.db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	_, err = orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1, // Different org from source pod
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourcePodAccessDenied))
}

func TestCreatePod_ResumeMode_NotTerminated(t *testing.T) {
	orch, podSvc := setupOrchestrator(t)

	agentTypeID := int64(1)
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		CreatedByID:    1,
	})
	require.NoError(t, err)
	// Pod is still "initializing" (default status)

	_, err = orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourcePodNotTerminated))
}

func TestCreatePod_ResumeMode_AlreadyResumed(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	podSvc.db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	// First resume should succeed
	_, err = orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})
	require.NoError(t, err)

	// Second resume from same source should fail
	_, err = orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourcePodAlreadyResumed))
}

func TestCreatePod_ResumeMode_RunnerMismatch(t *testing.T) {
	orch, podSvc := setupOrchestrator(t)

	// Insert a second runner
	podSvc.db.Exec("INSERT INTO runners (id, node_id, status, current_pods) VALUES (2, 'runner-002', 'online', 0)")

	agentTypeID := int64(1)
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1, // Source on runner 1
		AgentTypeID:    &agentTypeID,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	podSvc.db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	_, err = orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       2, // Different runner
		SourcePodKey:   sourcePod.PodKey,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrResumeRunnerMismatch))
}

func TestCreatePod_ResumeMode_InheritRunnerID(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	podSvc.db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	// RunnerID=0 -> should inherit from source pod
	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       0,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Pod.RunnerID)
}

func TestCreatePod_ResumeMode_InheritConfig(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	repoID := int64(10)
	ticketID := int64(20)
	branch := "feature-branch"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		RepositoryID:   &repoID,
		TicketID:       &ticketID,
		BranchName:     &branch,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	podSvc.db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	assert.Equal(t, &agentTypeID, result.Pod.AgentTypeID)
	assert.Equal(t, &repoID, result.Pod.RepositoryID)
	assert.Equal(t, &ticketID, result.Pod.TicketID)
	assert.Equal(t, &branch, result.Pod.BranchName)
}

func TestCreatePod_ResumeMode_SessionReused(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		CreatedByID:    1,
		SessionID:      "my-session-id",
	})
	require.NoError(t, err)
	podSvc.db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Pod.SessionID)
	assert.Equal(t, "my-session-id", *result.Pod.SessionID)
}

func TestCreatePod_ResumeMode_NoSessionID_GeneratesNew(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		CreatedByID:    1,
		SessionID:      "", // No session ID
	})
	require.NoError(t, err)
	podSvc.db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Pod.SessionID)
	assert.NotEmpty(t, *result.Pod.SessionID)
}

func TestCreatePod_ResumeMode_DisableResumeAgentSession(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	podSvc.db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	resumeOff := false
	_, err = orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID:     1,
		UserID:             1,
		SourcePodKey:       sourcePod.PodKey,
		ResumeAgentSession: &resumeOff,
	})

	require.NoError(t, err)
	// When ResumeAgentSession is false, resume_enabled/resume_session should NOT be set
}

func TestCreatePod_ResumeMode_CompletedPod(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	podSvc.db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusCompleted, sourcePod.PodKey)

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Pod)
}

func TestCreatePod_ResumeMode_OrphanedPod(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	podSvc.db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusOrphaned, sourcePod.PodKey)

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Pod)
}

func TestCreatePod_ResumeMode_SandboxPath(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)

	// Set sandbox path on source pod
	sandboxPath := "/home/user/sandbox/pod-123"
	podSvc.db.Model(&podDomain.Pod{}).Where("pod_key = ?", sourcePod.PodKey).Updates(map[string]interface{}{
		"sandbox_path": sandboxPath,
		"status":       podDomain.StatusTerminated,
	})

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Pod)
	assert.True(t, coord.createPodCalled)
	// SandboxConfig.LocalPath should be set when sandbox_path exists
	if coord.lastCmd.SandboxConfig != nil {
		assert.Equal(t, sandboxPath, coord.lastCmd.SandboxConfig.LocalPath)
	}
}

// ==================== buildPodCommand Tests ====================

func TestBuildPodCommand_WithRepository(t *testing.T) {
	prepScript := "npm install"
	prepTimeout := 600
	repo := &gitprovider.Repository{
		CloneURL:           "https://github.com/org/repo.git",
		DefaultBranch:      "develop",
		PreparationScript:  &prepScript,
		PreparationTimeout: &prepTimeout,
	}
	repoSvc := &mockRepoService{repo: repo}
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord), withRepoSvc(repoSvc))

	agentTypeID := int64(1)
	repoID := int64(10)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		RepositoryID:   &repoID,
	})

	require.NoError(t, err)
	require.NotNil(t, coord.lastCmd)
	require.NotNil(t, coord.lastCmd.SandboxConfig)
	assert.Equal(t, "https://github.com/org/repo.git", coord.lastCmd.SandboxConfig.RepositoryUrl)
	assert.Equal(t, "develop", coord.lastCmd.SandboxConfig.SourceBranch)
	assert.Equal(t, "npm install", coord.lastCmd.SandboxConfig.PreparationScript)
	assert.Equal(t, int32(600), coord.lastCmd.SandboxConfig.PreparationTimeout)
}

func TestBuildPodCommand_WithRepositoryURL(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	repoURL := "https://github.com/org/repo.git"
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		RepositoryURL:  &repoURL,
	})

	require.NoError(t, err)
	require.NotNil(t, coord.lastCmd.SandboxConfig)
	assert.Equal(t, repoURL, coord.lastCmd.SandboxConfig.RepositoryUrl)
}

func TestBuildPodCommand_BranchOverride(t *testing.T) {
	repo := &gitprovider.Repository{
		CloneURL:      "https://github.com/org/repo.git",
		DefaultBranch: "develop",
	}
	repoSvc := &mockRepoService{repo: repo}
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord), withRepoSvc(repoSvc))

	agentTypeID := int64(1)
	repoID := int64(10)
	branch := "feature/my-branch"
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		RepositoryID:   &repoID,
		BranchName:     &branch,
	})

	require.NoError(t, err)
	assert.Equal(t, "feature/my-branch", coord.lastCmd.SandboxConfig.SourceBranch)
}

func TestBuildPodCommand_WithTicket(t *testing.T) {
	ticketSvc := &mockTicketServiceForOrch{
		ticket: &ticket.Ticket{
			ID:         1,
			Identifier: "AM-42",
		},
	}
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord), withTicketSvc(ticketSvc))

	agentTypeID := int64(1)
	ticketID := int64(1)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		TicketID:       &ticketID,
	})

	require.NoError(t, err)
	assert.True(t, coord.createPodCalled)
}

func TestBuildPodCommand_WithTicketIdentifier(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord))

	agentTypeID := int64(1)
	ticketIdentifier := "AM-99"
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID:   1,
		UserID:           1,
		RunnerID:         1,
		AgentTypeID:      &agentTypeID,
		TicketIdentifier: &ticketIdentifier,
	})

	require.NoError(t, err)
	assert.True(t, coord.createPodCalled)
}

func TestBuildPodCommand_WithOAuthCredential(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred: &user.GitCredential{
			ID:             1,
			CredentialType: "oauth",
		},
		decryptedCred: &userService.DecryptedCredential{
			Type:  "oauth",
			Token: "github-token-123",
		},
	}
	coord := &mockPodCoordinator{}
	repoURL := "https://github.com/org/repo.git"
	orch, _ := setupOrchestrator(t, withCoordinator(coord), withUserSvc(userSvc))

	agentTypeID := int64(1)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		RepositoryURL:  &repoURL,
	})

	require.NoError(t, err)
	require.NotNil(t, coord.lastCmd.SandboxConfig)
	assert.Equal(t, "oauth", coord.lastCmd.SandboxConfig.CredentialType)
	assert.Equal(t, "github-token-123", coord.lastCmd.SandboxConfig.GitToken)
}

func TestBuildPodCommand_WithSSHCredential(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred: &user.GitCredential{
			ID:             1,
			CredentialType: "ssh_key",
		},
		decryptedCred: &userService.DecryptedCredential{
			Type:          "ssh_key",
			SSHPrivateKey: "-----BEGIN RSA PRIVATE KEY-----\nfake\n-----END RSA PRIVATE KEY-----",
		},
	}
	coord := &mockPodCoordinator{}
	repoURL := "git@github.com:org/repo.git"
	orch, _ := setupOrchestrator(t, withCoordinator(coord), withUserSvc(userSvc))

	agentTypeID := int64(1)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		RepositoryURL:  &repoURL,
	})

	require.NoError(t, err)
	require.NotNil(t, coord.lastCmd.SandboxConfig)
	assert.Equal(t, "ssh_key", coord.lastCmd.SandboxConfig.CredentialType)
	assert.Contains(t, coord.lastCmd.SandboxConfig.SshPrivateKey, "BEGIN RSA PRIVATE KEY")
}

func TestBuildPodCommand_RunnerLocalCredential_NoCredsSent(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred: &user.GitCredential{
			ID:             1,
			CredentialType: "runner_local",
		},
	}
	coord := &mockPodCoordinator{}
	repoURL := "https://github.com/org/repo.git"
	orch, _ := setupOrchestrator(t, withCoordinator(coord), withUserSvc(userSvc))

	agentTypeID := int64(1)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		RepositoryURL:  &repoURL,
	})

	require.NoError(t, err)
	require.NotNil(t, coord.lastCmd.SandboxConfig)
	assert.Empty(t, coord.lastCmd.SandboxConfig.CredentialType)
	assert.Empty(t, coord.lastCmd.SandboxConfig.GitToken)
}

// ==================== getUserGitCredential Tests ====================

func TestGetUserGitCredential_NilUserService(t *testing.T) {
	db := setupTestDB(t)
	podSvc := NewPodService(db)
	provider := newTestProvider()
	orch := NewPodOrchestrator(&PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: agent.NewConfigBuilder(provider),
	})

	result := orch.getUserGitCredential(context.Background(), 1)
	assert.Nil(t, result)
}

func TestGetUserGitCredential_NoDefaultCredential(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred:    nil,
		defaultCredErr: errors.New("not found"),
	}
	db := setupTestDB(t)
	podSvc := NewPodService(db)
	provider := newTestProvider()
	orch := NewPodOrchestrator(&PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: agent.NewConfigBuilder(provider),
		UserService:   userSvc,
	})

	result := orch.getUserGitCredential(context.Background(), 1)
	assert.Nil(t, result)
}

func TestGetUserGitCredential_RunnerLocal(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred: &user.GitCredential{
			ID:             1,
			CredentialType: "runner_local",
		},
	}
	db := setupTestDB(t)
	podSvc := NewPodService(db)
	provider := newTestProvider()
	orch := NewPodOrchestrator(&PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: agent.NewConfigBuilder(provider),
		UserService:   userSvc,
	})

	result := orch.getUserGitCredential(context.Background(), 1)
	assert.Nil(t, result) // runner_local returns nil
}

func TestGetUserGitCredential_DecryptError(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred: &user.GitCredential{
			ID:             1,
			CredentialType: "oauth",
		},
		decryptedErr: errors.New("decrypt failed"),
	}
	db := setupTestDB(t)
	podSvc := NewPodService(db)
	provider := newTestProvider()
	orch := NewPodOrchestrator(&PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: agent.NewConfigBuilder(provider),
		UserService:   userSvc,
	})

	result := orch.getUserGitCredential(context.Background(), 1)
	assert.Nil(t, result) // Error during decrypt -> returns nil
}

func TestGetUserGitCredential_Success_PAT(t *testing.T) {
	userSvc := &mockUserServiceForOrch{
		defaultCred: &user.GitCredential{
			ID:             1,
			CredentialType: "pat",
		},
		decryptedCred: &userService.DecryptedCredential{
			Type:  "pat",
			Token: "ghp_xxxxx",
		},
	}
	db := setupTestDB(t)
	podSvc := NewPodService(db)
	provider := newTestProvider()
	orch := NewPodOrchestrator(&PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: agent.NewConfigBuilder(provider),
		UserService:   userSvc,
	})

	result := orch.getUserGitCredential(context.Background(), 1)
	require.NotNil(t, result)
	assert.Equal(t, "pat", result.Type)
	assert.Equal(t, "ghp_xxxxx", result.Token)
}

func TestBuildPodCommand_RepoServiceError_IgnoresRepo(t *testing.T) {
	repoSvc := &mockRepoService{err: errors.New("repo not found")}
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord), withRepoSvc(repoSvc))

	agentTypeID := int64(1)
	repoID := int64(999)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		RepositoryID:   &repoID,
	})

	require.NoError(t, err) // Repo error is not fatal
	assert.Nil(t, coord.lastCmd.SandboxConfig)
}

func TestBuildPodCommand_TicketServiceError_IgnoresTicket(t *testing.T) {
	ticketSvc := &mockTicketServiceForOrch{err: errors.New("ticket not found")}
	coord := &mockPodCoordinator{}
	orch, _ := setupOrchestrator(t, withCoordinator(coord), withTicketSvc(ticketSvc))

	agentTypeID := int64(1)
	ticketID := int64(999)
	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentTypeID:    &agentTypeID,
		TicketID:       &ticketID,
	})

	require.NoError(t, err) // Ticket error is not fatal
}
