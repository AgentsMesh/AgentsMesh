package agentpod

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
	"github.com/anthropics/agentsmesh/backend/internal/testutil"
)

// setupIntegrationOrchestrator creates a PodOrchestrator wired with real DB
// (via testutil.SetupTestDB) and real PodService/ConfigBuilder, plus mock
// coordinator and billing for control over external interactions.
func setupIntegrationOrchestrator(t *testing.T, opts ...func(*PodOrchestratorDeps)) (*PodOrchestrator, *PodService, context.Context) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	ctx := context.Background()

	userID := testutil.CreateUser(t, db, "test@example.com", "testuser")
	orgID := testutil.CreateOrg(t, db, "test-org", userID)
	runnerID := testutil.CreateRunner(t, db, orgID, "runner-001")

	agentfileSrc := "AGENT claude\nEXECUTABLE claude\nMCP ON\nPROMPT_POSITION prepend\n"
	testutil.CreateAgent(t, db, "claude-code", "Claude Code", agentfileSrc)

	podSvc := NewPodService(infra.NewPodRepository(db))
	provider := &mockAgentConfigProvider{
		agentDef: &agentDomain.Agent{
			Slug: "claude-code", Name: "Claude Code",
			LaunchCommand: "claude", SupportedModes: "pty",
			AgentfileSource: &agentfileSrc,
		},
		config: agentDomain.ConfigValues{},
		creds:  agentDomain.EncryptedCredentials{},
		isRunner: true,
	}
	configBuilder := agent.NewConfigBuilder(provider)

	deps := &PodOrchestratorDeps{
		PodService:    podSvc,
		ConfigBuilder: configBuilder,
		AgentResolver: &mockAgentResolver{agentDef: provider.agentDef},
	}
	for _, opt := range opts {
		opt(deps)
	}

	// Store IDs in context for test use via helper
	ctx = context.WithValue(ctx, ctxKeyOrgID, orgID)
	ctx = context.WithValue(ctx, ctxKeyUserID, userID)
	ctx = context.WithValue(ctx, ctxKeyRunnerID, runnerID)

	return NewPodOrchestrator(deps), podSvc, ctx
}

// Context keys for passing fixture IDs through context.
type ctxKey string

const (
	ctxKeyOrgID    ctxKey = "orgID"
	ctxKeyUserID   ctxKey = "userID"
	ctxKeyRunnerID ctxKey = "runnerID"
)

func ctxOrgID(ctx context.Context) int64    { return ctx.Value(ctxKeyOrgID).(int64) }
func ctxUserID(ctx context.Context) int64   { return ctx.Value(ctxKeyUserID).(int64) }
func ctxRunnerID(ctx context.Context) int64 { return ctx.Value(ctxKeyRunnerID).(int64) }

// ---------- Test 1: Full lifecycle Create -> Running -> Terminated ----------

func TestPodLifecycle_CreateToTerminated(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc, ctx := setupIntegrationOrchestrator(t, withCoordinator(coord))

	// Step 1: Create pod via orchestrator
	result, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		RunnerID:       ctxRunnerID(ctx),
		AgentSlug:      "claude-code",
		Cols:           120, Rows: 40,
	})
	require.NoError(t, err)
	podKey := result.Pod.PodKey
	assert.Equal(t, podDomain.StatusInitializing, result.Pod.Status)
	assert.True(t, coord.createPodCalled, "coordinator should receive CreatePod")

	// Step 2: Simulate runner reporting pod_created -> running
	err = podSvc.HandlePodCreated(ctx, podKey, 12345, "/sandbox/path", "main")
	require.NoError(t, err)

	pod, err := podSvc.GetPod(ctx, podKey)
	require.NoError(t, err)
	assert.Equal(t, podDomain.StatusRunning, pod.Status)
	assert.NotNil(t, pod.SandboxPath)
	assert.Equal(t, "/sandbox/path", *pod.SandboxPath)

	// Step 3: Simulate runner reporting pod_terminated
	err = podSvc.HandlePodTerminated(ctx, podKey, nil)
	require.NoError(t, err)

	pod, err = podSvc.GetPod(ctx, podKey)
	require.NoError(t, err)
	assert.Equal(t, podDomain.StatusTerminated, pod.Status)
	assert.NotNil(t, pod.FinishedAt)
}

// ---------- Test 2: Resume mode reuses session_id + sets source_pod_key ----------

