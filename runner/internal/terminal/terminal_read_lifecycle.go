package terminal

import "time"

const readDrainGrace = 250 * time.Millisecond

func (t *Terminal) withReadOrder(fn func()) {
	t.mu.Lock()
	locker := t.readOrder
	t.mu.Unlock()
	if locker != nil {
		locker.Lock()
		defer locker.Unlock()
	}
	fn()
}

func (t *Terminal) signalReadDone() {
	if t.readDoneCh != nil {
		t.readDoneOnce.Do(func() { close(t.readDoneCh) })
	}
}

func (t *Terminal) signalReadActive() {
	if t.readActiveCh != nil {
		t.readActiveOnce.Do(func() { close(t.readActiveCh) })
	}
}

func (t *Terminal) signalReadProgress() {
	t.mu.Lock()
	t.readProgress++
	progress := t.readProgressCh
	t.mu.Unlock()
	if progress != nil {
		select {
		case progress <- struct{}{}:
		default:
		}
	}
}

func (t *Terminal) readProgressState() (uint64, <-chan struct{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.readProgress, t.readProgressCh
}

func (t *Terminal) readLifecycle() (bool, <-chan struct{}, <-chan struct{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.readStarted, t.readActiveCh, t.readDoneCh
}

func (t *Terminal) waitForReadDone() {
	started, _, done := t.readLifecycle()
	if started && done != nil {
		<-done
	}
}

// drainReadOutput lets a naturally exited PTY expose buffered tail bytes before
// closing the descriptor, then joins the reader before the exit callback runs.
func (t *Terminal) drainReadOutput() {
	started, active, done := t.readLifecycle()
	if !started || done == nil {
		return
	}

	// A very fast child may exit before the reader goroutine is scheduled. Do
	// not start the close grace period until that reader either owns the PTY or
	// has already exited, otherwise closing here can discard the entire output.
	select {
	case <-done:
		return
	case <-active:
	}

	lastProgress, progress := t.readProgressState()
	timer := time.NewTimer(readDrainGrace)
	defer timer.Stop()
	for {
		select {
		case <-done:
			return
		case <-progress:
			lastProgress, _ = t.readProgressState()
			resetDrainTimer(timer)
		case <-timer.C:
			closedForIdle := false
			t.withReadOrder(func() {
				current, _ := t.readProgressState()
				if current != lastProgress {
					lastProgress = current
					return
				}
				t.closePTY()
				closedForIdle = true
			})
			if closedForIdle {
				<-done
				return
			}
			resetDrainTimer(timer)
		}
	}
}

func resetDrainTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(readDrainGrace)
}
