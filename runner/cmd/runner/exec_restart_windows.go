//go:build windows

package main

import (
	"github.com/anthropics/agentsmesh/runner/internal/service"
)

// execRestartFunc returns the platform restart function for Windows.
// Windows does not support Unix syscall.Exec (exec-replace), so we
// fall back to the service manager: in service mode it asks the SCM
// to restart; in interactive mode it logs a manual-restart hint.
// The execPath parameter is accepted for API consistency with Unix but
// unused — Windows restarts via the Service Control Manager.
func execRestartFunc(_ string) func() (int, error) {
	return service.RestartFunc()
}
