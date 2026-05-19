package runner

import "time"

func (pc *PodCoordinator) isTerminateCooldown(podKey string) bool {
	pc.terminateCacheMu.Lock()
	defer pc.terminateCacheMu.Unlock()
	if lastSent, ok := pc.terminateSentCache[podKey]; ok {
		return time.Since(lastSent) < terminateCooldown
	}
	return false
}

func (pc *PodCoordinator) recordTerminateSent(podKey string) {
	pc.terminateCacheMu.Lock()
	defer pc.terminateCacheMu.Unlock()
	now := time.Now()
	pc.terminateSentCache[podKey] = now

	for key, t := range pc.terminateSentCache {
		if now.Sub(t) > terminateCacheCleanup {
			delete(pc.terminateSentCache, key)
		}
	}
}

func (pc *PodCoordinator) incrementMissCount(podKey string, runnerID int64) int {
	pc.podMissMu.Lock()
	defer pc.podMissMu.Unlock()
	pc.podMissCount[podKey]++
	pc.podMissOwner[podKey] = runnerID
	return pc.podMissCount[podKey]
}

func (pc *PodCoordinator) clearMissCount(podKey string) {
	pc.podMissMu.Lock()
	defer pc.podMissMu.Unlock()
	delete(pc.podMissCount, podKey)
	delete(pc.podMissOwner, podKey)
}

func (pc *PodCoordinator) clearMissCountsForRunner(runnerID int64) {
	pc.podMissMu.Lock()
	defer pc.podMissMu.Unlock()
	for podKey, ownerID := range pc.podMissOwner {
		if ownerID == runnerID {
			delete(pc.podMissCount, podKey)
			delete(pc.podMissOwner, podKey)
		}
	}
}

func (pc *PodCoordinator) incrementInitReportCount(podKey string) int {
	pc.initReportCountMu.Lock()
	defer pc.initReportCountMu.Unlock()
	pc.initReportCount[podKey]++
	return pc.initReportCount[podKey]
}

func (pc *PodCoordinator) clearInitReportCount(podKey string) {
	pc.initReportCountMu.Lock()
	defer pc.initReportCountMu.Unlock()
	delete(pc.initReportCount, podKey)
}
