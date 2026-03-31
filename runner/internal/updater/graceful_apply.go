package updater

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/process"
)

// executeUpdate creates a backup, downloads and replaces the binary, and restarts.
func (g *GracefulUpdater) executeUpdate(ctx context.Context) error {
	log := logger.Updater()

	g.mu.RLock()
	info := g.pendingInfo
	g.mu.RUnlock()

	if info == nil {
		g.setState(StateIdle)
		return fmt.Errorf("no pending update to apply")
	}

	log.Info("Applying update", "from", info.CurrentVersion, "to", info.LatestVersion)
	g.setState(StateApplying)

	// Create backup for potential rollback
	backupPath, err := g.updater.CreateBackup()
	if err != nil {
		log.Warn("Failed to create backup (rollback unavailable)", "error", err)
		// Continue without backup - rollback won't be possible
	}

	// Update binary in-place via detector.UpdateBinary
	if err := g.updater.updateBinary(ctx, info.LatestVersion); err != nil {
		log.Error("Failed to apply update", "version", info.LatestVersion, "error", err)
		g.mu.Lock()
		g.pendingInfo = nil
		g.mu.Unlock()
		g.setState(StateIdle)
		return fmt.Errorf("failed to apply update: %w", err)
	}

	g.mu.Lock()
	g.pendingInfo = nil
	g.mu.Unlock()

	log.Info("Update applied successfully", "from", info.CurrentVersion, "to", info.LatestVersion)

	// Restart
	g.setState(StateRestarting)
	if g.restartFunc != nil {
		pid, err := g.restartFunc()
		if err != nil {
			// The binary on disk is already updated. If we can't restart
			// in-process (e.g., /proc/self/exe points to a deleted .old file),
			// exit so the service manager (systemd) restarts us with the new
			// binary. Rolling back here would leave the old version running
			// permanently since subsequent upgrade attempts also fail once
			// /proc/self/exe is stale.
			//
			// Exit with non-zero code: systemd's Restart=on-failure only
			// restarts on non-zero exit, signal death, or timeout.
			log.Error("Restart failed after successful update, exiting for service manager restart", "error", err)
			g.exitFunc(1)
			return nil // unreachable in production (os.Exit never returns); needed for tests
		}

		// Health check if configured
		if g.healthChecker != nil && pid > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), g.healthTimeout)
			defer cancel()

			if err := g.healthChecker(ctx, pid); err != nil {
				log.Error("Health check failed, attempting rollback", "pid", pid, "error", err)
				// Terminate the unhealthy new process
				if proc, findErr := os.FindProcess(pid); findErr == nil && proc != nil {
					_ = proc.Kill()
				}
				if rbErr := g.rollbackUpdate(backupPath); rbErr != nil {
					log.Error("Rollback also failed", "error", rbErr)
				}
				g.setState(StateIdle)
				return fmt.Errorf("health check failed: %w", err)
			}
			log.Info("Health check passed for new process", "pid", pid)
		}
	}

	return nil
}

// rollbackUpdate attempts to restore the previous version from backup.
func (g *GracefulUpdater) rollbackUpdate(backupPath string) error {
	log := logger.Updater()
	if backupPath == "" {
		log.Error("No backup available for rollback")
		return fmt.Errorf("no backup available for rollback")
	}
	log.Info("Rolling back to previous version", "backup", backupPath)
	if err := g.updater.Rollback(); err != nil {
		log.Error("Rollback failed", "error", err)
		return fmt.Errorf("rollback failed: %w", err)
	}
	log.Info("Successfully rolled back to previous version")
	return nil
}

// DefaultRestartFunc returns a restart function that re-executes the current binary.
//
// Deprecated: This function calls os.Executable() at restart time, which returns a
// stale path after a self-upgrade (the old binary is renamed to .old then deleted,
// but /proc/self/exe still references it). Use a restart function that receives
// the executable path resolved at startup instead. See execRestartFunc in cmd/runner.
func DefaultRestartFunc() RestartFunc {
	return func() (int, error) {
		execPath, err := os.Executable()
		if err != nil {
			return 0, fmt.Errorf("failed to get executable path: %w", err)
		}

		// Start new process
		cmd := exec.Command(execPath, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = os.Environ()

		if err := cmd.Start(); err != nil {
			return 0, fmt.Errorf("failed to start new process: %w", err)
		}

		logger.Updater().Info("New process started, current process should exit", "pid", cmd.Process.Pid)
		// Note: Caller is responsible for graceful shutdown after this returns
		// Do NOT call os.Exit() here as it prevents proper cleanup
		return cmd.Process.Pid, nil
	}
}

// DefaultHealthChecker returns a health checker that validates the new process is running.
// minRunTime: the minimum time the new process should run before being considered healthy.
func DefaultHealthChecker(minRunTime time.Duration) HealthChecker {
	return func(ctx context.Context, pid int) error {
		// Wait for the specified minimum run time
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(minRunTime):
		}

		// Check if the process is still running (cross-platform)
		return process.IsAlive(pid)
	}
}
