package billing

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// RecordUsage records usage for an organization
func (s *Service) RecordUsage(ctx context.Context, orgID int64, usageType string, quantity float64, metadata billing.UsageMetadata) error {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return err
	}

	record := &billing.UsageRecord{
		OrganizationID: orgID,
		UsageType:      usageType,
		Quantity:       quantity,
		PeriodStart:    sub.CurrentPeriodStart,
		PeriodEnd:      sub.CurrentPeriodEnd,
		Metadata:       metadata,
	}

	return s.repo.CreateUsageRecord(ctx, record)
}

// GetUsage returns usage for an organization in current period
func (s *Service) GetUsage(ctx context.Context, orgID int64, usageType string) (float64, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return 0, err
	}

	return s.repo.SumUsageByPeriod(ctx, orgID, usageType, sub.CurrentPeriodStart, sub.CurrentPeriodEnd)
}

// GetUsageHistory returns usage history for an organization
func (s *Service) GetUsageHistory(ctx context.Context, orgID int64, usageType string, months int) ([]*billing.UsageRecord, error) {
	since := time.Now().AddDate(0, -months, 0)
	return s.repo.ListUsageHistory(ctx, orgID, usageType, since)
}
