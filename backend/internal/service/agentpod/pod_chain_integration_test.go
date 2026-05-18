package agentpod

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/service/agent"
)

// ==================== Helpers ====================

// acpAgentfile returns a base AgentFile that supports both pty and acp modes.
func acpAgentfile() string {
	return "AGENT claude\nEXECUTABLE claude\nMODE pty\nMCP ON\nPROMPT_POSITION prepend\n"
}

// acpProvider creates a mockAgentConfigProvider for an agent supporting pty+acp.
func acpProvider(agentfileSrc string) *mockAgentConfigProvider {
	return &mockAgentConfigProvider{
		agentDef: &agentDomain.Agent{
			Slug: "claude-code", Name: "Claude Code",
			LaunchCommand: "claude", SupportedModes: "pty,acp",
			AgentfileSource: &agentfileSrc, UsesLegacyColumns: true,
		},
		config: agentDomain.ConfigValues{}, creds: agentDomain.EncryptedCredentials{},
		isRunner: true,
	}
}

// acpResolver builds a mockAgentResolver that supports pty+acp.
func acpResolver(agentfileSrc string) *mockAgentResolver {
	return &mockAgentResolver{
		agentDef: &agentDomain.Agent{
			Slug: "claude-code", SupportedModes: "pty,acp",
			AgentfileSource: &agentfileSrc, UsesLegacyColumns: true,
		},
	}
}

// withConfigBuilder injects a custom ConfigBuilder into PodOrchestratorDeps.
func withConfigBuilder(cb *agent.ConfigBuilder) func(*PodOrchestratorDeps) {
	return func(d *PodOrchestratorDeps) { d.ConfigBuilder = cb }
}

// ==================== Test 1: AgentFile Layer -> Command ====================

func TestPodChain_AgentfileLayerToCommand(t *testing.T) {
	coord := &mockPodCoordinator{}
	agentfileSrc := acpAgentfile()
	provider := acpProvider(agentfileSrc)
	resolver := acpResolver(agentfileSrc)

	orch, podSvc, ctx := setupIntegrationOrchestrator(t,
		withCoordinator(coord),
		withAgentResolver(resolver),
		withConfigBuilder(agent.NewConfigBuilder(provider)),
	)

	layer := "MODE acp\nBRANCH \"feature-x\"\nPROMPT \"do something\"\nCONFIG permission_mode = \"bypassPermissions\"\n"
	result, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		RunnerID:       ctxRunnerID(ctx),
		AgentSlug:      "claude-code",
		AgentfileLayer:   &layer,
		Cols:           120, Rows: 40,
	})
	require.NoError(t, err)

	// Verify DB record reflects merged values
	dbPod, err := podSvc.GetPod(ctx, result.Pod.PodKey)
	require.NoError(t, err)
	assert.Equal(t, podDomain.InteractionModeACP, dbPod.InteractionMode)
	require.NotNil(t, dbPod.BranchName)
	assert.Equal(t, "feature-x", *dbPod.BranchName)
	require.NotNil(t, dbPod.PermissionMode)
	assert.Equal(t, "bypassPermissions", *dbPod.PermissionMode)
	assert.Equal(t, "do something", dbPod.Prompt)

	// Verify gRPC command content — Backend eval produces execution instructions
	cmd := coord.lastCmd
	require.NotNil(t, cmd)
	assert.Equal(t, result.Pod.PodKey, cmd.PodKey)
	assert.Equal(t, "claude", cmd.LaunchCommand)
	assert.Equal(t, "acp", cmd.InteractionMode, "MODE acp from layer should be reflected")
	assert.Equal(t, "do something", cmd.Prompt)
	assert.Equal(t, int32(120), cmd.Cols)
	assert.Equal(t, int32(40), cmd.Rows)

	// SandboxConfig.SourceBranch should reflect the BRANCH override
	if cmd.SandboxConfig != nil {
		assert.Equal(t, "feature-x", cmd.SandboxConfig.SourceBranch)
	}
}

