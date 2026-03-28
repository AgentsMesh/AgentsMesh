package updater

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/process"
)

// executeUpdate creates a backup, downloads and replaces the binary, and restarts.
func (g *GracefulUpdater) executeUpdate(ctx context.Context) error {
	g.mu.RLock()
	info := g.pendingInfo
	g.mu.RUnlock()

	if info == nil {
		g.setState(StateIdle)
		return fmt.Errorf("no pending update to apply")
	}

	g.setState(StateApplying)

	// Create backup for potential rollback
	backupPath, err := g.updater.CreateBackup()
	if err != nil {
		slog.Warn("failed to create backup", "error", err)
		// Continue without backup - rollback won't be possible
	}

	// Update binary in-place via detector.UpdateBinary
	if err := g.updater.updateBinary(ctx, info.LatestVersion); err != nil {
		g.mu.Lock()
		g.pendingInfo = nil
		g.mu.Unlock()
		g.setState(StateIdle)
		return fmt.Errorf("failed to apply update: %w", err)
	}

	g.mu.Lock()
	g.pendingInfo = nil
	g.mu.Unlock()

	slog.Info("update applied successfully", "from", info.CurrentVersion, "to", info.LatestVersion)

	// Restart
	g.setState(StateRestarting)
	if g.restartFunc != nil {
		pid, err := g.restartFunc()
		if err != nil {
			slog.Error("restart failed, attempting rollback", "error", err)
			if rbErr := g.rollbackUpdate(backupPath); rbErr != nil {
				slog.Error("rollback also failed", "error", rbErr)
			}
			g.setState(StateIdle)
			return fmt.Errorf("restart failed: %w", err)
		}

		// Health check if configured
		if g.healthChecker != nil && pid > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), g.healthTimeout)
			defer cancel()

			if err := g.healthChecker(ctx, pid); err != nil {
				slog.Error("health check failed, attempting rollback", "error", err)
				// Terminate the unhealthy new process
				if proc, findErr := os.FindProcess(pid); findErr == nil && proc != nil {
					_ = proc.Kill()
				}
				if rbErr := g.rollbackUpdate(backupPath); rbErr != nil {
					slog.Error("rollback also failed", "error", rbErr)
				}
				g.setState(StateIdle)
				return fmt.Errorf("health check failed: %w", err)
			}
			slog.Info("health check passed for new process", "pid", pid)
		}
	}

	return nil
}

// rollbackUpdate attempts to restore the previous version from backup.
func (g *GracefulUpdater) rollbackUpdate(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("no backup available for rollback")
	}
	if err := g.updater.Rollback(); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}
	slog.Info("successfully rolled back to previous version")
	return nil
}

// DefaultRestartFunc returns a restart function that re-executes the current binary.
// Note: This function starts a new process and signals the caller to exit gracefully.
// The caller should handle process termination appropriately.
// Returns the PID of the new process for health checking.
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

		slog.Info("new process started, current process should exit", "pid", cmd.Process.Pid)
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
