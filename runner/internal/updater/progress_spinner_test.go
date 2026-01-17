package updater

import (
	"testing"
	"time"
)

// Tests for SpinnerProgress functionality

func TestSpinnerProgress_StartStop(t *testing.T) {
	sp := NewSpinnerProgress("Test spinner")

	sp.Start()
	time.Sleep(100 * time.Millisecond)

	sp.Stop()
	sp.Stop() // Double stop should not panic
}

func TestSpinnerProgress_StopWithMessage(t *testing.T) {
	sp := NewSpinnerProgress("Test spinner")

	sp.Start()
	time.Sleep(100 * time.Millisecond)

	sp.StopWithMessage("Done!")
	sp.StopWithMessage("Done again!") // Double stop should not panic
}

func TestSpinnerProgress_StopBeforeStart(t *testing.T) {
	sp := NewSpinnerProgress("Test")
	sp.Stop() // Should not panic
}

func TestSpinnerProgress_StopWithMessageBeforeStart(t *testing.T) {
	sp := NewSpinnerProgress("Test")
	sp.StopWithMessage("Done!") // Should not panic
}

func TestConsoleProgress_ZeroTotal(t *testing.T) {
	cp := NewConsoleProgress()
	cp.Update(50, 0) // Should not panic
}

func TestConsoleProgress_ExceedsTotal(t *testing.T) {
	cp := NewConsoleProgress()
	cp.Update(150, 100) // Should not panic
}

func TestConsoleProgress_Throttling(t *testing.T) {
	cp := NewConsoleProgress()

	cp.Update(10, 100)
	cp.Update(20, 100)

	time.Sleep(150 * time.Millisecond)
	cp.Update(30, 100)
}

func TestConsoleProgress_CalculateETA(t *testing.T) {
	cp := NewConsoleProgress()
	cp.startTime = time.Now().Add(-10 * time.Second)
	cp.Update(50, 100)
}

func TestConsoleProgress_Completion(t *testing.T) {
	cp := NewConsoleProgress()
	cp.Update(100, 100)
}
