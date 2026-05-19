package loop

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/service/instance"
	"github.com/robfig/cron/v3"
)

type LoopScheduler struct {
	loopService  *LoopService
	orchestrator *LoopOrchestrator
	orgProvider  instance.LocalOrgProvider
	logger       *slog.Logger
	cronParser   cron.Parser
	stopCh       chan struct{}
	stopOnce     sync.Once
	wg           sync.WaitGroup
}

func NewLoopScheduler(
	loopService *LoopService,
	orchestrator *LoopOrchestrator,
	orgProvider instance.LocalOrgProvider,
	logger *slog.Logger,
) *LoopScheduler {
	return &LoopScheduler{
		loopService:  loopService,
		orchestrator: orchestrator,
		orgProvider:  orgProvider,
		logger:       logger.With("component", "loop_scheduler"),
		cronParser:   cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
		stopCh:       make(chan struct{}),
	}
}

func (s *LoopScheduler) getOrgIDs() []int64 {
	if s.orgProvider == nil {
		return nil
	}
	return s.orgProvider.GetLocalOrgIDs()
}

func (s *LoopScheduler) Start() {
	if err := s.InitializeNextRunTimes(context.Background()); err != nil {
		s.logger.Error("failed to initialize next run times", "error", err)
	}

	s.wg.Add(2)

	go s.safeLoop("cron_trigger", s.runCronLoop)
	go s.safeLoop("timeout_detection", s.runTimeoutLoop)

	s.logger.Info("loop scheduler started (cron check: 30s, timeout check: 60s)")
}

func (s *LoopScheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.wg.Wait()
		s.logger.Info("loop scheduler stopped")
	})
}

func (s *LoopScheduler) safeLoop(name string, fn func()) {
	defer s.wg.Done()
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("panic in scheduler goroutine, restarting after cooldown",
						"goroutine", name, "panic", r)
				}
			}()
			fn()
		}()
		select {
		case <-s.stopCh:
			return
		default:
			time.Sleep(5 * time.Second)
		}
	}
}

func (s *LoopScheduler) runCronLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			if err := s.CheckAndTriggerCronLoops(context.Background()); err != nil {
				s.logger.Error("cron loop check failed", "error", err)
			}
		}
	}
}

func (s *LoopScheduler) runTimeoutLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			if err := s.orchestrator.CheckTimeoutRuns(context.Background(), s.getOrgIDs()); err != nil {
				s.logger.Error("timeout check failed", "error", err)
			}
			if err := s.orchestrator.CheckApprovalTimeouts(context.Background(), s.getOrgIDs()); err != nil {
				s.logger.Error("approval timeout check failed", "error", err)
			}
			if err := s.orchestrator.CheckIdleLoopPods(context.Background(), s.getOrgIDs()); err != nil {
				s.logger.Error("idle loop pod check failed", "error", err)
			}
			if err := s.orchestrator.CleanupOrphanPendingRuns(context.Background(), s.getOrgIDs()); err != nil {
				s.logger.Error("orphan cleanup failed", "error", err)
			}
		}
	}
}

func (s *LoopScheduler) CalculateNextRun(cronExpr string) (*time.Time, error) {
	schedule, err := s.cronParser.Parse(cronExpr)
	if err != nil {
		return nil, err
	}
	next := schedule.Next(time.Now())
	return &next, nil
}

func (s *LoopScheduler) InitializeNextRunTimes(ctx context.Context) error {
	orgIDs := s.getOrgIDs()

	loops, err := s.loopService.FindLoopsNeedingNextRun(ctx, orgIDs)
	if err != nil {
		return err
	}

	for _, loop := range loops {
		if loop.CronExpression != nil {
			nextRunAt, err := s.CalculateNextRun(*loop.CronExpression)
			if err != nil {
				s.logger.Error("invalid cron expression", "loop_id", loop.ID, "error", err)
				continue
			}
			if err := s.loopService.UpdateNextRunAt(ctx, loop.ID, nextRunAt); err != nil {
				s.logger.Error("failed to set initial next_run_at", "error", err, "loop_id", loop.ID)
			}
		}
	}

	return nil
}
