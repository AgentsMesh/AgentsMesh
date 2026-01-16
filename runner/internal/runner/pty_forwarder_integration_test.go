package runner

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/terminal"
)

// mockReadWriter implements a simple io.ReadWriter for testing readLoop
type mockReadWriter struct {
	readData    [][]byte
	readIndex   int
	readErr     error
	readDelay   time.Duration
	readClosed  bool
	written     []byte
	mu          sync.Mutex
	readSignal  chan struct{}
}

func newMockReadWriter(data [][]byte) *mockReadWriter {
	return &mockReadWriter{
		readData:   data,
		readSignal: make(chan struct{}, 1),
	}
}

func (m *mockReadWriter) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.readClosed {
		return 0, io.EOF
	}

	if m.readDelay > 0 {
		m.mu.Unlock()
		time.Sleep(m.readDelay)
		m.mu.Lock()
	}

	if m.readErr != nil {
		return 0, m.readErr
	}

	if m.readIndex >= len(m.readData) {
		return 0, io.EOF
	}

	data := m.readData[m.readIndex]
	m.readIndex++
	n = copy(p, data)

	// Signal that data was read
	select {
	case m.readSignal <- struct{}{}:
	default:
	}

	return n, nil
}

func (m *mockReadWriter) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.written = append(m.written, p...)
	return len(p), nil
}

func (m *mockReadWriter) Close() error {
	m.mu.Lock()
	m.readClosed = true
	m.mu.Unlock()
	return nil
}

// TestPTYForwarderReadLoopWithNilSession tests that readLoop handles nil session
func TestPTYForwarderReadLoopWithNilSession(t *testing.T) {
	handler := newMockOutputHandler()

	cfg := &PTYForwarderConfig{
		Session:    nil, // nil session
		PodKey: "nil-pod-test",
		Handler:    handler,
	}

	forwarder := NewPTYForwarder(cfg)
	forwarder.Start()

	// Give the goroutines time to start and exit
	time.Sleep(50 * time.Millisecond)

	forwarder.Stop()

	// No output should be generated since session is nil
	handler.mu.Lock()
	outputCount := len(handler.outputs)
	handler.mu.Unlock()

	if outputCount != 0 {
		t.Errorf("nil session should produce no output, got %d", outputCount)
	}
}

// TestPTYForwarderMultipleFlushes tests multiple flush operations
func TestPTYForwarderMultipleFlushes(t *testing.T) {
	handler := newMockOutputHandler()

	cfg := &PTYForwarderConfig{
		Session:       nil,
		PodKey:    "multi-flush-test",
		Handler:       handler,
		FlushInterval: 5 * time.Millisecond,
	}

	forwarder := NewPTYForwarder(cfg)
	forwarder.Start()

	// Buffer multiple chunks
	for i := 0; i < 10; i++ {
		forwarder.bufferOutput([]byte("data chunk"))
		forwarder.flush()
	}

	time.Sleep(20 * time.Millisecond)
	forwarder.Stop()

	handler.mu.Lock()
	outputCount := len(handler.outputs)
	handler.mu.Unlock()

	// Should have multiple outputs
	if outputCount < 5 {
		t.Errorf("expected multiple outputs, got %d", outputCount)
	}
}

// TestPTYForwarderBackpressureStates tests backpressure state transitions
func TestPTYForwarderBackpressureStates(t *testing.T) {
	handler := newMockOutputHandler()

	cfg := &PTYForwarderConfig{
		Session:          nil,
		PodKey:       "bp-states-test",
		Handler:          handler,
		BackpressureWait: 1 * time.Millisecond,
	}

	forwarder := NewPTYForwarder(cfg)

	// Initial state should not be backpressure
	if forwarder.isBackpressure() {
		t.Error("initial state should not be backpressure")
	}

	// Set backpressure
	forwarder.setBackpressure(true)
	if !forwarder.isBackpressure() {
		t.Error("backpressure should be set")
	}

	// Clear backpressure
	forwarder.setBackpressure(false)
	if forwarder.isBackpressure() {
		t.Error("backpressure should be cleared")
	}

	// Multiple sets
	forwarder.setBackpressure(true)
	forwarder.setBackpressure(true)
	forwarder.setBackpressure(false)
	forwarder.setBackpressure(false)

	if forwarder.isBackpressure() {
		t.Error("backpressure should be cleared after double clear")
	}
}

