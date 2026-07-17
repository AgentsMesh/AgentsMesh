package main

import (
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/tasks"
)

// At most each table's TTL (15 min pending auths, 10 min reactivation tokens),
// so a dead row's residency is bounded by the interval, not by a multiple of its
// own lifetime. Also caps each purge's query budget: tasks.Scheduler derives the
// run context from Interval*2.
const registrationGCInterval = 10 * time.Minute

// Both tables are self-expiring but nothing ever drained them, so they grew
// without bound and — while runner_pending_auths still had FKs — wedged runner
// and organization deletion.
func startRegistrationGC(services *serviceContainer, logger *slog.Logger) *tasks.Scheduler {
	scheduler := tasks.NewScheduler(logger)

	for _, task := range []*tasks.Task{
		{
			Name:       "pending_auth_purge",
			Interval:   registrationGCInterval,
			Func:       services.runner.CleanupExpiredPendingAuths,
			RunOnStart: true,
		},
		{
			Name:       "reactivation_token_purge",
			Interval:   registrationGCInterval,
			Func:       services.runner.CleanupExpiredReactivationTokens,
			RunOnStart: true,
		},
	} {
		if err := scheduler.Register(task); err != nil {
			logger.Error("failed to register registration GC task", "task", task.Name, "error", err)
			return nil
		}
	}

	scheduler.Start()
	return scheduler
}
