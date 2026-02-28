package lifecycle

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// ActivityMonitor provides connection activity information for health checking.
type ActivityMonitor interface {
	// LastActivityTime returns the last time the connection sent or received data.
	// Returns zero time if no activity has occurred.
	LastActivityTime() time.Time
}

// WatchdogConfig contains configuration for the WatchdogService.
type WatchdogConfig struct {
	// ConnMonitor provides connection activity info (optional).
	ConnMonitor ActivityMonitor

	// Interval between health checks (default: 15s).
	Interval time.Duration

	// MaxGoroutines is the goroutine count threshold (default: 1000).
	MaxGoroutines int

	// MaxMemoryMB is the memory usage threshold in megabytes (default: 2048).
	MaxMemoryMB int

	// MaxFailCount is the number of consecutive failures before reporting unhealthy (default: 3).
	MaxFailCount int

	// ConnectionIdleTimeout is the max time without connection activity (default: 5m).
	ConnectionIdleTimeout time.Duration
}

// WatchdogService monitors Runner health and reports to systemd watchdog (on Linux).
// It implements suture.Service and is managed by the Supervisor tree.
// On consecutive health check failures, it returns an error to trigger Supervisor restart.
type WatchdogService struct {
	cfg       WatchdogConfig
	failCount int
}

// NewWatchdogService creates a new WatchdogService with the given configuration.
func NewWatchdogService(cfg WatchdogConfig) *WatchdogService {
	// Apply defaults
	if cfg.Interval == 0 {
		cfg.Interval = 15 * time.Second
	}
	if cfg.MaxGoroutines == 0 {
		cfg.MaxGoroutines = 1000
	}
	if cfg.MaxMemoryMB == 0 {
		cfg.MaxMemoryMB = 2048
	}
	if cfg.MaxFailCount == 0 {
		cfg.MaxFailCount = 3
	}
	if cfg.ConnectionIdleTimeout == 0 {
		cfg.ConnectionIdleTimeout = 5 * time.Minute
	}

	return &WatchdogService{cfg: cfg}
}

// Serve implements suture.Service. It periodically runs health checks.
func (w *WatchdogService) Serve(ctx context.Context) error {
	log := logger.Runner()
	log.Info("WatchdogService starting",
		"interval", w.cfg.Interval,
		"max_goroutines", w.cfg.MaxGoroutines,
		"max_memory_mb", w.cfg.MaxMemoryMB,
	)

	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.runChecks(); err != nil {
				w.failCount++
				log.Warn("Health check failed",
					"error", err,
					"consecutive_failures", w.failCount,
					"max_failures", w.cfg.MaxFailCount,
				)
				if w.failCount >= w.cfg.MaxFailCount {
					return fmt.Errorf("watchdog: unhealthy after %d consecutive failures: %w", w.failCount, err)
				}
			} else {
				if w.failCount > 0 {
					log.Info("Health check recovered", "previous_failures", w.failCount)
				}
				w.failCount = 0
				// Notify systemd watchdog (no-op on non-Linux platforms)
				notifySystemHealthy()
			}
		}
	}
}

// String returns the service name for logging.
func (w *WatchdogService) String() string {
	return "WatchdogService"
}

// runChecks performs all health checks and returns an error if any fail.
func (w *WatchdogService) runChecks() error {
	// Check goroutine count
	numGoroutines := runtime.NumGoroutine()
	if numGoroutines > w.cfg.MaxGoroutines {
		return fmt.Errorf("goroutine count %d exceeds threshold %d (possible leak)", numGoroutines, w.cfg.MaxGoroutines)
	}

	// Check memory usage using HeapInuse instead of Alloc.
	// HeapInuse reflects actual heap memory held by the runtime and is stable
	// across GC cycles, whereas Alloc fluctuates significantly and causes
	// false positives/negatives.
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	heapInuseMB := int(memStats.HeapInuse / 1024 / 1024)
	if heapInuseMB > w.cfg.MaxMemoryMB {
		return fmt.Errorf("heap memory %dMB exceeds threshold %dMB (possible leak)", heapInuseMB, w.cfg.MaxMemoryMB)
	}

	// Check connection activity (if monitor is available)
	if w.cfg.ConnMonitor != nil {
		lastActivity := w.cfg.ConnMonitor.LastActivityTime()
		if !lastActivity.IsZero() {
			idle := time.Since(lastActivity)
			if idle > w.cfg.ConnectionIdleTimeout {
				return fmt.Errorf("connection idle for %v (threshold %v, possible stuck)", idle, w.cfg.ConnectionIdleTimeout)
			}
		}
	}

	return nil
}