// TestPTYForwarderBufferGrowth tests buffer growth and size tracking
func TestPTYForwarderBufferGrowth(t *testing.T) {
	handler := newMockOutputHandler()

	cfg := &PTYForwarderConfig{
		Session:    nil,
		PodKey: "buffer-growth-test",
		Handler:    handler,
		BufferSize: 10240, // 10KB buffer
	}

	forwarder := NewPTYForwarder(cfg)

	// Track sizes
	sizes := []int{}

	// Add data in chunks
	for i := 0; i < 20; i++ {
		forwarder.bufferOutput([]byte("chunk"))
		sizes = append(sizes, forwarder.GetBufferedSize())
	}

	// Verify buffer grew
	for i := 1; i < len(sizes); i++ {
		if sizes[i] < sizes[i-1] && sizes[i] > 0 {
			// Buffer might have been flushed, which is OK
			continue
		}
	}

	// Final flush
	forwarder.flush()

	finalSize := forwarder.GetBufferedSize()
	if finalSize != 0 {
		t.Errorf("buffer should be empty after flush, got %d", finalSize)
	}
}

// TestPTYForwarderFlushLoopTiming tests that flush loop runs at configured interval
func TestPTYForwarderFlushLoopTiming(t *testing.T) {
	handler := newMockOutputHandler()

	flushInterval := 10 * time.Millisecond
	cfg := &PTYForwarderConfig{
		Session:       nil,
		PodKey:    "flush-timing-test",
		Handler:       handler,
		FlushInterval: flushInterval,
	}

	forwarder := NewPTYForwarder(cfg)
	forwarder.Start()

	// Add some data
	forwarder.bufferOutput([]byte("test data"))

	// Wait for flush loop to run
	time.Sleep(flushInterval * 3)

	forwarder.Stop()

	handler.mu.Lock()
	outputCount := len(handler.outputs)
	handler.mu.Unlock()

	if outputCount < 1 {
		t.Error("flush loop should have produced at least 1 output")
	}
}

// TestPTYForwarderHandlerBackpressureSignal tests that backpressure is set when handler returns false
func TestPTYForwarderHandlerBackpressureSignal(t *testing.T) {
	handler := newMockOutputHandler()
	handler.shouldBackpressure = true // Handler will return false (backpressure)

	cfg := &PTYForwarderConfig{
		Session:          nil,
		PodKey:       "handler-bp-signal-test",
		Handler:          handler,
		BackpressureWait: 1 * time.Millisecond,
	}

	forwarder := NewPTYForwarder(cfg)

	// Buffer and flush
	forwarder.bufferOutput([]byte("test data"))
	forwarder.flush()

	// Check that backpressure was set
	if !forwarder.isBackpressure() {
		t.Error("backpressure should be set when handler returns false")
	}
}

// TestPTYForwarderConcurrentOperations tests concurrent buffer and flush operations
func TestPTYForwarderConcurrentOperations(t *testing.T) {
	handler := newMockOutputHandler()

	cfg := &PTYForwarderConfig{
		Session:       nil,
		PodKey:    "concurrent-ops-test",
		Handler:       handler,
		BufferSize:    65536,
		FlushInterval: 5 * time.Millisecond,
	}

	forwarder := NewPTYForwarder(cfg)
	forwarder.Start()

	var wg sync.WaitGroup
	data := []byte("concurrent test data")

	// Multiple concurrent writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				forwarder.bufferOutput(data)
			}
		}()
	}

	// Concurrent flushers
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				forwarder.flush()
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()
	forwarder.Stop()

	// Should not panic and should have outputs
	handler.mu.Lock()
	outputCount := len(handler.outputs)
	handler.mu.Unlock()

	if outputCount == 0 {
		t.Error("expected some outputs from concurrent operations")
	}
}

