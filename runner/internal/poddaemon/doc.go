// Package poddaemon provides PTY session persistence ("daemonization") for Pods.
//
// It spawns a helper daemon process that owns the PTY file descriptor, so the
// child process (e.g., Claude Code, Aider) survives Runner restarts. After a
// Runner restart, [PodDaemonManager.RecoverSessions] rediscovers surviving
// daemons and re-attaches to them.
//
// Communication between the Runner and each daemon uses a Unix domain socket
// for IPC (command exchange, PTY resize, graceful shutdown, etc.).
//
// The name "poddaemon" refers to the daemon process that persists a single
// Pod's PTY session — it is NOT a daemon that manages Pods.
package poddaemon