// ==================== Test 2: Repo Slug Resolution ====================

func TestPodChain_RepoSlugResolution(t *testing.T) {
	coord := &mockPodCoordinator{}
	agentfileSrc := acpAgentfile()
	provider := acpProvider(agentfileSrc)

	// Mock repo service resolves slug -> repository with clone URL
	repoSvc := &mockRepoService{
		repo: &gitprovider.Repository{
			ID:            77,
			HttpCloneURL:  "https://github.com/org/repo-slug.git",
			DefaultBranch: "main",
		},
	}
	resolver := acpResolver(agentfileSrc)

	orch, _, ctx := setupIntegrationOrchestrator(t,
		withCoordinator(coord),
		withAgentResolver(resolver),
		withRepoSvc(repoSvc),
		withConfigBuilder(agent.NewConfigBuilder(provider)),
	)

	layer := "REPO \"org/repo-slug\"\n"
	result, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		RunnerID:       ctxRunnerID(ctx),
		AgentSlug:      "claude-code",
		AgentfileLayer:   &layer,
	})
	require.NoError(t, err)

	// Pod should have RepositoryID set from slug resolution
	require.NotNil(t, result.Pod.RepositoryID)
	assert.Equal(t, int64(77), *result.Pod.RepositoryID)

	// Command sandbox config should carry the repo URL
	cmd := coord.lastCmd
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.SandboxConfig)
	assert.Equal(t, "https://github.com/org/repo-slug.git", cmd.SandboxConfig.HttpCloneUrl)
}

// ==================== Test 3: Credential Flow ====================

func TestPodChain_CredentialFlow(t *testing.T) {
	coord := &mockPodCoordinator{}
	agentfileSrc := acpAgentfile()

	// Provider that resolves CREDENTIAL name to encrypted credentials
	credProvider := &mockAgentConfigProvider{
		agentDef: &agentDomain.Agent{
			Slug: "claude-code", Name: "Claude Code",
			LaunchCommand: "claude", SupportedModes: "pty,acp",
			AgentfileSource: &agentfileSrc, UsesLegacyColumns: true,
		},
		config:   agentDomain.ConfigValues{},
		creds:    agentDomain.EncryptedCredentials{"ANTHROPIC_API_KEY": "enc-key-123"},
		isRunner: false,
	}
	resolver := acpResolver(agentfileSrc)

	orch, _, ctx := setupIntegrationOrchestrator(t,
		withCoordinator(coord),
		withAgentResolver(resolver),
		withConfigBuilder(agent.NewConfigBuilder(credProvider)),
	)

	layer := "CREDENTIAL \"my-profile\"\n"
	result, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		RunnerID:       ctxRunnerID(ctx),
		AgentSlug:      "claude-code",
		AgentfileLayer:   &layer,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Pod)

	// The command should carry the resolved credentials from the provider
	cmd := coord.lastCmd
	require.NotNil(t, cmd)
	assert.Equal(t, "enc-key-123", cmd.Credentials["ANTHROPIC_API_KEY"])
	assert.False(t, cmd.IsRunnerHost, "credential profile should not be runner host")
}

// ==================== Test 4: Unsupported Interaction Mode ====================

