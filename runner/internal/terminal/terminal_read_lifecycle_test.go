package terminal

import (
	"io"
	"sync"
	"testing"
	"time"
)

type lifecyclePTY struct {
	payload       []byte
	readStarted   chan struct{}
	releaseRead   chan struct{}
	releaseWait   chan struct{}
	readStartOnce sync.Once
	closeOnce     sync.Once
	mu            sync.Mutex
	closed        bool
}

type gracefulLifecyclePTY struct {
	*lifecyclePTY
	gracefulOnce sync.Once
	graceful     chan struct{}
}

func newGracefulLifecyclePTY(payload string) *gracefulLifecyclePTY {
	return &gracefulLifecyclePTY{
		lifecyclePTY: newLifecyclePTY(payload),
		graceful:     make(chan struct{}),
	}
}

func (p *gracefulLifecyclePTY) GracefulStop() error {
	p.gracefulOnce.Do(func() {
		close(p.graceful)
		close(p.releaseWait)
		close(p.releaseRead)
	})
	return nil
}

type progressiveTailPTY struct {
	mu        sync.Mutex
	chunks    [][]byte
	delay     time.Duration
	closed    bool
	readStart sync.Once
	started   chan struct{}
}

func newProgressiveTailPTY(chunks ...string) *progressiveTailPTY {
	data := make([][]byte, 0, len(chunks))
	for _, chunk := range chunks {
		data = append(data, []byte(chunk))
	}
	return &progressiveTailPTY{
		chunks:  data,
		delay:   50 * time.Millisecond,
		started: make(chan struct{}),
	}
}

func (p *progressiveTailPTY) Read(dst []byte) (int, error) {
	p.readStart.Do(func() { close(p.started) })
	time.Sleep(p.delay)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || len(p.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(dst, p.chunks[0])
	p.chunks = p.chunks[1:]
	return n, nil
}

func (p *progressiveTailPTY) Write(data []byte) (int, error)  { return len(data), nil }
func (p *progressiveTailPTY) Resize(int, int) error           { return nil }
func (p *progressiveTailPTY) GetSize() (int, int, error)      { return 80, 24, nil }
func (p *progressiveTailPTY) Pid() int                        { return 43 }
func (p *progressiveTailPTY) SetReadDeadline(time.Time) error { return nil }
func (p *progressiveTailPTY) Wait() (int, error)              { return 0, nil }
func (p *progressiveTailPTY) Kill() error                     { return nil }
func (p *progressiveTailPTY) GracefulStop() error             { return nil }
func (p *progressiveTailPTY) Close() error {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()
	return nil
}

func newLifecyclePTY(payload string) *lifecyclePTY {
	return &lifecyclePTY{
		payload:     []byte(payload),
		readStarted: make(chan struct{}),
		releaseRead: make(chan struct{}),
		releaseWait: make(chan struct{}),
	}
}

func (p *lifecyclePTY) Read(dst []byte) (int, error) {
	p.readStartOnce.Do(func() { close(p.readStarted) })
	<-p.releaseRead
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed || len(p.payload) == 0 {
		return 0, io.EOF
	}
	n := copy(dst, p.payload)
	p.payload = nil
	return n, io.EOF
}

func (p *lifecyclePTY) Write(data []byte) (int, error)  { return len(data), nil }
func (p *lifecyclePTY) Resize(int, int) error           { return nil }
func (p *lifecyclePTY) GetSize() (int, int, error)      { return 80, 24, nil }
func (p *lifecyclePTY) Pid() int                        { return 42 }
func (p *lifecyclePTY) SetReadDeadline(time.Time) error { return nil }
func (p *lifecyclePTY) Kill() error                     { return nil }
func (p *lifecyclePTY) GracefulStop() error             { return nil }

func (p *lifecyclePTY) Wait() (int, error) {
	<-p.releaseWait
	return 0, nil
}

func (p *lifecyclePTY) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()
	})
	return nil
}

func (p *lifecyclePTY) isClosed() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.closed
}

