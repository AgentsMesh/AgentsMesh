package runner

func (h *RunnerMessageHandler) preparePerpetualRestart(
	pod *Pod,
	exitCode int,
	stopIO bool,
) (RelayRuntimeTransition, bool) {
	if stopIO || !isCleanExit(exitCode) {
		return 0, false
	}

	pod.lifecycleMu.Lock()
	defer pod.lifecycleMu.Unlock()
	current, ok := h.podStore.Get(pod.PodKey)
	if !ok || current != pod || !pod.Perpetual {
		return 0, false
	}
	transition, started := pod.TryBeginRelayRuntimeTransition()
	if !started {
		return 0, true
	}
	pod.RestartCount++
	h.retirePerpetualRuntime(pod)
	if !pod.clearRuntimeDuringTransition(transition) {
		return 0, true
	}
	return transition, true
}

// retirePerpetualRuntime drains the completed runtime, disconnects its cloud
// publisher, and closes its local viewer generation. The caller must block relay
// candidates before invoking it and keep them blocked until replacement ends.
func (h *RunnerMessageHandler) retirePerpetualRuntime(pod *Pod) {
	runtime := pod.installedRuntime()
	pod.StopStateDetector()
	if runtime.IO != nil {
		runtime.IO.Teardown()
	}
	pod.TeardownRelayTransports(h.runner.GetLocalRelayServer())
	pod.SetPTYError("")
}
