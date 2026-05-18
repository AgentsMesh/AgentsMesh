package agentpod

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestCreatePod_ResumeMode_AgentSlugMismatch_Rejected(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc, db := setupOrchestrator(t, withCoordinator(coord))

	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:      "claude-code",
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		AgentSlug:      "codex-cli", // Different agent than source pod
		SourcePodKey:   sourcePod.PodKey,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrResumeAgentMismatch))
	assert.Nil(t, result)
	assert.False(t, coord.createPodCalled, "no runner dispatch should happen on cross-agent resume")
}

func TestCreatePod_ResumeMode_AgentSlugMatch_Accepted(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc, db := setupOrchestrator(t, withCoordinator(coord))

	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:      "claude-code",
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	// Explicit AgentSlug matching source — should be accepted (not rejected).
	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		AgentSlug:      "claude-code",
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCreatePod_ResumeMode_Success(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc, db := setupOrchestrator(t, withCoordinator(coord))

	// Create source pod (terminated)
	agentSlug := "claude-code"
	sessionID := "existing-session-123"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		CreatedByID:    1,
		SessionID:      sessionID,
	})
	require.NoError(t, err)

	// Terminate the source pod (use raw SQL to avoid GREATEST() SQLite incompatibility)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Pod)
	// Should inherit runner_id and agent_slug from source pod
	assert.Equal(t, int64(1), result.Pod.RunnerID)
	assert.Equal(t, agentSlug, result.Pod.AgentSlug)
}

func TestCreatePod_ResumeMode_SourcePodNotFound(t *testing.T) {
	orch, _, _ := setupOrchestrator(t)

	_, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   "non-existent-pod",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourcePodNotFound))
}

func TestCreatePod_ResumeMode_AccessDenied(t *testing.T) {
	orch, podSvc, db := setupOrchestrator(t)

	agentSlug := "claude-code"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 999, // Different org
		RunnerID:       1,
		AgentSlug:    agentSlug,
		CreatedByID:    1,
	})
	require.NoError(t, err)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	_, err = orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1, // Different org from source pod
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSourcePodAccessDenied))
}

func TestCreatePod_ResumeMode_NotTerminated(t *testing.T) {
	orch, podSvc, _ := setupOrchestrator(t)

	agentSlug := "claude-code"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
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
	orch, podSvc, db := setupOrchestrator(t, withCoordinator(coord))

	agentSlug := "claude-code"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

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
	orch, podSvc, db := setupOrchestrator(t)

	// Insert a second runner
	db.Exec("INSERT INTO runners (id, node_id, status, current_pods) VALUES (2, 'runner-002', 'online', 0)")

	agentSlug := "claude-code"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1, // Source on runner 1
		AgentSlug:    agentSlug,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

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
	orch, podSvc, db := setupOrchestrator(t, withCoordinator(coord))

	agentSlug := "claude-code"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

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
	orch, podSvc, db := setupOrchestrator(t, withCoordinator(coord))

	agentSlug := "claude-code"
	repoID := int64(10)
	ticketID := int64(20)
	branch := "feature-branch"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		RepositoryID:   &repoID,
		TicketID:       &ticketID,
		BranchName:     &branch,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	assert.Equal(t, agentSlug, result.Pod.AgentSlug)
	assert.Equal(t, &repoID, result.Pod.RepositoryID)
	assert.Equal(t, &ticketID, result.Pod.TicketID)
	assert.Equal(t, &branch, result.Pod.BranchName)
}

func TestCreatePod_ResumeMode_SessionReused(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc, db := setupOrchestrator(t, withCoordinator(coord))

	agentSlug := "claude-code"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		CreatedByID:    1,
		SessionID:      "my-session-id",
	})
	require.NoError(t, err)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	assert.NotNil(t, result.Pod.SessionID)
	assert.Equal(t, "my-session-id", *result.Pod.SessionID)
	assert.Contains(t, coord.lastCmd.LaunchArgs, "--resume")
	assert.Contains(t, coord.lastCmd.LaunchArgs, "my-session-id")
	assert.NotContains(t, coord.lastCmd.LaunchArgs, "--session-id")
}