func TestPodLifecycle_ResumeMode(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc, ctx := setupIntegrationOrchestrator(t, withCoordinator(coord))

	orgID, userID, runnerID := ctxOrgID(ctx), ctxUserID(ctx), ctxRunnerID(ctx)

	// Create and terminate source pod
	source, err := podSvc.CreatePod(ctx, &CreatePodRequest{
		OrganizationID: orgID, RunnerID: runnerID,
		AgentSlug: "claude-code", CreatedByID: userID,
		SessionID: "session-abc",
	})
	require.NoError(t, err)
	err = podSvc.HandlePodTerminated(ctx, source.PodKey, nil)
	require.NoError(t, err)

	// Resume from terminated pod
	result, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: orgID, UserID: userID,
		SourcePodKey: source.PodKey,
	})
	require.NoError(t, err)

	resumed := result.Pod
	assert.NotEqual(t, source.PodKey, resumed.PodKey, "resumed pod gets new key")
	assert.NotNil(t, resumed.SourcePodKey)
	assert.Equal(t, source.PodKey, *resumed.SourcePodKey, "source_pod_key tracks origin")
	assert.NotNil(t, resumed.SessionID)
	assert.Equal(t, "session-abc", *resumed.SessionID, "session_id reused from source")
	assert.Equal(t, runnerID, resumed.RunnerID, "runner_id inherited")
}

// ---------- Test 3: Billing quota rejection ----------

func TestPodLifecycle_BillingQuotaReject(t *testing.T) {
	quotaErr := errors.New("concurrent pod limit reached")
	billing := &mockBillingService{err: quotaErr}
	orch, _, ctx := setupIntegrationOrchestrator(t, withBilling(billing))

	_, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		RunnerID:       ctxRunnerID(ctx),
		AgentSlug:      "claude-code",
	})

	require.Error(t, err)
	assert.Equal(t, quotaErr, err, "billing error propagated directly")
}

// ---------- Test 4: Runner auto-selection ----------

func TestPodLifecycle_RunnerAutoSelect(t *testing.T) {
	coord := &mockPodCoordinator{}
	autoRunner := &runnerDomain.Runner{ID: 999, NodeID: "auto-runner"}
	agentfileSrc := "AGENT claude\nEXECUTABLE claude\nPROMPT_POSITION prepend\n"

	selector := &mockRunnerSelector{runner: autoRunner}
	resolver := &mockAgentResolver{
		agentDef: &agentDomain.Agent{
			Slug: "claude-code", SupportedModes: "pty",
			AgentfileSource: &agentfileSrc,
		},
	}

	orch, _, ctx := setupIntegrationOrchestrator(t,
		withCoordinator(coord),
		withRunnerSelector(selector),
		withAgentResolver(resolver),
	)

	result, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		RunnerID:       0, // trigger auto-select
		AgentSlug:      "claude-code",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(999), result.Pod.RunnerID, "auto-selected runner used")
	assert.Equal(t, int64(999), coord.lastRunnerID)
}

// ---------- Test 5: Dispatch failure marks pod init_failed ----------

func TestPodLifecycle_DispatchFailure(t *testing.T) {
	coord := &mockPodCoordinator{err: errors.New("connection refused")}
	orch, podSvc, ctx := setupIntegrationOrchestrator(t, withCoordinator(coord))

	_, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		RunnerID:       ctxRunnerID(ctx),
		AgentSlug:      "claude-code",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRunnerDispatchFailed)

	// The pod was created in DB before dispatch; verify it's marked as error
	podKey := coord.lastCmd.PodKey
	pod, dbErr := podSvc.GetPod(ctx, podKey)
	require.NoError(t, dbErr)
	assert.Equal(t, podDomain.StatusError, pod.Status, "pod should be error after dispatch failure")
	assert.NotNil(t, pod.ErrorCode)
	assert.Equal(t, errCodeRunnerUnreachable, *pod.ErrorCode)
}

// ---------- Test 6: AgentFile layer merge extracts overrides ----------

func TestPodLifecycle_AgentfileLayerMerge(t *testing.T) {
	coord := &mockPodCoordinator{}

	baseAgentfile := "AGENT claude\nEXECUTABLE claude\nMODE pty\nMCP ON\nPROMPT_POSITION prepend\n"
	resolver := &mockAgentResolver{
		agentDef: &agentDomain.Agent{
			Slug: "claude-code", SupportedModes: "pty,acp",
			AgentfileSource: &baseAgentfile,
		},
	}

	orch, podSvc, ctx := setupIntegrationOrchestrator(t,
		withCoordinator(coord),
		withAgentResolver(resolver),
	)

	layer := `BRANCH "feature-x"
CONFIG permission_mode = "bypassPermissions"
PROMPT "Do the thing"
`
	result, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		RunnerID:       ctxRunnerID(ctx),
		AgentSlug:      "claude-code",
		AgentfileLayer:   &layer,
		Cols:           120, Rows: 40,
	})

	require.NoError(t, err)
	pod := result.Pod

	// Verify DB reflects merged values
	dbPod, err := podSvc.GetPod(ctx, pod.PodKey)
	require.NoError(t, err)
	assert.NotNil(t, dbPod.BranchName)
	assert.Equal(t, "feature-x", *dbPod.BranchName, "branch extracted from layer")
	assert.NotNil(t, dbPod.PermissionMode)
	assert.Equal(t, "bypassPermissions", *dbPod.PermissionMode, "permission_mode extracted from layer")
	assert.Equal(t, "Do the thing", dbPod.Prompt, "prompt extracted from layer")
}
