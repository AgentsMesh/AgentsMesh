package job

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/email"
	"gorm.io/gorm"
)

type SubscriptionScheduler struct {
	renewJob    *SubscriptionRenewJob
	emailJob    *RenewalReminderJob
	stopCh      chan struct{}
	stopOnce    sync.Once
	wg          sync.WaitGroup
	logger      *slog.Logger
}

func NewSubscriptionScheduler(db *gorm.DB, appConfig *config.Config, emailSvc email.Service, logger *slog.Logger) *SubscriptionScheduler {
	return &SubscriptionScheduler{
		renewJob: NewSubscriptionRenewJob(db, appConfig, logger),
		emailJob: NewRenewalReminderJob(db, emailSvc, logger),
		stopCh:   make(chan struct{}),
		logger:   logger,
	}
}

func (s *SubscriptionScheduler) Start() {
	s.logger.Info("starting subscription scheduler")

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runInitialJobs()
	}()

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

func (s *SubscriptionScheduler) Stop() {
	s.logger.Info("stopping subscription scheduler")
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *SubscriptionScheduler) runInitialJobs() {
	ctx, span := otel.Tracer("agentsmesh-backend").Start(context.Background(), "job.initial")
	defer span.End()

	if err := s.renewJob.FreezeExpiredSubscriptions(ctx); err != nil {
		s.logger.Error("failed to run initial freeze check", "error", err)
	}
}

func (s *SubscriptionScheduler) runHourlyJobs() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			ctx, span := otel.Tracer("agentsmesh-backend").Start(context.Background(), "job.hourly")

			if err := s.renewJob.FreezeExpiredSubscriptions(ctx); err != nil {
				s.logger.Error("failed to freeze expired subscriptions", "error", err)
			}

			if err := s.renewJob.Run(ctx); err != nil {
				s.logger.Error("failed to process subscription renewals", "error", err)
			}
			span.End()
		}
	}
}

func (s *SubscriptionScheduler) runDailyJobs() {
	now := time.Now().UTC()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	initialDelay := nextMidnight.Sub(now)

	select {
	case <-s.stopCh:
		return
	case <-time.After(initialDelay):
	}

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

func (s *SubscriptionScheduler) runDailyJobsOnce() {
	ctx, span := otel.Tracer("agentsmesh-backend").Start(context.Background(), "job.daily")
	defer span.End()

	if err := s.emailJob.Run(ctx); err != nil {
		s.logger.Error("failed to send renewal reminders", "error", err)
	}
}
