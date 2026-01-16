package runner

import (
	"io"
	"log"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/buffer"
	"github.com/anthropics/agentsmesh/runner/internal/terminal"
)

// OutputHandler defines the interface for handling PTY output.
type OutputHandler interface {
	// SendOutput sends output data to the destination.
	// Returns false if the send should be throttled (backpressure).
	SendOutput(podKey string, data []byte) bool
}

// PTYForwarder forwards PTY output with backpressure control.
// It buffers output when the downstream is slow to prevent data loss.
type PTYForwarder struct {
	session  *terminal.Session
	podKey   string
	handler  OutputHandler
	outputBuf   *buffer.Ring
	mu          sync.Mutex
	done        chan struct{}
	flushTicker *time.Ticker

	// Backpressure control
	backpressure     bool
	backpressureMu   sync.RWMutex
	maxBufferSize    int
	flushInterval    time.Duration
	backpressureWait time.Duration
}

// PTYForwarderConfig holds configuration for PTYForwarder.
type PTYForwarderConfig struct {
	Session    *terminal.Session
	PodKey     string
	Handler    OutputHandler
	BufferSize       int           // Size of output buffer (default: 64KB)
	FlushInterval    time.Duration // How often to flush buffered data (default: 50ms)
	BackpressureWait time.Duration // How long to wait when backpressure is detected (default: 100ms)
}

// NewPTYForwarder creates a new PTY forwarder.
func NewPTYForwarder(cfg *PTYForwarderConfig) *PTYForwarder {
	bufSize := cfg.BufferSize
	if bufSize <= 0 {
		bufSize = 64 * 1024 // 64KB default
	}

	flushInterval := cfg.FlushInterval
	if flushInterval <= 0 {
		flushInterval = 50 * time.Millisecond
	}

	backpressureWait := cfg.BackpressureWait
	if backpressureWait <= 0 {
		backpressureWait = 100 * time.Millisecond
	}

	return &PTYForwarder{
		session:          cfg.Session,
		podKey:           cfg.PodKey,
		handler:          cfg.Handler,
		outputBuf:        buffer.NewRing(bufSize),
		done:             make(chan struct{}),
		maxBufferSize:    bufSize,
		flushInterval:    flushInterval,
		backpressureWait: backpressureWait,
	}
}

// Start begins forwarding PTY output.
func (f *PTYForwarder) Start() {
	// Start flush ticker
	f.flushTicker = time.NewTicker(f.flushInterval)

	go f.readLoop()
	go f.flushLoop()

	log.Printf("[pty_forwarder] Started forwarding: pod_key=%s", f.podKey)
}

// Stop stops the forwarder.
func (f *PTYForwarder) Stop() {
	close(f.done)
	if f.flushTicker != nil {
		f.flushTicker.Stop()
	}
	log.Printf("[pty_forwarder] Stopped forwarding: pod_key=%s", f.podKey)
}

// readLoop continuously reads from PTY and buffers output.
func (f *PTYForwarder) readLoop() {
	// Guard against nil session
	if f.session == nil {
		log.Printf("[pty_forwarder] Read loop exiting: session is nil, pod_key=%s", f.podKey)
		return
	}

	buf := make([]byte, 4096)

	for {
		select {
		case <-f.done:
			return
		default:
		}

		n, err := f.session.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("[pty_forwarder] Read error: pod_key=%s, error=%v",
					f.podKey, err)
			}
			return
		}

		if n > 0 {
			f.bufferOutput(buf[:n])
		}
	}
}

// bufferOutput adds data to the output buffer.
func (f *PTYForwarder) bufferOutput(data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Write to buffer
	f.outputBuf.Write(data)

	// If buffer is getting full, try immediate flush
	if f.outputBuf.Len() > f.maxBufferSize/2 {
		f.flushLocked()
	}
}

// flushLoop periodically flushes buffered output.
func (f *PTYForwarder) flushLoop() {
	for {
		select {
		case <-f.done:
			// Final flush
			f.flush()
			return
		case <-f.flushTicker.C:
			f.flush()
		}
	}
}

// flush sends buffered data to the handler.
func (f *PTYForwarder) flush() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.flushLocked()
}

// flushLocked flushes with lock already held.
func (f *PTYForwarder) flushLocked() {
	data := f.outputBuf.Bytes()
	if len(data) == 0 {
		return
	}

	// Check backpressure
	if f.isBackpressure() {
		// Wait and retry
		time.Sleep(f.backpressureWait)
		if f.isBackpressure() {
			// Still backpressure, skip this flush
			log.Printf("[pty_forwarder] Backpressure detected, skipping flush: pod_key=%s, buffered=%d",
				f.podKey, len(data))
			return
		}
	}

	// Send output
	if !f.handler.SendOutput(f.podKey, data) {
		f.setBackpressure(true)
		log.Printf("[pty_forwarder] Backpressure signal received: pod_key=%s", f.podKey)
		return
	}

	// Clear buffer after successful send
	f.outputBuf.Reset()
	f.setBackpressure(false)
}

// isBackpressure returns current backpressure state.
func (f *PTYForwarder) isBackpressure() bool {
	f.backpressureMu.RLock()
	defer f.backpressureMu.RUnlock()
	return f.backpressure
}

// setBackpressure sets the backpressure state.
func (f *PTYForwarder) setBackpressure(bp bool) {
	f.backpressureMu.Lock()
	defer f.backpressureMu.Unlock()
	f.backpressure = bp
}

// GetBufferedSize returns the current buffered data size.
func (f *PTYForwarder) GetBufferedSize() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.outputBuf.Len()
}
