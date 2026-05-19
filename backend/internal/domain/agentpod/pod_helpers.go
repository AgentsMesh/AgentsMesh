package agentpod

func ActiveStatuses() []string {
	return []string{StatusInitializing, StatusRunning, StatusPaused, StatusDisconnected}
}

func TerminalStatuses() []string {
	return []string{StatusTerminated, StatusOrphaned, StatusError}
}

func IsPodStatusActive(status string) bool {
	return status == StatusRunning ||
		status == StatusInitializing ||
		status == StatusPaused ||
		status == StatusDisconnected
}

func IsPodStatusTerminal(status string) bool {
	return status == StatusTerminated ||
		status == StatusOrphaned ||
		status == StatusError
}

func IsPodStatusFinished(status string) bool {
	return status == StatusCompleted || IsPodStatusTerminal(status)
}
