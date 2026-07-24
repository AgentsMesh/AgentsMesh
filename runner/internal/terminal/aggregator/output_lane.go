package aggregator

import "sync"

type queuedOutput struct {
	data                   []byte
	generation             uint64
	acceptedWhileConnected bool
	barrier                *outputBarrierResult
}

type outputLane struct {
	name       string
	byteBudget int
	queue      chan queuedOutput
	onDesync   func(OutputHealth)
	stopCh     chan struct{}
	stopOnce   sync.Once

	mu              sync.Mutex
	writer          RelayWriter
	generation      uint64
	queuedBytes     int
	desynced        bool
	snapshotPending bool
	reason          OutputDesyncReason
	err             error
	stopped         bool
}

func newOutputLane(
	name string,
	writer RelayWriter,
	generation uint64,
	byteBudget int,
	itemLimit int,
	onDesync func(OutputHealth),
) *outputLane {
	lane := &outputLane{
		name:       name,
		writer:     writer,
		generation: generation,
		byteBudget: byteBudget,
		queue:      make(chan queuedOutput, itemLimit),
		onDesync:   onDesync,
		stopCh:     make(chan struct{}),
	}
	go lane.run()
	return lane
}

func (l *outputLane) enqueue(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	l.mu.Lock()
	if l.stopped || l.writer == nil || l.desynced || l.snapshotPending {
		l.mu.Unlock()
		return false
	}
	generation := l.generation
	if len(data) > l.byteBudget-l.queuedBytes {
		health, changed := l.desyncLocked(OutputDesyncOverflow, nil)
		l.mu.Unlock()
		l.notifyDesync(health, changed)
		return false
	}

	item := queuedOutput{data: data, generation: generation}
	l.queuedBytes += len(data)
	select {
	case l.queue <- item:
		l.mu.Unlock()
		return true
	default:
		l.queuedBytes -= len(data)
		health, changed := l.desyncLocked(OutputDesyncOverflow, nil)
		l.mu.Unlock()
		l.notifyDesync(health, changed)
		return false
	}
}

func (l *outputLane) enqueueBarrier() *outputBarrierResult {
	result := newOutputBarrierResult()
	l.mu.Lock()
	if l.stopped || l.writer == nil || l.desynced || l.snapshotPending {
		l.mu.Unlock()
		result.complete(false)
		return result
	}
	item := queuedOutput{generation: l.generation, barrier: result}
	select {
	case l.queue <- item:
		l.mu.Unlock()
		return result
	default:
		health, changed := l.desyncLocked(OutputDesyncOverflow, nil)
		l.mu.Unlock()
		l.notifyDesync(health, changed)
		result.complete(false)
		return result
	}
}

func (l *outputLane) run() {
	for {
		select {
		case <-l.stopCh:
			l.failQueuedBarriers()
			return
		case item := <-l.queue:
			l.deliver(item)
		}
	}
}

func (l *outputLane) deliver(item queuedOutput) {
	l.mu.Lock()
	writer := l.writer
	current := item.generation == l.generation
	canSend := current && !l.stopped && !l.desynced && writer != nil
	l.mu.Unlock()

	if item.barrier != nil {
		if !current {
			item.barrier.complete(true)
			return
		}
		if !canSend {
			item.barrier.complete(false)
			return
		}
		if err := writer.FlushOutput(item.barrier.ctx); err != nil {
			l.markDesynced(item.generation, OutputDesyncFlushError, err)
			item.barrier.complete(false)
			return
		}
		item.barrier.complete(true)
		return
	}
	defer l.releaseBytes(len(item.data))
	if !canSend {
		return
	}
	// The first check is the worker-side acceptance point. No observer is an
	// intentional drop. A disconnect after acceptance desynchronizes the stream.
	item.acceptedWhileConnected = writer.IsConnected()
	if !item.acceptedWhileConnected {
		return
	}
	if !writer.IsConnected() {
		l.markDesynced(item.generation, OutputDesyncDisconnected, nil)
		return
	}
	if err := writer.SendOutput(item.data); err != nil {
		l.markDesynced(item.generation, OutputDesyncSendError, err)
	}
}

func (l *outputLane) releaseBytes(size int) {
	l.mu.Lock()
	l.queuedBytes -= size
	if l.queuedBytes < 0 {
		l.queuedBytes = 0
	}
	l.mu.Unlock()
}

func (l *outputLane) stop() {
	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return
	}
	l.stopped = true
	l.mu.Unlock()
	l.stopOnce.Do(func() { close(l.stopCh) })
}

func (l *outputLane) failQueuedBarriers() {
	for {
		select {
		case item := <-l.queue:
			if item.barrier != nil {
				item.barrier.complete(false)
			} else {
				l.releaseBytes(len(item.data))
			}
		default:
			return
		}
	}
}
