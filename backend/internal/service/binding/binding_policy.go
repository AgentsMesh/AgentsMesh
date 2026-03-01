package binding

import (
	"context"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/channel"
)

// evaluatePolicy evaluates the binding policy to determine if auto-approve
func (s *Service) evaluatePolicy(ctx context.Context, initiatorPod, targetPod, policy string) (bool, string) {
	// If explicit policy is set, use it
	if policy == channel.BindingPolicyExplicitOnly {
		return false, channel.BindingStatusPending
	}

	// If no pod querier available, require explicit confirmation
	if s.podQuerier == nil {
		return false, channel.BindingStatusPending
	}

	// Get pod info
	initiatorInfo, err := s.podQuerier.GetPodInfo(ctx, initiatorPod)
	if err != nil {
		return false, channel.BindingStatusPending
	}

	targetInfo, err := s.podQuerier.GetPodInfo(ctx, targetPod)
	if err != nil {
		return false, channel.BindingStatusPending
	}

	// Same user - auto approve
	initiatorUserID, ok1 := initiatorInfo["user_id"]
	targetUserID, ok2 := targetInfo["user_id"]
	if ok1 && ok2 && initiatorUserID == targetUserID {
		return true, channel.BindingStatusActive
	}

	// Same project - check policy
	if policy == channel.BindingPolicySameProjectAuto {
		initiatorProjectID, ok1 := initiatorInfo["project_id"]
		targetProjectID, ok2 := targetInfo["project_id"]
		if ok1 && ok2 && initiatorProjectID == targetProjectID {
			return true, channel.BindingStatusActive
		}
	}

	// Default: require explicit confirmation
	return false, channel.BindingStatusPending
}
