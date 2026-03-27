package runner

import "time"

// ==================== Terminate Command Backoff ====================

// isTerminateCooldown checks whether a terminate command for the given podKey
// is within the cooldown period.
func (pc *PodCoordinator) isTerminateCooldown(podKey string) bool {
	pc.terminateCacheMu.Lock()
	defer pc.terminateCacheMu.Unlock()
	if lastSent, ok := pc.terminateSentCache[podKey]; ok {
		return time.Since(lastSent) < terminateCooldown
	}
	return false
}

// recordTerminateSent records that a terminate command was sent.
// Also performs lazy cleanup of expired entries.
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

// ==================== Pod Miss Counter ====================

// incrementMissCount increments the consecutive heartbeat miss count for a pod.
func (pc *PodCoordinator) incrementMissCount(podKey string, runnerID int64) int {
	pc.podMissMu.Lock()
	defer pc.podMissMu.Unlock()
	pc.podMissCount[podKey]++
	pc.podMissOwner[podKey] = runnerID
	return pc.podMissCount[podKey]
}

// clearMissCount removes the miss counter for a specific pod.
func (pc *PodCoordinator) clearMissCount(podKey string) {
	pc.podMissMu.Lock()
	defer pc.podMissMu.Unlock()
	delete(pc.podMissCount, podKey)
	delete(pc.podMissOwner, podKey)
}

// clearMissCountsForRunner removes all miss counters for pods belonging to the given runner.
// Uses an in-memory reverse index to avoid TOCTOU races with DB queries.
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

// ==================== Init Report Counter ====================

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
