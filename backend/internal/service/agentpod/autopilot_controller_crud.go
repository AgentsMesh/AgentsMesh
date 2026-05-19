package agentpod

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

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

func (s *AutopilotControllerService) ListAutopilotControllers(ctx context.Context, orgID int64) ([]*agentpod.AutopilotController, error) {
	return s.repo.ListByOrg(ctx, orgID)
}

func (s *AutopilotControllerService) CreateAutopilotController(ctx context.Context, pod *agentpod.AutopilotController) error {
	return s.repo.Create(ctx, pod)
}

func (s *AutopilotControllerService) UpdateAutopilotController(ctx context.Context, pod *agentpod.AutopilotController) error {
	return s.repo.Save(ctx, pod)
}

func (s *AutopilotControllerService) UpdateAutopilotControllerStatus(ctx context.Context, autopilotPodKey string, updates map[string]interface{}) error {
	return s.repo.UpdateStatusByKey(ctx, autopilotPodKey, updates)
}

func (s *AutopilotControllerService) GetIterations(ctx context.Context, autopilotPodID int64) ([]*agentpod.AutopilotIteration, error) {
	return s.repo.ListIterations(ctx, autopilotPodID)
}

func (s *AutopilotControllerService) CreateIteration(ctx context.Context, iteration *agentpod.AutopilotIteration) error {
	return s.repo.CreateIteration(ctx, iteration)
}

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

func (s *AutopilotControllerService) GetApprovalTimedOut(ctx context.Context, orgIDs []int64) ([]*agentpod.AutopilotController, error) {
	return s.repo.GetApprovalTimedOut(ctx, orgIDs)
}
