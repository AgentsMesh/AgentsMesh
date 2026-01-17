package updater

import (
	"context"
	"log"
	"sync"
	"time"
)

// BackgroundChecker periodically checks for updates in the background.
type BackgroundChecker struct {
	updater       *Updater
	graceful      *GracefulUpdater
	interval      time.Duration
	initialDelay  time.Duration
	autoApply     bool
	onUpdate      func(info *UpdateInfo) // Called when a new update is found
	onError       func(err error)        // Called when check fails

	// State
	mu            sync.RWMutex
	running       bool
	lastCheck     time.Time
	lastError     error
	latestInfo    *UpdateInfo
	cancel        context.CancelFunc
}

// CheckerOption configures the BackgroundChecker.
type CheckerOption func(*BackgroundChecker)

// WithOnUpdate sets the callback for when an update is found.
func WithOnUpdate(cb func(info *UpdateInfo)) CheckerOption {
	return func(c *BackgroundChecker) {
		c.onUpdate = cb
	}
}

// WithOnError sets the callback for check errors.
func WithOnError(cb func(err error)) CheckerOption {
	return func(c *BackgroundChecker) {
		c.onError = cb
	}
}

// WithAutoApply controls whether to automatically apply updates.
func WithAutoApply(apply bool) CheckerOption {
	return func(c *BackgroundChecker) {
		c.autoApply = apply
	}
}

// WithInitialDelay sets the initial delay before the first check.
func WithInitialDelay(delay time.Duration) CheckerOption {
	return func(c *BackgroundChecker) {
		c.initialDelay = delay
	}
}

// NewBackgroundChecker creates a new background update checker.
func NewBackgroundChecker(updater *Updater, graceful *GracefulUpdater, interval time.Duration, opts ...CheckerOption) *BackgroundChecker {
	c := &BackgroundChecker{
		updater:      updater,
		graceful:     graceful,
		interval:     interval,
		initialDelay: 30 * time.Second, // Default initial delay
		autoApply:    true,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Start begins periodic update checks.
func (c *BackgroundChecker) Start(ctx context.Context) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true

	checkCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.mu.Unlock()

	go c.run(checkCtx)
}

// Stop stops the background checker.
func (c *BackgroundChecker) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	if c.cancel != nil {
		c.cancel()
	}
	c.running = false
}

// IsRunning returns true if the checker is running.
func (c *BackgroundChecker) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// LastCheck returns the time of the last check.
func (c *BackgroundChecker) LastCheck() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastCheck
}

// LastError returns the last error, if any.
func (c *BackgroundChecker) LastError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastError
}

// LatestInfo returns the latest update info.
func (c *BackgroundChecker) LatestInfo() *UpdateInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latestInfo
}

// CheckNow triggers an immediate check.
func (c *BackgroundChecker) CheckNow(ctx context.Context) (*UpdateInfo, error) {
	return c.doCheck(ctx)
}

func (c *BackgroundChecker) run(ctx context.Context) {
	// Do initial check after a short delay
	select {
	case <-ctx.Done():
		return
	case <-time.After(c.initialDelay):
	}

	c.doCheck(ctx)

	// Periodic checks
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.mu.Lock()
			c.running = false
			c.mu.Unlock()
			return
		case <-ticker.C:
			c.doCheck(ctx)
		}
	}
}

func (c *BackgroundChecker) doCheck(ctx context.Context) (*UpdateInfo, error) {
	c.mu.Lock()
	c.lastCheck = time.Now()
	c.mu.Unlock()

	log.Printf("[updater] Checking for updates...")

	info, err := c.updater.CheckForUpdate(ctx)
	if err != nil {
		c.mu.Lock()
		c.lastError = err
		c.mu.Unlock()

		log.Printf("[updater] Update check failed: %v", err)
		if c.onError != nil {
			c.onError(err)
		}
		return nil, err
	}

	c.mu.Lock()
	c.lastError = nil
	c.latestInfo = info
	c.mu.Unlock()

	if !info.HasUpdate {
		log.Printf("[updater] No update available (current: %s, latest: %s)", info.CurrentVersion, info.LatestVersion)
		return info, nil
	}

	log.Printf("[updater] Update available: %s -> %s", info.CurrentVersion, info.LatestVersion)

	// Notify callback
	if c.onUpdate != nil {
		c.onUpdate(info)
	}

	// Auto-apply if enabled and graceful updater is available
	if c.autoApply && c.graceful != nil {
		go func() {
			// Use a new background context since we want the update to complete
			// even if the check context is cancelled. The graceful updater has
			// its own timeout (maxWaitTime) for draining.
			updateCtx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := c.graceful.ScheduleUpdate(updateCtx); err != nil {
				log.Printf("[updater] Failed to schedule update: %v", err)
			}
		}()
	}

	return info, nil
}

// UpdateAvailable returns true if an update is available.
func (c *BackgroundChecker) UpdateAvailable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latestInfo != nil && c.latestInfo.HasUpdate
}

// NextCheckIn returns the duration until the next check.
func (c *BackgroundChecker) NextCheckIn() time.Duration {
	c.mu.RLock()
	lastCheck := c.lastCheck
	c.mu.RUnlock()

	if lastCheck.IsZero() {
		return 0
	}

	nextCheck := lastCheck.Add(c.interval)
	remaining := time.Until(nextCheck)
	if remaining < 0 {
		return 0
	}
	return remaining
}
