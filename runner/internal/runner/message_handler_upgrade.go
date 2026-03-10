package runner

import (
	"context"
	"fmt"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// OnUpgradeRunner handles remote upgrade command from server.
func (h *RunnerMessageHandler) OnUpgradeRunner(cmd *runnerv1.UpgradeRunnerCommand) error {
	log := logger.Runner()
	log.Info("Received upgrade command",
		"request_id", cmd.RequestId,
		"target_version", cmd.TargetVersion,
		"force", cmd.Force,
	)

	u := h.runner.GetUpdater()
	if u == nil {
		h.sendUpgradeStatus(cmd.RequestId, cmd.TargetVersion, "failed", 0, "", "updater not configured", nil)
		return fmt.Errorf("updater not configured")
	}

	// Discard if another upgrade is already in progress
	if !h.runner.TryStartUpgrade() {
		log.Info("Upgrade already in progress, discarding command", "request_id", cmd.RequestId)
		return nil
	}

	// Pod count check: refuse if pods are running and force is not set
	podCount := h.runner.GetActivePodCount()
	if !cmd.Force && podCount > 0 {
		h.runner.FinishUpgrade()
		msg := fmt.Sprintf("cannot upgrade: %d active pod(s), use force to override", podCount)
		h.sendUpgradeStatus(cmd.RequestId, cmd.TargetVersion, "failed", 0, msg, msg, u)
		return fmt.Errorf("%s", msg)
	}

	// Enter draining mode to prevent new pods
	h.runner.SetDraining(true)

	// Run upgrade asynchronously (already in goroutine from grpc_handler)
	h.runUpgrade(cmd)
	return nil
}

// runUpgrade executes the upgrade process and reports status at each phase.
func (h *RunnerMessageHandler) runUpgrade(cmd *runnerv1.UpgradeRunnerCommand) {
	log := logger.Runner()
	u := h.runner.GetUpdater()
	tv := cmd.TargetVersion

	// Centralized cleanup: always restore draining and upgrade flags on exit.
	// Set to false only when restart succeeds (process will exit).
	restoreDraining := true
	defer func() {
		if restoreDraining {
			h.runner.SetDraining(false)
		}
		h.runner.FinishUpgrade()
	}()

	// Check if runner is shutting down before starting
	runCtx := h.runner.GetRunContext()
	if runCtx.Err() != nil {
		h.sendUpgradeFailure(cmd.RequestId, tv, "runner is shutting down", u)
		return
	}

	ctx, cancel := context.WithTimeout(runCtx, 5*time.Minute)
	defer cancel()

	// Phase: checking
	h.sendUpgradeStatus(cmd.RequestId, tv, "checking", 0, "Checking for updates...", "", u)

	if tv == "" {
		// Update to latest version
		h.sendUpgradeStatus(cmd.RequestId, tv, "downloading", 0, "Downloading latest version...", "", u)

		newVersion, err := u.UpdateNow(ctx, nil)
		if err != nil {
			h.sendUpgradeFailure(cmd.RequestId, tv, fmt.Sprintf("update failed: %v", err), u)
			return
		}
		if newVersion == "" {
			h.sendUpgradeStatus(cmd.RequestId, tv, "completed", 100, "Already up to date", "", u)
			return
		}

		log.Info("Update downloaded and applied", "new_version", newVersion)
	} else {
		// Update to specific version
		h.sendUpgradeStatus(cmd.RequestId, tv, "downloading", 0,
			fmt.Sprintf("Downloading version %s...", tv), "", u)

		if err := u.UpdateToVersion(ctx, tv, nil); err != nil {
			h.sendUpgradeFailure(cmd.RequestId, tv, fmt.Sprintf("update to %s failed: %v", tv, err), u)
			return
		}

		log.Info("Update downloaded and applied", "target_version", tv)
	}

	// Phase: applying
	h.sendUpgradeStatus(cmd.RequestId, tv, "applying", 90, "Update applied, preparing restart...", "", u)

	// Phase: restarting
	h.sendUpgradeStatus(cmd.RequestId, tv, "restarting", 95, "Restarting service...", "", u)

	restartFn := h.runner.GetRestartFunc()
	if restartFn != nil {
		if _, err := restartFn(); err != nil {
			log.Error("Restart after upgrade failed", "error", err)
			h.sendUpgradeStatus(cmd.RequestId, tv, "completed", 100,
				"Update applied but restart failed, manual restart required", "", u)
			return
		}
		// Restart succeeded — process will exit and service manager restarts it.
		// Do NOT restore draining; the new process will reconnect.
		restoreDraining = false
	} else {
		log.Warn("No restart function configured, update applied but restart required manually")
		h.sendUpgradeStatus(cmd.RequestId, tv, "completed", 100,
			"Update applied, manual restart required", "", u)
	}
}

// sendUpgradeFailure reports an upgrade failure via status event.
func (h *RunnerMessageHandler) sendUpgradeFailure(requestID, targetVersion, errMsg string, u interface{ CurrentVersion() string }) {
	logger.Runner().Error("Upgrade failed", "request_id", requestID, "error", errMsg)
	h.sendUpgradeStatus(requestID, targetVersion, "failed", 0, "", errMsg, u)
}

// sendUpgradeStatus sends an upgrade status event to the server.
func (h *RunnerMessageHandler) sendUpgradeStatus(requestID, targetVersion, phase string, progress int32, message, errMsg string, u interface{ CurrentVersion() string }) {
	currentVersion := ""
	if u != nil {
		currentVersion = u.CurrentVersion()
	}

	event := &runnerv1.UpgradeStatusEvent{
		RequestId:      requestID,
		Phase:          phase,
		Progress:       progress,
		Message:        message,
		Error:          errMsg,
		CurrentVersion: currentVersion,
		TargetVersion:  targetVersion,
	}

	if err := h.conn.SendUpgradeStatus(event); err != nil {
		logger.Runner().Error("Failed to send upgrade status",
			"request_id", requestID,
			"phase", phase,
			"error", err,
		)
	}
}
