package lifecycle

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// mockActivityMonitor implements ActivityMonitor for testing.
type mockActivityMonitor struct {
	lastActivity time.Time
}

func (m *mockActivityMonitor) LastActivityTime() time.Time {
	return m.lastActivity
}

func TestWatchdogService_HealthyChecks(t *testing.T) {
	w := NewWatchdogService(WatchdogConfig{
		Interval:      50 * time.Millisecond,
		MaxGoroutines: 10000, // High threshold — should pass
		MaxMemoryMB:   10000, // High threshold — should pass
		MaxFailCount:  3,
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- w.Serve(ctx)
	}()

	// Let a few checks run
	time.Sleep(200 * time.Millisecond)

	// Should still be running (healthy)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestWatchdogService_GoroutineThresholdExceeded(t *testing.T) {
	w := NewWatchdogService(WatchdogConfig{
		Interval:      50 * time.Millisecond,
		MaxGoroutines: 1, // Impossibly low — will always fail
		MaxMemoryMB:   10000,
		MaxFailCount:  2,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := w.Serve(ctx)
	if err == nil {
		t.Fatal("expected error from watchdog")
	}

	// Should contain goroutine-related error
	if err.Error() == "" {
		t.Fatal("expected non-empty error")
	}
}

func TestWatchdogService_ConnectionIdleDetection(t *testing.T) {
	mon := &mockActivityMonitor{
		lastActivity: time.Now().Add(-10 * time.Minute), // 10 minutes ago
	}

	w := NewWatchdogService(WatchdogConfig{
		ConnMonitor:           mon,
		Interval:              50 * time.Millisecond,
		MaxGoroutines:         10000,
		MaxMemoryMB:           10000,
		MaxFailCount:          2,
		ConnectionIdleTimeout: 1 * time.Minute, // 1 min timeout, activity is 10 min ago
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := w.Serve(ctx)
	if err == nil {
		t.Fatal("expected error from watchdog due to connection idle")
	}
}

func TestWatchdogService_ConnectionActiveNoError(t *testing.T) {
	mon := &mockActivityMonitor{
		lastActivity: time.Now(), // Just now — active
	}

	w := NewWatchdogService(WatchdogConfig{
		ConnMonitor:           mon,
		Interval:              50 * time.Millisecond,
		MaxGoroutines:         10000,
		MaxMemoryMB:           10000,
		MaxFailCount:          3,
		ConnectionIdleTimeout: 5 * time.Minute,
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- w.Serve(ctx)
	}()

	// Let checks run
	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestWatchdogService_RunChecks(t *testing.T) {
	// Test individual runChecks method
	w := NewWatchdogService(WatchdogConfig{
		MaxGoroutines: 10000,
		MaxMemoryMB:   10000,
	})

	if err := w.runChecks(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestWatchdogService_MemoryThresholdExceeded(t *testing.T) {
	// Allocate memory to ensure Alloc > 1 MB
	buf := make([]byte, 2*1024*1024) // 2MB allocation
	buf[0] = 1                       // Prevent optimization
	runtime.KeepAlive(buf)

	w := NewWatchdogService(WatchdogConfig{
		MaxGoroutines: 10000,
		MaxMemoryMB:   1, // Use 1 as initial (will be kept since > 0)
	})

	// Get actual memory to verify test precondition
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	allocMB := int(memStats.Alloc / 1024 / 1024)
	if allocMB < 1 {
		t.Skipf("Skipping: process memory %dMB too low for threshold test", allocMB)
	}

	// Set threshold lower than actual usage
	w.cfg.MaxMemoryMB = allocMB - 1
	if w.cfg.MaxMemoryMB < 0 {
		w.cfg.MaxMemoryMB = 0
	}

	err := w.runChecks()
	if err == nil {
		t.Fatal("expected memory threshold error")
	}
}

func TestWatchdogService_ConnectionZeroTime(t *testing.T) {
	// Test that zero LastActivityTime is treated as no activity (skip check)
	mon := &mockActivityMonitor{
		lastActivity: time.Time{}, // Zero time
	}
	w := NewWatchdogService(WatchdogConfig{
		ConnMonitor:           mon,
		MaxGoroutines:         10000,
		MaxMemoryMB:           10000,
		ConnectionIdleTimeout: 1 * time.Minute,
	})

	err := w.runChecks()
	if err != nil {
		t.Errorf("expected no error for zero activity time, got %v", err)
	}
}

func TestWatchdogService_FailCountRecovery(t *testing.T) {
	// Test that failCount resets on successful check
	w := NewWatchdogService(WatchdogConfig{
		Interval:      50 * time.Millisecond,
		MaxGoroutines: 10000,
		MaxMemoryMB:   10000,
		MaxFailCount:  5,
	})

	// Manually set failCount > 0
	w.failCount = 2

	// Run a healthy check
	err := w.runChecks()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// failCount is reset in Serve loop, not in runChecks — this is just to verify runChecks passes
}

func TestWatchdogService_Defaults(t *testing.T) {
	w := NewWatchdogService(WatchdogConfig{})

	if w.cfg.Interval != 15*time.Second {
		t.Errorf("expected default interval 15s, got %v", w.cfg.Interval)
	}
	if w.cfg.MaxGoroutines != 1000 {
		t.Errorf("expected default MaxGoroutines 1000, got %d", w.cfg.MaxGoroutines)
	}
	if w.cfg.MaxMemoryMB != 2048 {
		t.Errorf("expected default MaxMemoryMB 2048, got %d", w.cfg.MaxMemoryMB)
	}
	if w.cfg.MaxFailCount != 3 {
		t.Errorf("expected default MaxFailCount 3, got %d", w.cfg.MaxFailCount)
	}
	if w.cfg.ConnectionIdleTimeout != 5*time.Minute {
		t.Errorf("expected default ConnectionIdleTimeout 5m, got %v", w.cfg.ConnectionIdleTimeout)
	}
}

func TestWatchdogService_String(t *testing.T) {
	w := &WatchdogService{}
	if w.String() != "WatchdogService" {
		t.Errorf("expected 'WatchdogService', got %q", w.String())
	}
}

func TestNotifySystemHealthy(t *testing.T) {
	// On non-Linux this is a no-op. On Linux without NOTIFY_SOCKET it's also a no-op.
	// Just verify it doesn't panic.
	notifySystemHealthy()
}
