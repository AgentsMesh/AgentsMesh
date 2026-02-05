package runner

import (
	"time"
)

// Start starts the background flush loop
func (b *HeartbeatBatcher) Start() {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return
	}
	b.running = true
	// Create new channels for this lifecycle (allows restart after Stop)
	b.stopCh = make(chan struct{})
	b.doneCh = make(chan struct{})
	stopCh := b.stopCh
	doneCh := b.doneCh
	b.mu.Unlock()

	go b.flushLoop(stopCh, doneCh)
	b.logger.Info("heartbeat batcher started", "interval", b.interval)
}

// Stop stops the batcher and flushes remaining items
func (b *HeartbeatBatcher) Stop() {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return
	}
	b.running = false
	stopCh := b.stopCh
	doneCh := b.doneCh
	b.mu.Unlock()

	close(stopCh)
	<-doneCh
	b.logger.Info("heartbeat batcher stopped")
}

// flushLoop runs the periodic flush
func (b *HeartbeatBatcher) flushLoop(stopCh <-chan struct{}, doneCh chan<- struct{}) {
	defer close(doneCh)

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.flush()
		case <-stopCh:
			// Final flush before exit
			b.flush()
			return
		}
	}
}

// Flush immediately flushes all buffered heartbeats to the database
// This is useful for testing and graceful shutdown scenarios
func (b *HeartbeatBatcher) Flush() {
	b.flush()
}
