package billing

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// BillingOverview represents billing overview
type BillingOverview struct {
	Plan               *billing.SubscriptionPlan `json:"plan"`
	Status             string                    `json:"status"`
	BillingCycle       string                    `json:"billing_cycle"`
	CurrentPeriodStart time.Time                 `json:"current_period_start"`
	CurrentPeriodEnd   time.Time                 `json:"current_period_end"`
	CancelAtPeriodEnd  bool                      `json:"cancel_at_period_end"`
	Usage              UsageOverview             `json:"usage"`
}

// UsageOverview represents usage overview
type UsageOverview struct {
	PodMinutes         float64 `json:"pod_minutes"`
	IncludedPodMinutes float64 `json:"included_pod_minutes"`
	Users              int     `json:"users"`
	MaxUsers           int     `json:"max_users"`
	Runners            int     `json:"runners"`
	MaxRunners         int     `json:"max_runners"`
	ConcurrentPods     int     `json:"concurrent_pods"`
	MaxConcurrentPods  int     `json:"max_concurrent_pods"`
	Repositories       int     `json:"repositories"`
	MaxRepositories    int     `json:"max_repositories"`
}

// DeploymentInfo represents deployment information
type DeploymentInfo struct {
	DeploymentType     string   `json:"deployment_type"`
	AvailableProviders []string `json:"available_providers"`
}

// GetBillingOverview returns billing overview for an organization
func (s *Service) GetBillingOverview(ctx context.Context, orgID int64) (*BillingOverview, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return nil, err
	}

	plan := sub.Plan
	if plan == nil {
		plan, _ = s.GetPlanByID(ctx, sub.PlanID)
	}

	// Get current usage
	podMinutes, _ := s.GetUsage(ctx, orgID, billing.UsageTypePodMinutes)

	// Count resources
	userCount, _ := s.repo.CountOrgMembers(ctx, orgID)
	runnerCount, _ := s.repo.CountRunners(ctx, orgID)
	repoCount, _ := s.repo.CountRepositories(ctx, orgID)
	concurrentPodCount, _ := s.repo.CountActivePods(ctx, orgID)

	return &BillingOverview{
		Plan:               plan,
		Status:             sub.Status,
		BillingCycle:       sub.BillingCycle,
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		CancelAtPeriodEnd:  sub.CancelAtPeriodEnd,
		Usage: UsageOverview{
			PodMinutes:         podMinutes,
			IncludedPodMinutes: float64(plan.IncludedPodMinutes),
			Users:              int(userCount),
			MaxUsers:           plan.MaxUsers,
			Runners:            int(runnerCount),
			MaxRunners:         plan.MaxRunners,
			ConcurrentPods:     int(concurrentPodCount),
			MaxConcurrentPods:  plan.MaxConcurrentPods,
			Repositories:       int(repoCount),
			MaxRepositories:    plan.MaxRepositories,
		},
	}, nil
}

// GetDeploymentInfo returns deployment type and available payment providers
func (s *Service) GetDeploymentInfo() *DeploymentInfo {
	if s.paymentConfig == nil {
		return &DeploymentInfo{
			DeploymentType:     "global",
			AvailableProviders: []string{},
		}
	}

	return &DeploymentInfo{
		DeploymentType:     string(s.paymentConfig.DeploymentType),
		AvailableProviders: s.paymentConfig.GetAvailableProviders(),
	}
}