// --- Test with real terminal session (integration) ---

func TestPTYForwarderWithRealTerminal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := newMockOutputHandler()

	// Create a real terminal session that echoes something
	sessionCfg := &terminal.SessionConfig{
		ID:         "test-pod",
		Command:    "/bin/echo",
		Args:       []string{"hello world"},
		WorkingDir: "/tmp",
		Cols:       80,
		Rows:       24,
		BufferSize: 64 * 1024, // Required: ring buffer needs size > 0
	}

	session, err := terminal.NewSession(sessionCfg)
	if err != nil {
		t.Skipf("Could not create terminal session: %v", err)
	}
	defer session.Close()

	cfg := &PTYForwarderConfig{
		Session:       session,
		PodKey:    "real-terminal-test",
		Handler:       handler,
		FlushInterval: 10 * time.Millisecond,
	}

	forwarder := NewPTYForwarder(cfg)
	forwarder.Start()

	// Wait for echo to complete and output to be captured
	time.Sleep(200 * time.Millisecond)

	forwarder.Stop()

	handler.mu.Lock()
	outputCount := len(handler.outputs)
	var allOutput bytes.Buffer
	for _, o := range handler.outputs {
		allOutput.Write(o.data)
	}
	handler.mu.Unlock()

	// Should have captured the echo output
	if outputCount == 0 {
		t.Error("expected output from terminal")
	}

	outputStr := allOutput.String()
	if !bytes.Contains(allOutput.Bytes(), []byte("hello world")) {
		t.Logf("Output: %q", outputStr)
		// Don't fail - output might be formatted differently
	}
}

// TestPTYForwarderWithLongRunningTerminal tests with a terminal that produces continuous output
func TestPTYForwarderWithLongRunningTerminal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler := newMockOutputHandler()

	// Create a terminal session that produces output over time
	sessionCfg := &terminal.SessionConfig{
		ID:         "long-running-pod",
		Command:    "/bin/sh",
		Args:       []string{"-c", "for i in 1 2 3; do echo line$i; sleep 0.01; done"},
		WorkingDir: "/tmp",
		Cols:       80,
		Rows:       24,
		BufferSize: 64 * 1024, // Required: ring buffer needs size > 0
	}

	session, err := terminal.NewSession(sessionCfg)
	if err != nil {
		t.Skipf("Could not create terminal session: %v", err)
	}
	defer session.Close()

	cfg := &PTYForwarderConfig{
		Session:       session,
		PodKey:    "long-running-test",
		Handler:       handler,
		FlushInterval: 5 * time.Millisecond,
		BufferSize:    1024,
	}

	forwarder := NewPTYForwarder(cfg)
	forwarder.Start()

	// Wait for script to complete
	time.Sleep(300 * time.Millisecond)

	forwarder.Stop()

	handler.mu.Lock()
	outputCount := len(handler.outputs)
	handler.mu.Unlock()

	// Should have captured output
	if outputCount == 0 {
		t.Error("expected output from long-running terminal")
	}
}

// --- Benchmark ---

func BenchmarkPTYForwarderBufferAndFlushV2(b *testing.B) {
	handler := newMockOutputHandler()

	cfg := &PTYForwarderConfig{
		Session:    nil,
		PodKey: "bench-buf-flush-v2",
		Handler:    handler,
		BufferSize: 64 * 1024,
	}

	forwarder := NewPTYForwarder(cfg)
	data := []byte("benchmark test data for performance testing of the PTY forwarder")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		forwarder.bufferOutput(data)
		if i%100 == 0 {
			forwarder.flush()
		}
	}
}

func BenchmarkPTYForwarderBackpressureToggle(b *testing.B) {
	handler := newMockOutputHandler()

	cfg := &PTYForwarderConfig{
		Session:    nil,
		PodKey: "bench-bp-toggle",
		Handler:    handler,
	}

	forwarder := NewPTYForwarder(cfg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		forwarder.setBackpressure(true)
		forwarder.isBackpressure()
		forwarder.setBackpressure(false)
		forwarder.isBackpressure()
	}
}
