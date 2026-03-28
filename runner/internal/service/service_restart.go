package service

import (
	"fmt"
	"os"
	"runtime"

	"github.com/kardianos/service"
)

// RestartForUpdate restarts the service after an update.
// This should be called after the binary has been replaced.
func RestartForUpdate() error {
	if !service.Interactive() {
		if runtime.GOOS == "darwin" {
			// macOS launchd: kardianos/service.Restart() uses launchctl unload/load.
			// When called from within the service process, unload kills the process
			// before load can execute, leaving the service unregistered in launchd.
			// Instead, exit and let KeepAlive=true restart us with the new binary.
			log.Info("Update applied, exiting for launchd KeepAlive restart")
			os.Exit(0)
		}

		// Linux/Windows: s.Restart() uses atomic service manager commands
		// (systemctl restart / sc.exe) that work correctly from within the process.
		prg := &Program{}
		s, err := service.New(prg, ServiceConfig())
		if err != nil {
			log.Error("Failed to create service for update restart", "error", err)
			return fmt.Errorf("failed to create service: %w", err)
		}
		err = s.Restart()
		if err != nil {
			log.Error("Failed to restart service for update", "error", err)
			return fmt.Errorf("failed to restart service: %w", err)
		}
		log.Info("Service restarted for update")
		return nil
	}

	// If running interactively, just log and exit
	log.Info("Update applied. Please restart the runner manually.")
	return nil
}

// RestartFunc returns a function that can be used to restart the service.
// This is designed to be passed to the graceful updater.
// Returns pid=0 because the service manager spawns the new process and we
// don't have its PID. GracefulUpdater.executeUpdate() only runs the
// health check when pid > 0, so this is safe.
func RestartFunc() func() (int, error) {
	return func() (int, error) {
		return 0, RestartForUpdate()
	}
}
