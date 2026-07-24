package runner

// cleanupFailedPodRuntime owns a runtime that never reached Running. Stop is
// still required because process startup may have succeeded partially before
// the caller received an error.
func (h *RunnerMessageHandler) cleanupFailedPodRuntime(pod *Pod) {
	if pod == nil {
		return
	}
	transition := pod.BeginRelayRuntimeTransition()
	runtime := pod.installedRuntime()
	pod.SetStatus(PodStatusFailed)
	if runtime.IO != nil {
		runtime.IO.Stop()
		runtime.IO.Teardown()
	}
	pod.TeardownRelayTransports(h.runner.GetLocalRelayServer())
	pod.clearRuntimeDuringTransition(transition)
}