func TestNaturalExitDeliversTailBeforeExitHandler(t *testing.T) {
	proc := newLifecyclePTY("tail")
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	exited := make(chan int, 1)
	var output []byte

	term, err := New(Options{
		Command: "fake",
		PTYFactory: func(string, []string, string, []string, int, int) (PtyProcess, error) {
			return proc, nil
		},
		OnOutput: func(data []byte) {
			output = append(output, data...)
			close(handlerStarted)
			<-releaseHandler
		},
		OnExit: func(code int) { exited <- code },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := term.Start(); err != nil {
		t.Fatal(err)
	}
	<-proc.readStarted
	close(proc.releaseWait)
	close(proc.releaseRead)
	<-handlerStarted

	select {
	case <-exited:
		t.Fatal("exit handler overtook output handler")
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseHandler)

	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Fatal("exit handler did not run")
	}
	if got := string(output); got != "tail" {
		t.Fatalf("output = %q, want tail", got)
	}
}

func TestFastExitWaitsForReaderToOwnPTY(t *testing.T) {
	proc := newLifecyclePTY("fast-output")
	close(proc.releaseRead)
	close(proc.releaseWait)
	output := make(chan string, 1)
	exited := make(chan struct{}, 1)

	term, err := New(Options{
		Command: "fake",
		PTYFactory: func(string, []string, string, []string, int, int) (PtyProcess, error) {
			return proc, nil
		},
		OnOutput: func(data []byte) { output <- string(data) },
		OnExit:   func(int) { exited <- struct{}{} },
	})
	if err != nil {
		t.Fatal(err)
	}

	var readerGate sync.Mutex
	readerGate.Lock()
	term.SetReadOrderLocker(&readerGate)
	if err := term.Start(); err != nil {
		readerGate.Unlock()
		t.Fatal(err)
	}
	time.Sleep(readDrainGrace + 50*time.Millisecond)
	if proc.isClosed() {
		t.Error("PTY closed before reader acquired its first read transaction")
	}
	readerGate.Unlock()

	select {
	case got := <-output:
		if got != "fast-output" {
			t.Fatalf("output = %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fast process output was lost")
	}
	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Fatal("fast process exit handler did not run")
	}
}

func TestStopDrainsOutputProducedByGracefulShutdown(t *testing.T) {
	proc := newGracefulLifecyclePTY("shutdown-tail")
	var output []byte
	var outputMu sync.Mutex
	term, err := New(Options{
		Command: "fake",
		PTYFactory: func(string, []string, string, []string, int, int) (PtyProcess, error) {
			return proc, nil
		},
		OnOutput: func(data []byte) {
			outputMu.Lock()
			output = append(output, data...)
			outputMu.Unlock()
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var readerGate sync.Mutex
	readerGate.Lock()
	term.SetReadOrderLocker(&readerGate)
	if err := term.Start(); err != nil {
		readerGate.Unlock()
		t.Fatal(err)
	}
	stopped := make(chan struct{})
	go func() {
		term.Stop()
		close(stopped)
	}()
	select {
	case <-proc.graceful:
	case <-time.After(2 * time.Second):
		readerGate.Unlock()
		t.Fatal("graceful stop was not requested")
	}
	readerGate.Unlock()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("terminal stop did not finish")
	}

	outputMu.Lock()
	got := string(output)
	outputMu.Unlock()
	if got != "shutdown-tail" {
		t.Fatalf("output = %q, want graceful shutdown tail", got)
	}
}

func TestNaturalExitDrainExtendsWhileReadsMakeProgress(t *testing.T) {
	proc := newProgressiveTailPTY("a", "b", "c", "d", "e", "f", "g", "h")
	var output []byte
	var outputMu sync.Mutex
	exited := make(chan struct{})
	term, err := New(Options{
		Command: "fake",
		PTYFactory: func(string, []string, string, []string, int, int) (PtyProcess, error) {
			return proc, nil
		},
		OnOutput: func(data []byte) {
			outputMu.Lock()
			output = append(output, data...)
			outputMu.Unlock()
		},
		OnExit: func(int) { close(exited) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := term.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-proc.started:
	case <-time.After(2 * time.Second):
		t.Fatal("reader did not start")
	}
	select {
	case <-exited:
	case <-time.After(3 * time.Second):
		t.Fatal("natural exit did not finish")
	}

	outputMu.Lock()
	got := string(output)
	outputMu.Unlock()
	if got != "abcdefgh" {
		t.Fatalf("output = %q, want all progressive tail chunks", got)
	}
}
