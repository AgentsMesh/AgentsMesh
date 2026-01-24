package terminal

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestBackpressureController_NewBackpressureController(t *testing.T) {
	bc := NewBackpressureController(nil, nil)

	if bc.IsPaused() {
		t.Error("Should not be paused initially")
	}
}

func TestBackpressureController_Pause(t *testing.T) {
	var pauseCalled int32
	bc := NewBackpressureController(
		func() { atomic.AddInt32(&pauseCalled, 1) },
		nil,
	)

	bc.Pause()
	if !bc.IsPaused() {
		t.Error("Should be paused after Pause()")
	}
	if atomic.LoadInt32(&pauseCalled) != 1 {
		t.Error("onPause should be called once")
	}

	// Double pause should not call onPause again
	bc.Pause()
	if atomic.LoadInt32(&pauseCalled) != 1 {
		t.Error("onPause should not be called on double pause")
	}
}

func TestBackpressureController_Resume(t *testing.T) {
	var resumeCalled int32
	bc := NewBackpressureController(
		nil,
		func() { atomic.AddInt32(&resumeCalled, 1) },
	)

	// Resume without pause should return false and not call callback
	wasPaused := bc.Resume()
	if wasPaused {
		t.Error("Resume should return false when not paused")
	}
	if atomic.LoadInt32(&resumeCalled) != 0 {
		t.Error("onResume should not be called when not paused")
	}

	// Pause then resume
	bc.Pause()
	wasPaused = bc.Resume()
	if !wasPaused {
		t.Error("Resume should return true when was paused")
	}
	if bc.IsPaused() {
		t.Error("Should not be paused after Resume()")
	}
	if atomic.LoadInt32(&resumeCalled) != 1 {
		t.Error("onResume should be called once")
	}
}

func TestBackpressureController_ResumeCh(t *testing.T) {
	bc := NewBackpressureController(nil, nil)

	ch := bc.ResumeCh()
	if ch == nil {
		t.Fatal("ResumeCh should not be nil")
	}

	// Pause and resume should signal channel
	bc.Pause()
	bc.Resume()

	select {
	case <-ch:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Error("ResumeCh should receive signal after resume")
	}
}

func TestBackpressureController_SetCallbacks(t *testing.T) {
	bc := NewBackpressureController(nil, nil)

	var pauseCalled, resumeCalled int32
	bc.SetCallbacks(
		func() { atomic.AddInt32(&pauseCalled, 1) },
		func() { atomic.AddInt32(&resumeCalled, 1) },
	)

	bc.Pause()
	if atomic.LoadInt32(&pauseCalled) != 1 {
		t.Error("New onPause callback should be called")
	}

	bc.Resume()
	if atomic.LoadInt32(&resumeCalled) != 1 {
		t.Error("New onResume callback should be called")
	}
}

func TestBackpressureController_NilCallbacks(t *testing.T) {
	bc := NewBackpressureController(nil, nil)

	// Should not panic with nil callbacks
	bc.Pause()
	bc.Resume()

	if bc.IsPaused() {
		t.Error("Should not be paused after resume")
	}
}

func TestBackpressureController_Concurrent(t *testing.T) {
	bc := NewBackpressureController(nil, nil)

	done := make(chan struct{})

	// Concurrent pause/resume should not cause race conditions
	go func() {
		for i := 0; i < 100; i++ {
			bc.Pause()
			bc.Resume()
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			bc.IsPaused()
		}
		done <- struct{}{}
	}()

	<-done
	<-done
}
