package runner

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

type orderedStreamPTY struct {
	readStarted chan struct{}
	releaseRead chan struct{}
	releaseWait chan struct{}
	readOnce    sync.Once
	waitOnce    sync.Once
	onResize    func()
}

func newOrderedStreamPTY() *orderedStreamPTY {
	return &orderedStreamPTY{
		readStarted: make(chan struct{}),
		releaseRead: make(chan struct{}),
		releaseWait: make(chan struct{}),
	}
}

func (p *orderedStreamPTY) Read(dst []byte) (int, error) {
	first := false
	p.readOnce.Do(func() {
		first = true
		close(p.readStarted)
	})
	if !first {
		return 0, io.EOF
	}
	<-p.releaseRead
	return copy(dst, []byte("tail")), io.EOF
}

func (p *orderedStreamPTY) Write(data []byte) (int, error)  { return len(data), nil }
func (p *orderedStreamPTY) Close() error                    { return nil }
func (p *orderedStreamPTY) GetSize() (int, int, error)      { return 80, 24, nil }
func (p *orderedStreamPTY) Pid() int                        { return 42 }
func (p *orderedStreamPTY) SetReadDeadline(time.Time) error { return nil }
func (p *orderedStreamPTY) Wait() (int, error)              { <-p.releaseWait; return 0, nil }
func (p *orderedStreamPTY) Kill() error                     { p.releaseProcess(); return nil }
func (p *orderedStreamPTY) GracefulStop() error             { p.releaseProcess(); return nil }

func (p *orderedStreamPTY) Resize(int, int) error {
	if p.onResize != nil {
		p.onResize()
	}
	return nil
}

func (p *orderedStreamPTY) releaseProcess() {
	p.waitOnce.Do(func() { close(p.releaseWait) })
}

func assembleOrderedRuntime(t *testing.T, proc *orderedStreamPTY, notify func(int, []string)) (*Pod, *PTYComponents) {
	t.Helper()
	term, err := terminal.New(terminal.Options{
		Command: "fake",
		PTYFactory: func(string, []string, string, []string, int, int) (terminal.PtyProcess, error) {
			return proc, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	pod := &Pod{PodKey: "ordered-stream"}
	virtualTerm := vt.NewVirtualTerminal(80, 24, 100)
	agg := aggregator.NewSmartAggregator(nil)
	pod.installRuntime(assemblePTYRuntime(pod, term, virtualTerm, agg, nil, nil))
	term.SetOutputHandler(NewPTYOutputHandler(pod.PodKey, pod.IO.(*PTYPodIO).components, notify))
	if err := pod.IO.Start(); err != nil {
		t.Fatal(err)
	}
	return pod, pod.IO.(*PTYPodIO).components
}

func TestPTYResizeCannotOvertakeOwnedRead(t *testing.T) {
	proc := newOrderedStreamPTY()
	var mu sync.Mutex
	events := make([]string, 0, 2)
	record := func(event string) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}
	proc.onResize = func() { record("resize") }
	pod, comps := assembleOrderedRuntime(t, proc, func(int, []string) { record("feed") })
	defer func() {
		pod.IO.Stop()
		pod.IO.Teardown()
	}()

	select {
	case <-proc.readStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("PTY read did not start")
	}
	if comps.readMu.TryLock() {
		comps.readMu.Unlock()
		close(proc.releaseRead)
		t.Fatal("read transaction did not hold readMu")
	}
	resizeStarted := make(chan struct{})
	resizeDone := make(chan error, 1)
	go func() {
		close(resizeStarted)
		_, err := pod.IO.(TerminalAccess).Resize(100, 30)
		resizeDone <- err
	}()
	<-resizeStarted
	var resizeErr error
	resizeReturnedEarly := false
	select {
	case resizeErr = <-resizeDone:
		resizeReturnedEarly = true
	case <-time.After(50 * time.Millisecond):
	}
	close(proc.releaseRead)
	if !resizeReturnedEarly {
		select {
		case resizeErr = <-resizeDone:
		case <-time.After(2 * time.Second):
			t.Fatal("resize did not finish after read transaction")
		}
	}
	if resizeReturnedEarly {
		t.Error("resize overtook an in-flight PTY read")
	}
	if resizeErr != nil {
		t.Fatal(resizeErr)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 2 || events[0] != "feed" || events[1] != "resize" {
		t.Fatalf("event order = %v, want [feed resize]", events)
	}
}

func TestPTYTeardownCannotOvertakeTailHandler(t *testing.T) {
	proc := newOrderedStreamPTY()
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	pod, comps := assembleOrderedRuntime(t, proc, func(int, []string) {
		close(handlerStarted)
		<-releaseHandler
	})
	defer pod.IO.Stop()

	close(proc.releaseRead)
	select {
	case <-handlerStarted:
	case <-time.After(2 * time.Second):
		close(releaseHandler)
		t.Fatal("tail output handler did not start")
	}
	teardownDone := make(chan struct{})
	go func() {
		pod.IO.Teardown()
		close(teardownDone)
	}()
	teardownReturnedEarly := false
	select {
	case <-teardownDone:
		teardownReturnedEarly = true
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseHandler)
	if !teardownReturnedEarly {
		select {
		case <-teardownDone:
		case <-time.After(2 * time.Second):
			t.Fatal("teardown did not finish after output handler")
		}
	}
	if teardownReturnedEarly {
		t.Error("teardown overtook the tail output handler")
	}
	if got := comps.VirtualTerminal.GetOutput(1); got == "" {
		t.Fatal("tail output was not fed before teardown")
	}
}
