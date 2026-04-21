package agentpod

import (
	"context"
	"testing"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
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

func (m *mockRepoService) FindByOrgSlug(_ context.Context, _ int64, _ string) (*gitprovider.Repository, error) {
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

func (m *mockTicketServiceForOrch) GetTicketBySlug(_ context.Context, _ int64, _ string) (*ticket.Ticket, error) {
	return m.ticket, m.err
}

// mockAgentConfigProvider implements agent.AgentConfigProvider for ConfigBuilder.
type mockAgentConfigProvider struct {
	agentDef *agentDomain.Agent
	agentErr error
	config   agentDomain.ConfigValues
	creds    agentDomain.EncryptedCredentials
	isRunner bool
	credsErr error
}

func (m *mockAgentConfigProvider) GetAgent(_ context.Context, _ string) (*agentDomain.Agent, error) {
	return m.agentDef, m.agentErr
}

func (m *mockAgentConfigProvider) GetEffectiveCredentialsForPod(_ context.Context, _ int64, _ string, _ *int64) (agentDomain.EncryptedCredentials, bool, error) {
	return m.creds, m.isRunner, m.credsErr
}

func (m *mockAgentConfigProvider) ResolveCredentialsByName(_ context.Context, _ int64, _, profileName string) (agentDomain.EncryptedCredentials, bool, error) {
	if profileName == "runner_host" {
		return nil, true, nil
	}
	return m.creds, m.isRunner, m.credsErr
}

// mockRunnerSelector implements RunnerSelectorForOrchestrator for testing.
type mockRunnerSelector struct {
	runner *runnerDomain.Runner
	err    error
}

func (m *mockRunnerSelector) SelectRunnerWithAffinity(_ context.Context, _ int64, _ int64, _ string, _ *runnerDomain.AffinityHints, _ map[int64]int) (*runnerDomain.Runner, error) {
	return m.runner, m.err
}

// mockAgentResolver implements AgentResolverForOrchestrator for testing.
type mockAgentResolver struct {
	agentDef *agentDomain.Agent
	err      error
}

func (m *mockAgentResolver) GetAgent(_ context.Context, _ string) (*agentDomain.Agent, error) {
	return m.agentDef, m.err
}

// ==================== Helper Functions ====================

// setupOrchestratorTestDB extends setupTestDB with additional tables required
// by GORM Preload in GetPod (agents, repositories).
// We keep setupTestDB unchanged to avoid breaking existing tests.
func setupOrchestratorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupTestDB(t)

	// agents table — needed by Preload("Agent") when AgentSlug is set
	db.Exec(`CREATE TABLE IF NOT EXISTS agents (
		id INTEGER PRIMARY KEY,
		slug TEXT,
		name TEXT,
		launch_command TEXT,
		description TEXT,
		config_schema TEXT DEFAULT '{}',
		agentfile_source TEXT,
		supported_modes TEXT NOT NULL DEFAULT 'pty',
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
		http_clone_url TEXT,
		ssh_clone_url TEXT,
		external_id TEXT,
		name TEXT,
		slug TEXT,
		default_branch TEXT DEFAULT 'main',
		preparation_script TEXT,
		preparation_timeout INTEGER DEFAULT 300,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)

	return db
}

func newTestProvider() *mockAgentConfigProvider {
	agentfile := `
AGENT claude
EXECUTABLE claude
MCP ON
arg "--session-id" config.session_id when config.session_id != "" and not config.resume_enabled
arg "--resume" config.resume_session when config.resume_enabled
PROMPT_POSITION prepend
`
	return &mockAgentConfigProvider{
		agentDef: &agentDomain.Agent{
			Slug:           "claude-code",
			Name:           "Claude Code",
			LaunchCommand:  "claude",
			SupportedModes: "pty",
			AgentfileSource:  &agentfile,
		},
		config:   agentDomain.ConfigValues{},
		creds:    agentDomain.EncryptedCredentials{},
		isRunner: true,
	}
}

func setupOrchestrator(t *testing.T, opts ...func(*PodOrchestratorDeps)) (*PodOrchestrator, *PodService, *gorm.DB) {
	t.Helper()
	db := setupOrchestratorTestDB(t)
	podSvc := newTestPodService(db)

	provider := newTestProvider()
	configBuilder := agent.NewConfigBuilder(provider)

	deps := &PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: configBuilder,
		AgentResolver: &mockAgentResolver{agentDef: provider.agentDef},
	}

	for _, opt := range opts {
		opt(deps)
	}

	return NewPodOrchestrator(deps), podSvc, db
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

func withRunnerSelector(rs RunnerSelectorForOrchestrator) func(*PodOrchestratorDeps) {
	return func(d *PodOrchestratorDeps) { d.RunnerSelector = rs }
}

func withAgentResolver(ar AgentResolverForOrchestrator) func(*PodOrchestratorDeps) {
	return func(d *PodOrchestratorDeps) { d.AgentResolver = ar }
}

func ptrStr(s string) *string { return &s }