func TestCreatePod_ResumeMode_CodexUsesCodexResumeLast(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc, db := setupOrchestrator(t,
		withCoordinator(coord),
		withAgentConfigProvider(newCodexTestProvider()),
	)

	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:      "codex-cli",
		CreatedByID:    1,
		SessionID:      "platform-session-id-not-codex-thread",
	})
	require.NoError(t, err)

	sandboxPath := "/home/user/sandbox/codex-source"
	db.Model(&podDomain.Pod{}).Where("pod_key = ?", sourcePod.PodKey).Updates(map[string]interface{}{
		"sandbox_path": sandboxPath,
		"status":       podDomain.StatusTerminated,
	})

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Pod)
	require.True(t, coord.createPodCalled)
	require.NotNil(t, coord.lastCmd)
	require.NotNil(t, coord.lastCmd.SandboxConfig)

	assert.Equal(t, "codex", coord.lastCmd.LaunchCommand)
	assert.Equal(t, "append", coord.lastCmd.PromptPosition)
	assert.Equal(t, sandboxPath, coord.lastCmd.SandboxConfig.LocalPath)
	assert.Equal(t, []string{"resume", "--last", "--ask-for-approval", "untrusted"}, coord.lastCmd.LaunchArgs)
	assert.NotContains(t, coord.lastCmd.LaunchArgs, "platform-session-id-not-codex-thread")
	assert.NotContains(t, coord.lastCmd.LaunchArgs, "--session-id")
}

func TestCreatePod_ResumeMode_CodexPreservesSourceApprovalMode(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc, db := setupOrchestrator(t,
		withCoordinator(coord),
		withAgentConfigProvider(newCodexTestProvider()),
	)

	sourceLayer := `CONFIG approval_mode = "never"`
	source, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:      "codex-cli",
		AgentfileLayer: &sourceLayer,
	})
	require.NoError(t, err)

	sourcePod, err := podSvc.GetPod(context.Background(), source.Pod.PodKey)
	require.NoError(t, err)
	assert.Nil(t, sourcePod.Model)
	assert.Nil(t, sourcePod.PermissionMode)
	assert.Equal(t, "never", sourcePod.ResolvedConfig["approval_mode"])

	sandboxPath := "/home/user/sandbox/codex-never"
	db.Model(&podDomain.Pod{}).Where("pod_key = ?", source.Pod.PodKey).Updates(map[string]interface{}{
		"sandbox_path": sandboxPath,
		"status":       podDomain.StatusTerminated,
	})

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   source.Pod.PodKey,
	})
	require.NoError(t, err)

	assert.Nil(t, result.Pod.Model)
	assert.Nil(t, result.Pod.PermissionMode)
	assert.Equal(t, "never", result.Pod.ResolvedConfig["approval_mode"])
	assert.Equal(t, []string{"resume", "--last", "--ask-for-approval", "never"}, coord.lastCmd.LaunchArgs)
}

func TestCreatePod_ResumeMode_ClaudePreservesSourcePermissionMode(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc, db := setupOrchestrator(t,
		withCoordinator(coord),
		withAgentConfigProvider(newClaudePermissionTestProvider()),
	)

	sourceLayer := `CONFIG permission_mode = "plan"`
	source, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		RunnerID:       1,
		AgentSlug:      "claude-code",
		AgentfileLayer: &sourceLayer,
	})
	require.NoError(t, err)

	sourcePod, err := podSvc.GetPod(context.Background(), source.Pod.PodKey)
	require.NoError(t, err)
	assert.Equal(t, "plan", sourcePod.ResolvedConfig[agentDomain.ConfigKeyPermissionMode])

	db.Model(&podDomain.Pod{}).Where("pod_key = ?", source.Pod.PodKey).Update("status", podDomain.StatusTerminated)

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   source.Pod.PodKey,
	})
	require.NoError(t, err)

	assert.Equal(t, "plan", result.Pod.ResolvedConfig[agentDomain.ConfigKeyPermissionMode])
	assert.Contains(t, coord.lastCmd.LaunchArgs, "--resume")
	assert.Contains(t, coord.lastCmd.LaunchArgs, "--permission-mode")
	assert.Contains(t, coord.lastCmd.LaunchArgs, "plan")
	assert.NotContains(t, coord.lastCmd.LaunchArgs, "bypassPermissions")
}

