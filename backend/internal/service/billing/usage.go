package billing

import (
	"context"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
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

	return s.db.WithContext(ctx).Create(record).Error
}

// GetUsage returns usage for an organization in current period
func (s *Service) GetUsage(ctx context.Context, orgID int64, usageType string) (float64, error) {
	sub, err := s.GetSubscription(ctx, orgID)
	if err != nil {
		return 0, err
	}

	var total float64
	if err := s.db.WithContext(ctx).Model(&billing.UsageRecord{}).
		Where("organization_id = ? AND usage_type = ? AND period_start >= ? AND period_end <= ?",
			orgID, usageType, sub.CurrentPeriodStart, sub.CurrentPeriodEnd).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

// GetUsageHistory returns usage history for an organization
func (s *Service) GetUsageHistory(ctx context.Context, orgID int64, usageType string, months int) ([]*billing.UsageRecord, error) {
	since := time.Now().AddDate(0, -months, 0)

	var records []*billing.UsageRecord
	query := s.db.WithContext(ctx).Where("organization_id = ? AND period_start >= ?", orgID, since)

	if usageType != "" {
		query = query.Where("usage_type = ?", usageType)
	}

	if err := query.Order("period_start DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}
