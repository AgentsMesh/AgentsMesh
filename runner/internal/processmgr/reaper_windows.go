//go:build windows

package processmgr

// reapOrphans on Windows is a no-op: the OS automatically frees process
// kernel objects once every handle is closed, so there is no "zombie state"
// equivalent to Unix. The reaper goroutine still runs to keep the contract
// symmetric and to act as a heartbeat that the Manager is alive.
func reapOrphans() int { return 0 }