func TestPodChain_UnsupportedInteractionMode(t *testing.T) {
	coord := &mockPodCoordinator{}

	// Agent only supports "pty"
	ptyOnlySrc := "AGENT claude\nEXECUTABLE claude\nMODE pty\nPROMPT_POSITION prepend\n"
	ptyOnlyResolver := &mockAgentResolver{
		agentDef: &agentDomain.Agent{
			Slug: "claude-code", SupportedModes: "pty",
			AgentfileSource: &ptyOnlySrc, UsesLegacyColumns: true,
		},
	}
	provider := acpProvider(ptyOnlySrc) // provider doesn't matter for mode validation

	orch, podSvc, ctx := setupIntegrationOrchestrator(t,
		withCoordinator(coord),
		withAgentResolver(ptyOnlyResolver),
		withConfigBuilder(agent.NewConfigBuilder(provider)),
	)

	// AgentFile layer requests MODE acp, but agent only supports pty
	layer := "MODE acp\n"
	_, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		RunnerID:       ctxRunnerID(ctx),
		AgentSlug:      "claude-code",
		AgentfileLayer:   &layer,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedInteractionMode)
	assert.False(t, coord.createPodCalled, "coordinator should not be called on mode mismatch")

	// Verify no pod was created in DB (mode check happens before DB insert)
	// We check via pod service — the only pod in DB was NOT created by this call
	_ = podSvc // pod service available but no pod key to query
}

// ==================== Test 5: ConfigBuilder Failure ====================

func TestPodChain_ConfigBuilderFailure(t *testing.T) {
	coord := &mockPodCoordinator{}

	// Provider that fails on GetAgent (simulating config build failure)
	failProvider := &mockAgentConfigProvider{
		agentErr: errors.New("agent config not available"),
	}

	agentfileSrc := acpAgentfile()
	resolver := acpResolver(agentfileSrc)

	orch, podSvc, ctx := setupIntegrationOrchestrator(t,
		withCoordinator(coord),
		withAgentResolver(resolver),
		withConfigBuilder(agent.NewConfigBuilder(failProvider)),
	)

	result, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		RunnerID:       ctxRunnerID(ctx),
		AgentSlug:      "claude-code",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrConfigBuildFailed)
	assert.Nil(t, result)
	assert.False(t, coord.createPodCalled, "coordinator should not be called when config build fails")

	// Pod was created in DB before config build; it remains in initializing status
	// (MarkInitFailed is only called on dispatch failure, not config build failure)
	_ = podSvc
}

// ==================== Test 6: Dispatch Failure Marks Error ====================

func TestPodChain_DispatchFailureMarksError(t *testing.T) {
	coord := &mockPodCoordinator{err: errors.New("runner connection refused")}

	agentfileSrc := acpAgentfile()
	provider := acpProvider(agentfileSrc)
	resolver := acpResolver(agentfileSrc)

	orch, podSvc, ctx := setupIntegrationOrchestrator(t,
		withCoordinator(coord),
		withAgentResolver(resolver),
		withConfigBuilder(agent.NewConfigBuilder(provider)),
	)

	layer := "PROMPT \"deploy fix\"\n"
	_, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		RunnerID:       ctxRunnerID(ctx),
		AgentSlug:      "claude-code",
		AgentfileLayer:   &layer,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRunnerDispatchFailed)

	// The command was built and sent to coordinator (which failed)
	require.NotNil(t, coord.lastCmd, "command should have been built before dispatch failure")

	// Pod should exist in DB with error status
	podKey := coord.lastCmd.PodKey
	dbPod, dbErr := podSvc.GetPod(ctx, podKey)
	require.NoError(t, dbErr)
	assert.Equal(t, podDomain.StatusError, dbPod.Status)
	require.NotNil(t, dbPod.ErrorCode)
	assert.Equal(t, errCodeRunnerUnreachable, *dbPod.ErrorCode)
	require.NotNil(t, dbPod.ErrorMessage)
	assert.Contains(t, *dbPod.ErrorMessage, "runner connection refused")
}

