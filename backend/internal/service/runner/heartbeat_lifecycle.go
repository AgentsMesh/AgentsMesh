package runner

import (
	"time"
)

func (b *HeartbeatBatcher) Start() {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return
	}
	b.running = true
	b.stopCh = make(chan struct{})
	b.doneCh = make(chan struct{})
	stopCh := b.stopCh
	doneCh := b.doneCh
	b.mu.Unlock()

	go b.flushLoop(stopCh, doneCh)
	b.logger.Info("heartbeat batcher started", "interval", b.interval)
}

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

func (b *HeartbeatBatcher) flushLoop(stopCh <-chan struct{}, doneCh chan<- struct{}) {
	defer close(doneCh)

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.flush()
		case <-stopCh:
			b.flush()
			return
		}
	}
}

func (b *HeartbeatBatcher) Flush() {
	b.flush()
}