func TestCreatePod_ResumeMode_ClaudePreservesLegacyPermissionColumn(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc, db := setupOrchestrator(t,
		withCoordinator(coord),
		withAgentConfigProvider(newClaudePermissionTestProvider()),
	)

	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:      "claude-code",
		CreatedByID:    1,
		SessionID:      "legacy-session",
		PermissionMode: "dontAsk",
	})
	require.NoError(t, err)
	db.Model(&podDomain.Pod{}).Where("pod_key = ?", sourcePod.PodKey).Update("status", podDomain.StatusTerminated)

	result, err := orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID: 1,
		UserID:         1,
		SourcePodKey:   sourcePod.PodKey,
	})
	require.NoError(t, err)

	assert.Equal(t, "dontAsk", result.Pod.ResolvedConfig[agentDomain.ConfigKeyPermissionMode])
	assert.Contains(t, coord.lastCmd.LaunchArgs, "--permission-mode")
	assert.Contains(t, coord.lastCmd.LaunchArgs, "dontAsk")
	assert.NotContains(t, coord.lastCmd.LaunchArgs, "bypassPermissions")
}

func TestCreatePod_ResumeMode_NoSessionID_GeneratesNew(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc, db := setupOrchestrator(t, withCoordinator(coord))

	agentSlug := "claude-code"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		CreatedByID:    1,
		SessionID:      "", // No session ID
	})
	require.NoError(t, err)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

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
	orch, podSvc, db := setupOrchestrator(t, withCoordinator(coord))

	agentSlug := "claude-code"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusTerminated, sourcePod.PodKey)

	resumeOff := false
	_, err = orch.CreatePod(context.Background(), &OrchestrateCreatePodRequest{
		OrganizationID:     1,
		UserID:             1,
		SourcePodKey:       sourcePod.PodKey,
		ResumeAgentSession: &resumeOff,
	})

	require.NoError(t, err)
	// When ResumeAgentSession is false, resume_enabled/resume_session should NOT be set
	assert.NotContains(t, coord.lastCmd.LaunchArgs, "--resume")
	// Resume-mode create never injects config.session_id, so --session-id must stay off
	assert.NotContains(t, coord.lastCmd.LaunchArgs, "--session-id")
}

func TestCreatePod_ResumeMode_CompletedPod(t *testing.T) {
	coord := &mockPodCoordinator{}
	orch, podSvc, db := setupOrchestrator(t, withCoordinator(coord))

	agentSlug := "claude-code"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusCompleted, sourcePod.PodKey)

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
	orch, podSvc, db := setupOrchestrator(t, withCoordinator(coord))

	agentSlug := "claude-code"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)
	db.Exec("UPDATE pods SET status = ? WHERE pod_key = ?", podDomain.StatusOrphaned, sourcePod.PodKey)

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
	orch, podSvc, db := setupOrchestrator(t, withCoordinator(coord))

	agentSlug := "claude-code"
	sourcePod, err := podSvc.CreatePod(context.Background(), &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		AgentSlug:    agentSlug,
		CreatedByID:    1,
		SessionID:      "session-1",
	})
	require.NoError(t, err)

	// Set sandbox path on source pod
	sandboxPath := "/home/user/sandbox/pod-123"
	db.Model(&podDomain.Pod{}).Where("pod_key = ?", sourcePod.PodKey).Updates(map[string]interface{}{
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
