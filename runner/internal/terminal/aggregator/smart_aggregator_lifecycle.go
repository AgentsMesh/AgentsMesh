package aggregator

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// Flush forces an immediate flush of the buffer.
func (a *SmartAggregator) Flush() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.stopped {
		a.forceFlushLocked()
	}
}

// Stop stops the aggregator and boundedly drains remaining output.
func (a *SmartAggregator) Stop() {
	a.mu.Lock()
	if a.stopped {
		done := a.stopDone
		timeout := a.drainTimeout
		a.mu.Unlock()
		waitForOutputDrain(done, timeout)
		return
	}
	a.stopped = true
	logger.Terminal().Info("SmartAggregator stopped")
	a.forceFlushLocked()
	barrier := a.router.barrier()
	timeout := a.drainTimeout
	done := a.stopDone
	a.mu.Unlock()

	if !barrier.wait(timeout) {
		logger.Terminal().Warn("SmartAggregator output drain timed out", "timeout", timeout)
	}
	a.router.stop()
	close(done)
}

func waitForOutputDrain(done <-chan struct{}, timeout time.Duration) bool {
	if timeout <= 0 {
		timeout = defaultOutputDrainTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

func (a *SmartAggregator) IsStopped() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stopped
}

func (a *SmartAggregator) BufferLen() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.buffer.Len()
}

// SetOutputDestination installs one independently ordered output sink.
func (a *SmartAggregator) SetOutputDestination(name string, writer RelayWriter) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.stopped {
		a.router.SetDestination(name, writer)
	}
}

// RemoveOutputDestination removes one sink without disturbing the others.
func (a *SmartAggregator) RemoveOutputDestination(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.stopped {
		a.router.RemoveDestination(name)
	}
}

func (a *SmartAggregator) SetPTYLogger(ptyLogger *PTYLogger) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ptyLogger = ptyLogger
}

func (a *SmartAggregator) calculateDelay(usage float64) time.Duration {
	return a.delay.CalculateForUsage(usage)
}