// TestResumeIntegration_CodexFullChain exercises the full create → snapshot →
// terminate → resume chain for a non-Claude (Codex) agent. It verifies that:
//   - the resolved AgentFile CONFIG snapshot lands on Pod.ResolvedConfig
//   - codex pods do NOT touch legacy Pod.Model / Pod.PermissionMode columns
//   - resume re-injects the source pod's CONFIG snapshot and produces the
//     `codex resume --last` launch command honored by Codex CLI
func TestResumeIntegration_CodexFullChain(t *testing.T) {
	coord := &mockPodCoordinator{}
	codexAgentfile := "AGENT codex\nEXECUTABLE codex\nMODE pty\n" +
		"CONFIG approval_mode SELECT(\"untrusted\", \"on-request\", \"never\") = \"untrusted\"\n" +
		"PROMPT_POSITION append\n" +
		"arg \"resume\" \"--last\" when config.resume_enabled and mode != \"acp\"\n" +
		"arg \"--ask-for-approval\" config.approval_mode when config.approval_mode != \"\" and mode != \"acp\"\n"

	codexProvider := &mockAgentConfigProvider{
		agentDef: &agentDomain.Agent{
			Slug:              "codex-cli",
			Name:              "Codex CLI",
			LaunchCommand:     "codex",
			SupportedModes:    "pty",
			AgentfileSource:   &codexAgentfile,
			UsesLegacyColumns: false,
		},
		config:   agentDomain.ConfigValues{},
		creds:    agentDomain.EncryptedCredentials{},
		isRunner: true,
	}
	codexResolver := &mockAgentResolver{agentDef: codexProvider.agentDef}

	orch, podSvc, ctx := setupIntegrationOrchestrator(t,
		withCoordinator(coord),
		withAgentResolver(codexResolver),
		withConfigBuilder(agent.NewConfigBuilder(codexProvider)),
	)

	sourceLayer := `CONFIG approval_mode = "never"`
	source, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		RunnerID:       ctxRunnerID(ctx),
		AgentSlug:      "codex-cli",
		AgentfileLayer: &sourceLayer,
	})
	require.NoError(t, err)
	require.NotNil(t, source.Pod)

	sourceDB, err := podSvc.GetPod(ctx, source.Pod.PodKey)
	require.NoError(t, err)
	assert.Equal(t, "never", sourceDB.ResolvedConfig["approval_mode"], "snapshot should land on ResolvedConfig")
	assert.Nil(t, sourceDB.Model, "codex must not write legacy Model column")
	assert.Nil(t, sourceDB.PermissionMode, "codex must not write legacy PermissionMode column")

	// Terminate + attach a sandbox path so resume can pick up local_path.
	sandboxPath := "/home/user/sandbox/codex-source"
	_, err = podSvc.UpdateByKey(ctx, source.Pod.PodKey, map[string]interface{}{
		"sandbox_path": sandboxPath,
		"status":       podDomain.StatusTerminated,
	})
	require.NoError(t, err)

	resumed, err := orch.CreatePod(ctx, &OrchestrateCreatePodRequest{
		OrganizationID: ctxOrgID(ctx),
		UserID:         ctxUserID(ctx),
		SourcePodKey:   source.Pod.PodKey,
	})
	require.NoError(t, err)
	require.NotNil(t, resumed.Pod)

	resumedDB, err := podSvc.GetPod(ctx, resumed.Pod.PodKey)
	require.NoError(t, err)
	assert.Equal(t, "never", resumedDB.ResolvedConfig["approval_mode"], "snapshot should survive resume")
	assert.Nil(t, resumedDB.Model)
	assert.Nil(t, resumedDB.PermissionMode)
	assert.Equal(t, "codex-cli", resumedDB.AgentSlug, "agent slug should be inherited from source pod")

	require.NotNil(t, coord.lastCmd)
	assert.Equal(t, "codex", coord.lastCmd.LaunchCommand)
	assert.Equal(t, "append", coord.lastCmd.PromptPosition)
	assert.Equal(t, []string{"resume", "--last", "--ask-for-approval", "never"}, coord.lastCmd.LaunchArgs)
	require.NotNil(t, coord.lastCmd.SandboxConfig)
	assert.Equal(t, sandboxPath, coord.lastCmd.SandboxConfig.LocalPath)
}
