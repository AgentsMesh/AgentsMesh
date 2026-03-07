package loop

import (
	"context"
	"log/slog"
	"sync"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/anthropics/agentsmesh/backend/internal/service/instance"
	"github.com/robfig/cron/v3"
)

// LoopScheduler handles cron-based loop triggering and timeout detection.
// Start() begins two periodic goroutines:
//   - Cron trigger check every 30 seconds
//   - Timeout detection every 60 seconds
//
// Org-scoped: Uses LocalOrgProvider to only process loops belonging to orgs
// whose Runners are connected to this server instance.
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

// NewLoopScheduler creates a new LoopScheduler.
//
// orgProvider scopes all cron/timeout queries to the local instance's orgs.
// If orgProvider is nil, all orgs are processed (single-instance mode).
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

// getOrgIDs returns the local org IDs from the provider, or nil if no provider is set.
func (s *LoopScheduler) getOrgIDs() []int64 {
	if s.orgProvider == nil {
		return nil
	}
	return s.orgProvider.GetLocalOrgIDs()
}

// Start begins the periodic cron check and timeout detection goroutines.
func (s *LoopScheduler) Start() {
	// Initialize next_run_at for any cron loops that need it
	if err := s.InitializeNextRunTimes(context.Background()); err != nil {
		s.logger.Error("failed to initialize next run times", "error", err)
	}

	s.wg.Add(2)

	// Cron trigger check (every 30 seconds)
	go s.safeLoop("cron_trigger", s.runCronLoop)

	// Timeout detection + orphan cleanup (every 60 seconds)
	go s.safeLoop("timeout_detection", s.runTimeoutLoop)

	s.logger.Info("loop scheduler started (cron check: 30s, timeout check: 60s)")
}

// Stop gracefully stops the scheduler and waits for goroutines to exit.
// Safe to call multiple times.
func (s *LoopScheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.wg.Wait()
		s.logger.Info("loop scheduler stopped")
	})
}

// safeLoop runs fn in an infinite recovery loop. If fn panics, it logs and restarts
// after a 5-second cooldown. Stops when stopCh is closed.
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
		// fn returned normally (stopCh closed) or panicked — check if we should stop
		select {
		case <-s.stopCh:
			return
		default:
			// Panic recovery path — cooldown before restart to avoid tight panic loops
			time.Sleep(5 * time.Second)
		}
	}
}

// runCronLoop runs the cron trigger check ticker loop.
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

// runTimeoutLoop runs the timeout detection and orphan cleanup ticker loop.
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
			if err := s.orchestrator.CleanupOrphanPendingRuns(context.Background(), s.getOrgIDs()); err != nil {
				s.logger.Error("orphan cleanup failed", "error", err)
			}
		}
	}
}

// CheckAndTriggerCronLoops checks for loops with cron scheduling that are due and triggers them.
// Uses FOR UPDATE SKIP LOCKED within per-loop transactions for multi-instance safety:
// each loop is claimed atomically so only one instance processes it.
// Queries are scoped to local orgs via LocalOrgProvider.
func (s *LoopScheduler) CheckAndTriggerCronLoops(ctx context.Context) error {
	orgIDs := s.getOrgIDs()

	// Get candidate loops (lightweight query, no lock, org-scoped)
	dueLoops, err := s.loopService.GetDueCronLoops(ctx, orgIDs)
	if err != nil {
		s.logger.Error("failed to get due cron loops", "error", err)
		return err
	}

	if len(dueLoops) == 0 {
		return nil
	}

	s.logger.Info("found due cron loops", "count", len(dueLoops))

	for _, loop := range dueLoops {
		// Calculate next_run_at before claiming so we can advance it atomically
		var nextRunAt *time.Time
		if loop.CronExpression != nil {
			var calcErr error
			nextRunAt, calcErr = s.CalculateNextRun(*loop.CronExpression)
			if calcErr != nil {
				s.logger.Error("invalid cron expression, skipping loop",
					"loop_id", loop.ID, "cron", *loop.CronExpression, "error", calcErr)
				continue
			}
		}

		// Try to claim this loop with a short transaction
		claimed, err := s.loopService.ClaimCronLoop(ctx, loop.ID, nextRunAt)
		if err != nil {
			s.logger.Error("failed to claim cron loop", "loop_id", loop.ID, "error", err)
			continue
		}
		if !claimed {
			continue // Another instance claimed it
		}

		// Trigger run outside transaction (lock already released, next_run_at already advanced)
		result, err := s.orchestrator.TriggerRun(ctx, &TriggerRunRequest{
			LoopID:        loop.ID,
			TriggerType:   loopDomain.RunTriggerCron,
			TriggerSource: "cron",
		})
		if err != nil {
			s.logger.Error("failed to trigger cron loop", "loop_id", loop.ID, "error", err)
			continue
		}

		// Start the run asynchronously (Pod creation + Autopilot setup)
		// Uses the loop creator as the acting user for cron-triggered runs.
		if !result.Skipped && result.Run != nil && result.Loop != nil {
			go s.orchestrator.StartRun(context.Background(), result.Loop, result.Run, result.Loop.CreatedByID)
		}
	}

	return nil
}

// CalculateNextRun calculates the next execution time from a cron expression
func (s *LoopScheduler) CalculateNextRun(cronExpr string) (*time.Time, error) {
	schedule, err := s.cronParser.Parse(cronExpr)
	if err != nil {
		return nil, err
	}
	next := schedule.Next(time.Now())
	return &next, nil
}

// InitializeNextRunTimes sets initial next_run_at for all enabled cron loops that don't have one.
// Scoped to local orgs via LocalOrgProvider.
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
