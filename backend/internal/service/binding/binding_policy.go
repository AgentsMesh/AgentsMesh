package binding

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func (s *Service) evaluatePolicy(ctx context.Context, initiatorPod, targetPod, policy string) (bool, string) {
	if policy == channel.BindingPolicyExplicitOnly {
		return false, channel.BindingStatusPending
	}

	if s.podQuerier == nil {
		return false, channel.BindingStatusPending
	}

	initiatorInfo, err := s.podQuerier.GetPodInfo(ctx, initiatorPod)
	if err != nil {
		return false, channel.BindingStatusPending
	}

	targetInfo, err := s.podQuerier.GetPodInfo(ctx, targetPod)
	if err != nil {
		return false, channel.BindingStatusPending
	}

	initiatorUserID, ok1 := initiatorInfo["user_id"]
	targetUserID, ok2 := targetInfo["user_id"]
	if ok1 && ok2 && initiatorUserID == targetUserID {
		return true, channel.BindingStatusActive
	}

	if policy == channel.BindingPolicySameProjectAuto {
		initiatorProjectID, ok1 := initiatorInfo["project_id"]
		targetProjectID, ok2 := targetInfo["project_id"]
		if ok1 && ok2 && initiatorProjectID == targetProjectID {
			return true, channel.BindingStatusActive
		}
	}

	return false, channel.BindingStatusPending
}
