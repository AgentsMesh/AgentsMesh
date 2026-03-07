package job

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	"gorm.io/gorm"
)

// SubscriptionScheduler manages scheduled subscription-related jobs
type SubscriptionScheduler struct {
	renewJob    *SubscriptionRenewJob
	emailJob    *RenewalReminderJob
	stopCh      chan struct{}
	stopOnce    sync.Once
	wg          sync.WaitGroup
	logger      *slog.Logger
}

// NewSubscriptionScheduler creates a new subscription scheduler
// appConfig is needed for URL derivation in payment providers
func NewSubscriptionScheduler(db *gorm.DB, appConfig *config.Config, emailSvc email.Service, logger *slog.Logger) *SubscriptionScheduler {
	return &SubscriptionScheduler{
		renewJob: NewSubscriptionRenewJob(db, appConfig, logger),
		emailJob: NewRenewalReminderJob(db, emailSvc, logger),
		stopCh:   make(chan struct{}),
		logger:   logger,
	}
}

// Start begins the scheduled jobs
func (s *SubscriptionScheduler) Start() {
	s.logger.Info("starting subscription scheduler")

	// Run jobs immediately on startup
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runInitialJobs()
	}()

	// Start periodic job runners
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runHourlyJobs()
	}()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runDailyJobs()
	}()
}

// Stop stops all scheduled jobs and waits for goroutines to exit
func (s *SubscriptionScheduler) Stop() {
	s.logger.Info("stopping subscription scheduler")
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

// runInitialJobs runs jobs immediately on startup
func (s *SubscriptionScheduler) runInitialJobs() {
	ctx := context.Background()

	// Run freeze check on startup
	if err := s.renewJob.FreezeExpiredSubscriptions(ctx); err != nil {
		s.logger.Error("failed to run initial freeze check", "error", err)
	}
}

// runHourlyJobs runs jobs every hour
func (s *SubscriptionScheduler) runHourlyJobs() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			ctx := context.Background()

			// Freeze expired subscriptions
			if err := s.renewJob.FreezeExpiredSubscriptions(ctx); err != nil {
				s.logger.Error("failed to freeze expired subscriptions", "error", err)
			}

			// Process auto-renewals (for CN payments with agreement/contract)
			if err := s.renewJob.Run(ctx); err != nil {
				s.logger.Error("failed to process subscription renewals", "error", err)
			}
		}
	}
}

// runDailyJobs runs jobs once a day (at midnight UTC)
func (s *SubscriptionScheduler) runDailyJobs() {
	// Calculate time until next midnight UTC
	now := time.Now().UTC()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	initialDelay := nextMidnight.Sub(now)

	// Wait until midnight
	select {
	case <-s.stopCh:
		return
	case <-time.After(initialDelay):
	}

	// Run daily jobs
	s.runDailyJobsOnce()

	// Then run every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.runDailyJobsOnce()
		}
	}
}

// runDailyJobsOnce executes all daily jobs once
func (s *SubscriptionScheduler) runDailyJobsOnce() {
	ctx := context.Background()

	// Send renewal reminder emails
	if err := s.emailJob.Run(ctx); err != nil {
		s.logger.Error("failed to send renewal reminders", "error", err)
	}
}
