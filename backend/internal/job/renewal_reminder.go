package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/billing"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/email"
	"gorm.io/gorm"
)

// RenewalReminderJob sends renewal reminder emails
type RenewalReminderJob struct {
	db       *gorm.DB
	emailSvc email.Service
	logger   *slog.Logger
}

// NewRenewalReminderJob creates a new renewal reminder job
func NewRenewalReminderJob(db *gorm.DB, emailSvc email.Service, logger *slog.Logger) *RenewalReminderJob {
	return &RenewalReminderJob{
		db:       db,
		emailSvc: emailSvc,
		logger:   logger,
	}
}

// SubscriptionWithOrg combines subscription with org info for email
type SubscriptionWithOrg struct {
	billing.Subscription
	OrgName    string `gorm:"column:org_name"`
	OrgSlug    string `gorm:"column:org_slug"`
	OwnerEmail string `gorm:"column:owner_email"`
	PlanName   string `gorm:"column:plan_name"`
}

// Run executes the renewal reminder job
// Sends reminders at 7 days, 3 days, and 1 day before expiry
func (j *RenewalReminderJob) Run(ctx context.Context) error {
	if j.emailSvc == nil {
		j.logger.Debug("email service not configured, skipping renewal reminders")
		return nil
	}

	j.logger.Info("starting renewal reminder job")

	reminderDays := []int{7, 3, 1}
	totalSent := 0

	for _, days := range reminderDays {
		sent, err := j.sendRemindersForDay(ctx, days)
		if err != nil {
			j.logger.Error("failed to send reminders", "days", days, "error", err)
			continue
		}
		totalSent += sent
	}

	j.logger.Info("renewal reminder job completed", "total_sent", totalSent)
	return nil
}

// sendRemindersForDay sends reminders for subscriptions expiring in N days
func (j *RenewalReminderJob) sendRemindersForDay(ctx context.Context, days int) (int, error) {
	targetDate := time.Now().UTC().AddDate(0, 0, days)
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.AddDate(0, 0, 1)

	// Find subscriptions expiring on this day
	// Includes both active and trialing subscriptions
	// - active/trialing status
	// - not set to auto-renew (manual renewal needed)
	// - not set to cancel at period end
	var subscriptions []SubscriptionWithOrg
	err := j.db.WithContext(ctx).
		Table("subscriptions s").
		Select("s.*, o.name as org_name, o.slug as org_slug, u.email as owner_email, sp.display_name as plan_name").
		Joins("JOIN organizations o ON o.id = s.organization_id").
		Joins("JOIN organization_members om ON om.organization_id = o.id AND om.role = 'owner'").
		Joins("JOIN users u ON u.id = om.user_id").
		Joins("JOIN subscription_plans sp ON sp.id = s.plan_id").
		Where("s.status IN ?", []string{billing.SubscriptionStatusActive, billing.SubscriptionStatusTrialing}).
		Where("s.auto_renew = ?", false).
		Where("s.cancel_at_period_end = ?", false).
		Where("s.current_period_end >= ?", startOfDay).
		Where("s.current_period_end < ?", endOfDay).
		Find(&subscriptions).Error
	if err != nil {
		return 0, fmt.Errorf("failed to find subscriptions: %w", err)
	}

	sent := 0
	for _, sub := range subscriptions {
		if err := j.sendReminderEmail(ctx, &sub, days); err != nil {
			j.logger.Error("failed to send reminder email",
				"subscription_id", sub.ID,
				"email", sub.OwnerEmail,
				"status", sub.Status,
				"error", err,
			)
			continue
		}
		sent++
	}

	if sent > 0 {
		j.logger.Info("sent renewal reminders", "days", days, "count", sent)
	}

	return sent, nil
}

// sendReminderEmail sends a renewal reminder email
func (j *RenewalReminderJob) sendReminderEmail(ctx context.Context, sub *SubscriptionWithOrg, days int) error {
	// Use SendRenewalReminder if available, otherwise log and skip
	reminderSvc, ok := j.emailSvc.(email.RenewalReminderSender)
	if !ok {
		j.logger.Debug("email service does not support renewal reminders", "email", sub.OwnerEmail)
		return nil
	}

	return reminderSvc.SendRenewalReminder(ctx, sub.OwnerEmail, sub.OrgName, sub.PlanName, sub.CurrentPeriodEnd, days, sub.OrgSlug)
}
