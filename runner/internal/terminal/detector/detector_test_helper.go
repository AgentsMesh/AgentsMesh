package detector

import (
	"time"
)

// testDetectorConfig returns a config with short thresholds for fast testing.
// Thresholds must leave sufficient margin above simulateContinuousOutput's 30ms
// sleep to avoid false transitions on platforms with coarse timer resolution
// (e.g., Windows ~15ms granularity).
func testDetectorConfig() MultiSignalConfig {
	return MultiSignalConfig{
		IdleThreshold:    100 * time.Millisecond,
		ConfirmThreshold: 100 * time.Millisecond,
		MinStableTime:    100 * time.Millisecond,
		WaitingThreshold: 0.6,
	}
}

// simulateOutputAndScreen simulates output and screen update, then waits for stability.
func simulateOutputAndScreen(d *MultiSignalDetector, bytes int, lines []string) {
	d.OnOutput(bytes)
	d.OnScreenUpdate(lines)
	time.Sleep(150 * time.Millisecond)
	d.OnScreenUpdate(lines) // Same content for stability
}

// simulateContinuousOutput simulates continuous output bursts.
func simulateContinuousOutput(d *MultiSignalDetector, rounds int, bytesPerRound int, lines []string) {
	for i := 0; i < rounds; i++ {
		d.OnOutput(bytesPerRound)
		d.OnScreenUpdate(lines)
		time.Sleep(30 * time.Millisecond)
		d.DetectState()
	}
}
