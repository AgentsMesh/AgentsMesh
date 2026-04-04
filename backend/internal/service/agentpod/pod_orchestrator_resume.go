package agentpod

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	podDomain "github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
)

// handleResumeMode validates the source pod and inherits configuration.
func (o *PodOrchestrator) handleResumeMode(ctx context.Context, req *OrchestrateCreatePodRequest) (*podDomain.Pod, string, error) {
	sourcePod, err := o.podService.GetPod(ctx, req.SourcePodKey)
	if err != nil {
		return nil, "", ErrSourcePodNotFound
	}

	if sourcePod.OrganizationID != req.OrganizationID {
		return nil, "", ErrSourcePodAccessDenied
	}

	if sourcePod.Status != podDomain.StatusTerminated &&
		sourcePod.Status != podDomain.StatusCompleted &&
		sourcePod.Status != podDomain.StatusOrphaned {
		return nil, "", ErrSourcePodNotTerminated
	}

	existingResumePod, err := o.podService.GetActivePodBySourcePodKey(ctx, req.SourcePodKey)
	if err == nil && existingResumePod != nil {
		return nil, "", ErrSourcePodAlreadyResumed
	}

	if req.RunnerID == 0 {
		req.RunnerID = sourcePod.RunnerID
	} else if sourcePod.RunnerID != req.RunnerID {
		return nil, "", ErrResumeRunnerMismatch
	}

	// Inherit configuration from source pod
	if req.AgentSlug == "" {
		req.AgentSlug = sourcePod.AgentSlug
	}
	if req.RepositoryID == nil {
		req.RepositoryID = sourcePod.RepositoryID
	}
	if req.TicketID == nil {
		req.TicketID = sourcePod.TicketID
	}
	if req.BranchName == nil {
		req.BranchName = sourcePod.BranchName
	}

	// Reuse session ID from source pod
	var sessionID string
	if sourcePod.SessionID != nil && *sourcePod.SessionID != "" {
		sessionID = *sourcePod.SessionID
	} else {
		sessionID = uuid.New().String()
	}

	resumeAgentSession := req.ResumeAgentSession == nil || *req.ResumeAgentSession
	// Resume fields (resume_enabled, resume_session) are injected into AgentFile
	// via systemOverrides in CreatePod, not through ConfigOverrides.
	_ = resumeAgentSession

	return sourcePod, sessionID, nil
}

// getUserGitCredential retrieves the default Git credential for a user.
// Returns nil if using runner_local (Runner will use local Git config).
func (o *PodOrchestrator) getUserGitCredential(ctx context.Context, userID int64) *userService.DecryptedCredential {
	if o.userService == nil {
		return nil
	}

	defaultCred, err := o.userService.GetDefaultGitCredential(ctx, userID)
	if err != nil || defaultCred == nil {
		return nil
	}

	if defaultCred.CredentialType == "runner_local" {
		return nil
	}

	decrypted, err := o.userService.GetDecryptedCredentialToken(ctx, userID, defaultCred.ID)
	if err != nil {
		slog.Warn("failed to decrypt Git credential", "user_id", userID, "error", err)
		return nil
	}

	return decrypted
}
