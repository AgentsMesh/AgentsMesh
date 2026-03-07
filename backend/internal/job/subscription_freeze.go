package job

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/billing"
)

// FreezeExpiredSubscriptions freezes subscriptions that have expired without renewal
// This includes both active subscriptions and trial subscriptions
// This should be called periodically (e.g., every hour)
func (j *SubscriptionRenewJob) FreezeExpiredSubscriptions(ctx context.Context) error {
	j.logger.Info("checking for expired subscriptions to freeze")

	now := time.Now()

	// Freeze expired active subscriptions
	// - status is active
	// - current_period_end has passed
	// - are not set to cancel (they would have been canceled already)
	activeResult := j.db.WithContext(ctx).
		Model(&billing.Subscription{}).
		Where("status = ?", billing.SubscriptionStatusActive).
		Where("current_period_end < ?", now).
		Where("cancel_at_period_end = ?", false).
		Updates(map[string]interface{}{
			"status":    billing.SubscriptionStatusFrozen,
			"frozen_at": now,
		})

	if activeResult.Error != nil {
		return fmt.Errorf("failed to freeze expired active subscriptions: %w", activeResult.Error)
	}

	if activeResult.RowsAffected > 0 {
		j.logger.Info("froze expired active subscriptions", "count", activeResult.RowsAffected)
	}

	// Freeze expired trial subscriptions
	// - status is trialing
	// - current_period_end has passed (trial ended)
	trialResult := j.db.WithContext(ctx).
		Model(&billing.Subscription{}).
		Where("status = ?", billing.SubscriptionStatusTrialing).
		Where("current_period_end < ?", now).
		Updates(map[string]interface{}{
			"status":    billing.SubscriptionStatusFrozen,
			"frozen_at": now,
		})

	if trialResult.Error != nil {
		return fmt.Errorf("failed to freeze expired trial subscriptions: %w", trialResult.Error)
	}

	if trialResult.RowsAffected > 0 {
		j.logger.Info("froze expired trial subscriptions", "count", trialResult.RowsAffected)
	}

	// Also update organization subscription_status for frozen subscriptions
	if activeResult.RowsAffected > 0 || trialResult.RowsAffected > 0 {
		if err := j.db.WithContext(ctx).Exec(`
			UPDATE organizations o
			SET subscription_status = 'frozen'
			FROM subscriptions s
			WHERE s.organization_id = o.id
			AND s.status = 'frozen'
			AND o.subscription_status != 'frozen'
		`).Error; err != nil {
			return fmt.Errorf("failed to sync organization subscription_status: %w", err)
		}
	}

	return nil
}

// SendRenewalReminders sends reminder emails for upcoming renewals
// This should be called daily
func (j *SubscriptionRenewJob) SendRenewalReminders(ctx context.Context) error {
	j.logger.Info("sending renewal reminder emails")

	// Find subscriptions expiring in 7 days, 3 days, or 1 day
	reminderDays := []int{7, 3, 1}

	for _, days := range reminderDays {
		targetDate := time.Now().AddDate(0, 0, days)
		startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := startOfDay.AddDate(0, 0, 1)

		var subscriptions []billing.Subscription
		err := j.db.WithContext(ctx).
			Where("status = ?", billing.SubscriptionStatusActive).
			Where("auto_renew = ?", false). // Only remind manual renewal users
			Where("current_period_end >= ?", startOfDay).
			Where("current_period_end < ?", endOfDay).
			Find(&subscriptions).Error
		if err != nil {
			j.logger.Error("failed to find subscriptions for reminder",
				"days", days,
				"error", err,
			)
			continue
		}

		for _, sub := range subscriptions {
			// TODO: Send email reminder
			// This would integrate with the email service
			j.logger.Info("would send renewal reminder",
				"subscription_id", sub.ID,
				"organization_id", sub.OrganizationID,
				"days_until_expiry", days,
			)
		}
	}

	return nil
}
