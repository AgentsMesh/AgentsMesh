package agentpod

// ActiveStatuses returns the list of Pod statuses considered active.
// Use this for SQL IN clauses to avoid hardcoding status strings.
func ActiveStatuses() []string {
	return []string{StatusInitializing, StatusRunning, StatusPaused, StatusDisconnected}
}

// TerminalStatuses returns the list of Pod statuses considered terminal (non-recoverable).
// Does NOT include StatusCompleted (graceful completion is not "terminal").
func TerminalStatuses() []string {
	return []string{StatusTerminated, StatusOrphaned, StatusError}
}

// IsPodStatusActive returns true if the given Pod status string represents an active state.
// Use this instead of comparing against individual status constants when you don't have a Pod instance.
func IsPodStatusActive(status string) bool {
	return status == StatusRunning ||
		status == StatusInitializing ||
		status == StatusPaused ||
		status == StatusDisconnected
}

// IsPodStatusTerminal returns true if the given Pod status string represents a terminal (non-recoverable) state.
// Note: StatusCompleted is NOT terminal — it represents graceful completion.
// Use IsPodStatusFinished for "done in any way" checks.
func IsPodStatusTerminal(status string) bool {
	return status == StatusTerminated ||
		status == StatusOrphaned ||
		status == StatusError
}

// IsPodStatusFinished returns true if the Pod execution is done (either gracefully completed or terminal).
func IsPodStatusFinished(status string) bool {
	return status == StatusCompleted || IsPodStatusTerminal(status)
}
