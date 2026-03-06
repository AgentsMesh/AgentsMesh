package safego

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGo_NormalExecution(t *testing.T) {
	ResetPanicCount()

	var executed atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)

	Go("test-normal", func() {
		defer wg.Done()
		executed.Store(true)
	})

	wg.Wait()

	if !executed.Load() {
		t.Error("expected function to be executed")
	}
	if PanicCount() != 0 {
		t.Errorf("expected 0 panics, got %d", PanicCount())
	}
}

func TestGo_PanicRecovery(t *testing.T) {
	ResetPanicCount()

	Go("test-panic", func() {
		panic("test panic")
	})

	// Wait for run()'s deferred recover to increment panicCount.
	// We poll PanicCount directly instead of using a done channel inside fn,
	// because fn's inner defer runs before run()'s outer recover defer,
	// which would create a race between the signal and the counter increment.
	deadline := time.After(2 * time.Second)
	for PanicCount() != 1 {
		select {
		case <-deadline:
			t.Fatalf("timeout: expected 1 panic, got %d", PanicCount())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestGo_DoesNotCrashProcess(t *testing.T) {
	ResetPanicCount()

	// Start a goroutine that panics
	Go("test-crash", func() {
		panic("should not crash process")
	})

	// Wait for panic recovery to complete by polling PanicCount.
	// Same rationale as TestGo_PanicRecovery — avoids inner/outer defer race.
	deadline := time.After(2 * time.Second)
	for PanicCount() != 1 {
		select {
		case <-deadline:
			t.Fatalf("timeout: expected 1 panic, got %d", PanicCount())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	// If we get here, the process didn't crash
}

func TestGoLoop_NormalReturn(t *testing.T) {
	ResetPanicCount()

	var callCount atomic.Int32
	done := make(chan struct{})

	GoLoop("test-loop-normal", func() {
		callCount.Add(1)
		close(done)
		// Normal return — should NOT restart
	}, 3)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	// Wait a bit to ensure no extra restarts
	time.Sleep(100 * time.Millisecond)

	if callCount.Load() != 1 {
		t.Errorf("expected 1 call, got %d", callCount.Load())
	}
	if PanicCount() != 0 {
		t.Errorf("expected 0 panics, got %d", PanicCount())
	}
}

func TestGoLoop_PanicAndRestart(t *testing.T) {
	ResetPanicCount()

	var callCount atomic.Int32
	done := make(chan struct{}, 1)

	GoLoop("test-loop-restart", func() {
		n := callCount.Add(1)
		if n < 3 {
			panic("restart me")
		}
		// Third call succeeds
		select {
		case done <- struct{}{}:
		default:
		}
	}, 5)

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for successful execution")
	}

	if callCount.Load() != 3 {
		t.Errorf("expected 3 calls, got %d", callCount.Load())
	}
	if PanicCount() != 2 {
		t.Errorf("expected 2 panics, got %d", PanicCount())
	}
}

func TestGoLoop_MaxRestartsExceeded(t *testing.T) {
	ResetPanicCount()

	var callCount atomic.Int32
	done := make(chan struct{})

	GoLoop("test-loop-max", func() {
		defer func() {
			if callCount.Load() >= 3 {
				select {
				case <-done:
				default:
					close(done)
				}
			}
		}()
		callCount.Add(1)
		panic("always panic")
	}, 3)

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for max restarts")
	}

	// Wait a bit to confirm no further restarts
	time.Sleep(200 * time.Millisecond)

	if callCount.Load() != 3 {
		t.Errorf("expected 3 calls (initial + 2 restarts capped at 3), got %d", callCount.Load())
	}
}

func TestGoLoop_UnlimitedRestarts(t *testing.T) {
	ResetPanicCount()

	var callCount atomic.Int32
	done := make(chan struct{}, 1)

	GoLoop("test-loop-unlimited", func() {
		n := callCount.Add(1)
		if n < 5 {
			panic("restart me")
		}
		// Fifth call succeeds
		select {
		case done <- struct{}{}:
		default:
		}
	}, 0) // maxRestarts=0 means unlimited

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("timeout")
	}

	if callCount.Load() != 5 {
		t.Errorf("expected 5 calls, got %d", callCount.Load())
	}
}

func TestRunWithPanicFlag_NoPanic(t *testing.T) {
	panicked := runWithPanicFlag("test", func() {
		// Normal return
	})
	if panicked {
		t.Error("expected panicked to be false for normal return")
	}
}

func TestRunWithPanicFlag_WithPanic(t *testing.T) {
	ResetPanicCount()

	panicked := runWithPanicFlag("test", func() {
		panic("test")
	})
	if !panicked {
		t.Error("expected panicked to be true after panic")
	}
}
