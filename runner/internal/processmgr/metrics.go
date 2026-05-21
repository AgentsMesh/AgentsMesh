package processmgr

import "sync/atomic"

// Metrics is a snapshot of lifetime counters and current registry size. It is
// returned by the /debug/processes endpoint so operators can answer
// "how many zombies did we silently reap?" without standing up Prometheus.
type Metrics struct {
	StartedTotal     map[string]int64 `json:"started_total"`
	ExitedTotal      map[string]int64 `json:"exited_total"`
	ZombieReapedTotal int64           `json:"zombie_reaped_total"`
	Alive             int             `json:"alive"`
}

// Counter values intentionally live in package state rather than on Manager so
// New(test-manager) and Init(prod-manager) share a single global view. Zombie
// reaps in particular are global by nature — the reaper observes the whole
// process, not a specific Manager.
var (
	startedByMode = newCounterMap()
	exitedByMode  = newCounterMap()
	zombieReaped  atomic.Int64
)

type counterMap struct {
	values [3]atomic.Int64 // indexed by Mode
}

func newCounterMap() *counterMap { return &counterMap{} }

func (c *counterMap) inc(m Mode) {
	if int(m) < len(c.values) {
		c.values[m].Add(1)
	}
}

func (c *counterMap) snapshot() map[string]int64 {
	out := make(map[string]int64, len(c.values))
	for i := range c.values {
		out[Mode(i).String()] = c.values[i].Load()
	}
	return out
}

func observeStart(mode Mode, _ string) { startedByMode.inc(mode) }

func observeExit(p Handle) { exitedByMode.inc(p.Mode()) }

func observeZombieReaped(n int) {
	if n > 0 {
		zombieReaped.Add(int64(n))
	}
}

func currentMetrics(alive int) Metrics {
	return Metrics{
		StartedTotal:      startedByMode.snapshot(),
		ExitedTotal:       exitedByMode.snapshot(),
		ZombieReapedTotal: zombieReaped.Load(),
		Alive:             alive,
	}
}

// resetMetricsForTest is used only by tests; production code has no reason to
// reset counters. Same-package _test.go files can call it directly; external
// packages must not depend on this internal API.
func resetMetricsForTest() {
	startedByMode = newCounterMap()
	exitedByMode = newCounterMap()
	zombieReaped.Store(0)
}
