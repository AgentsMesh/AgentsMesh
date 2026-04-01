package agentpod

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

// ========== CRUD Operations ==========

// GetAutopilotController retrieves an AutopilotController by organization ID and key
func (s *AutopilotControllerService) GetAutopilotController(ctx context.Context, orgID int64, autopilotPodKey string) (*agentpod.AutopilotController, error) {
	controller, err := s.repo.GetByOrgAndKey(ctx, orgID, autopilotPodKey)
	if err != nil {
		return nil, err
	}
	if controller == nil {
		return nil, ErrAutopilotControllerNotFound
	}
	return controller, nil
}

// ListAutopilotControllers lists all AutopilotControllers for an organization
func (s *AutopilotControllerService) ListAutopilotControllers(ctx context.Context, orgID int64) ([]*agentpod.AutopilotController, error) {
	return s.repo.ListByOrg(ctx, orgID)
}

// CreateAutopilotController creates a new AutopilotController record.
func (s *AutopilotControllerService) CreateAutopilotController(ctx context.Context, pod *agentpod.AutopilotController) error {
	return s.repo.Create(ctx, pod)
}

// UpdateAutopilotController updates an existing AutopilotController
func (s *AutopilotControllerService) UpdateAutopilotController(ctx context.Context, pod *agentpod.AutopilotController) error {
	return s.repo.Save(ctx, pod)
}

// UpdateAutopilotControllerStatus updates the status fields of an AutopilotController
func (s *AutopilotControllerService) UpdateAutopilotControllerStatus(ctx context.Context, autopilotPodKey string, updates map[string]interface{}) error {
	return s.repo.UpdateStatusByKey(ctx, autopilotPodKey, updates)
}

// GetIterations retrieves all iterations for an AutopilotController
func (s *AutopilotControllerService) GetIterations(ctx context.Context, autopilotPodID int64) ([]*agentpod.AutopilotIteration, error) {
	return s.repo.ListIterations(ctx, autopilotPodID)
}

// CreateIteration creates a new iteration record
func (s *AutopilotControllerService) CreateIteration(ctx context.Context, iteration *agentpod.AutopilotIteration) error {
	return s.repo.CreateIteration(ctx, iteration)
}

// GetAutopilotControllerByKey retrieves an AutopilotController by key only
func (s *AutopilotControllerService) GetAutopilotControllerByKey(ctx context.Context, autopilotPodKey string) (*agentpod.AutopilotController, error) {
	controller, err := s.repo.GetByKey(ctx, autopilotPodKey)
	if err != nil {
		return nil, err
	}
	if controller == nil {
		return nil, ErrAutopilotControllerNotFound
	}
	return controller, nil
}

// GetActiveAutopilotControllerForPod retrieves active AutopilotController for a pod
func (s *AutopilotControllerService) GetActiveAutopilotControllerForPod(ctx context.Context, podKey string) (*agentpod.AutopilotController, error) {
	controller, err := s.repo.GetActiveForPod(ctx, podKey)
	if err != nil {
		return nil, err
	}
	if controller == nil {
		return nil, ErrAutopilotControllerNotFound
	}
	return controller, nil
}

// GetApprovalTimedOut returns autopilot controllers in waiting_approval phase
// whose approval timeout has elapsed. Used by the scheduler to stop stale approvals.
func (s *AutopilotControllerService) GetApprovalTimedOut(ctx context.Context, orgIDs []int64) ([]*agentpod.AutopilotController, error) {
	return s.repo.GetApprovalTimedOut(ctx, orgIDs)
}
