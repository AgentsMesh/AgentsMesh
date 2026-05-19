package agentpod

import (
	"context"
	"log/slog"

	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

func (o *PodOrchestrator) buildAffinityHints(ctx context.Context, req *OrchestrateCreatePodRequest) *runnerDomain.AffinityHints {
	hints := &runnerDomain.AffinityHints{}

	repoSlug := ""
	if req.AgentfileLayer != nil {
		repoSlug = peekRepoSlug(*req.AgentfileLayer)
	}
	if repoSlug == "" && o.agentResolver != nil {
		if agentDef, err := o.agentResolver.GetAgent(ctx, req.AgentSlug); err == nil && agentDef != nil && agentDef.AgentfileSource != nil {
			repoSlug = peekRepoSlug(*agentDef.AgentfileSource)
		}
	}

	if repoSlug != "" && o.repoService != nil {
		repo, err := o.repoService.FindByOrgSlug(ctx, req.OrganizationID, repoSlug)
		if err == nil && repo != nil {
			hints.RepositoryID = &repo.ID
		}
	}

	return hints
}

// fetchRepoHistory queries pod history for repo affinity scoring.
// Returns nil if no repo hint or podRepo unavailable.
func (o *PodOrchestrator) fetchRepoHistory(ctx context.Context, orgID int64, hints *runnerDomain.AffinityHints) map[int64]int {
	if hints == nil || hints.RepositoryID == nil || o.podRepo == nil {
		return nil
	}
	histories, err := o.podRepo.ListRunnersByRepo(ctx, orgID, *hints.RepositoryID, 10)
	if err != nil {
		slog.Warn("repo history lookup failed, ignoring repo affinity", "error", err)
		return nil
	}
	m := make(map[int64]int, len(histories))
	for _, h := range histories {
		m[h.RunnerID] = h.PodCount
	}
	return m
}
