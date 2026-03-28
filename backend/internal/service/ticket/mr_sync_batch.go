package ticket

import (
	"context"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// CheckPodForNewMR checks if a pod's branch has an MR.
func (s *MRSyncService) CheckPodForNewMR(ctx context.Context, pod *agentpod.Pod) (*ticket.MergeRequest, error) {
	if pod.BranchName == nil || pod.TicketID == nil {
		return nil, nil
	}
	return s.checkPod(ctx, pod.ID, pod.OrganizationID, *pod.BranchName, *pod.TicketID)
}

// BatchCheckPods checks active pods for new MRs.
func (s *MRSyncService) BatchCheckPods(ctx context.Context) ([]*ticket.MergeRequest, error) {
	if s.gitProvider == nil {
		return nil, ErrNoGitProvider
	}

	pods, err := s.repo.FindPodsWithoutMR(ctx)
	if err != nil {
		return nil, err
	}

	var newMRs []*ticket.MergeRequest
	for _, p := range pods {
		if p.BranchName == nil || p.TicketID == nil {
			continue
		}
		mr, err := s.checkPod(ctx, p.ID, p.OrganizationID, *p.BranchName, *p.TicketID)
		if err != nil {
			slog.Warn("batch check pod failed", "pod_id", p.ID, "error", err)
			continue
		}
		if mr != nil {
			newMRs = append(newMRs, mr)
		}
	}

	slog.Info("batch check pods completed", "checked", len(pods), "new_mrs", len(newMRs))
	return newMRs, nil
}

// checkPod is the internal implementation shared by CheckPodForNewMR and BatchCheckPods.
func (s *MRSyncService) checkPod(ctx context.Context, podID, orgID int64, branchName string, ticketID int64) (*ticket.MergeRequest, error) {
	if s.gitProvider == nil {
		return nil, ErrNoGitProvider
	}

	t, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t == nil || t.RepositoryID == nil {
		return nil, ErrNoRepositoryLink
	}

	externalID, err := s.repo.GetRepoExternalID(ctx, *t.RepositoryID)
	if err != nil {
		return nil, err
	}

	mrs, err := s.gitProvider.ListMergeRequestsByBranch(ctx, externalID, branchName, "all")
	if err != nil {
		return nil, err
	}
	if len(mrs) == 0 {
		return nil, nil
	}

	mrData := s.buildMRData(mrs[0])
	return s.FindOrCreateMR(ctx, orgID, t, mrData, &podID)
}

// BatchSyncMRStatus syncs status for open MRs.
func (s *MRSyncService) BatchSyncMRStatus(ctx context.Context) ([]*ticket.MergeRequest, error) {
	if s.gitProvider == nil {
		return nil, ErrNoGitProvider
	}

	mrs, err := s.repo.ListOpenMRsWithTicket(ctx)
	if err != nil {
		return nil, err
	}

	var updated []*ticket.MergeRequest
	for _, mr := range mrs {
		if mr.Ticket == nil || mr.Ticket.RepositoryID == nil {
			continue
		}

		externalID, err := s.repo.GetRepoExternalID(ctx, *mr.Ticket.RepositoryID)
		if err != nil {
			slog.Warn("batch sync MR: failed to get repo external ID", "mr_id", mr.ID, "error", err)
			continue
		}

		mrInfo, err := s.gitProvider.GetMergeRequest(ctx, externalID, mr.MRIID)
		if err != nil {
			slog.Warn("batch sync MR: failed to fetch MR from provider", "mr_id", mr.ID, "mr_iid", mr.MRIID, "error", err)
			continue
		}

		mrData := s.buildMRData(mrInfo)
		s.updateMRFromData(mr, mrData)

		if err := s.repo.SaveMR(ctx, mr); err != nil {
			slog.Warn("batch sync MR: failed to save MR", "mr_id", mr.ID, "error", err)
			continue
		}
		updated = append(updated, mr)
	}

	slog.Info("batch sync MR status completed", "total", len(mrs), "updated", len(updated))
	return updated, nil
}

// SyncMRByURL syncs a single MR by its URL.
func (s *MRSyncService) SyncMRByURL(ctx context.Context, mrURL string) (*ticket.MergeRequest, error) {
	mr, err := s.repo.GetMRByURLWithTicket(ctx, mrURL)
	if err != nil {
		return nil, err
	}
	if mr == nil {
		return nil, ErrMRNotFound
	}

	if mr.Ticket == nil || mr.Ticket.RepositoryID == nil {
		return nil, ErrNoRepositoryLink
	}

	externalID, err := s.repo.GetRepoExternalID(ctx, *mr.Ticket.RepositoryID)
	if err != nil {
		return nil, err
	}

	mrInfo, err := s.gitProvider.GetMergeRequest(ctx, externalID, mr.MRIID)
	if err != nil {
		return nil, err
	}

	mrData := s.buildMRData(mrInfo)
	s.updateMRFromData(mr, mrData)

	if err := s.repo.SaveMR(ctx, mr); err != nil {
		return nil, err
	}
	return mr, nil
}
