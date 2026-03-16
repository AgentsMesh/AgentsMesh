//go:build !windows

package poddaemon

import (
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockDaemonProcess is a mock implementation of daemonProcess for testing.
type mockDaemonProcess struct {
	mu sync.Mutex

	readData  []byte // data returned by Read
	readErr   error
	readCh    chan []byte // if set, Read blocks until data arrives
	writeData []byte     // last data written via Write
	writeErr  error

	resizeCols int
	resizeRows int
	resizeErr  error

	pid     int
	waitCh  chan int // blocks Wait() until value sent
	waitErr error

	gracefulStopErr   error
	gracefulStopCount int
	killErr           error
	killCount         int
	closed            bool
}

func newMockProcess(pid int) *mockDaemonProcess {
	return &mockDaemonProcess{
		pid:    pid,
		readCh: make(chan []byte, 16),
		waitCh: make(chan int, 1),
	}
}

func (m *mockDaemonProcess) Read(p []byte) (int, error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	data, ok := <-m.readCh
	if !ok {
		return 0, io.EOF
	}
	n := copy(p, data)
	return n, nil
}

func (m *mockDaemonProcess) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	m.writeData = make([]byte, len(p))
	copy(m.writeData, p)
	return len(p), nil
}

func (m *mockDaemonProcess) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockDaemonProcess) Resize(cols, rows int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resizeCols = cols
	m.resizeRows = rows
	return m.resizeErr
}

func (m *mockDaemonProcess) Pid() int { return m.pid }

func (m *mockDaemonProcess) Wait() (int, error) {
	code := <-m.waitCh
	return code, m.waitErr
}

func (m *mockDaemonProcess) GracefulStop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gracefulStopCount++
	return m.gracefulStopErr
}

func (m *mockDaemonProcess) Kill() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.killCount++
	return m.killErr
}

// shortSocketDir returns a short temp dir suitable for Unix sockets (macOS 104-byte limit).
func shortSocketDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "pd-")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// setupDaemonServer creates a daemonServer with a mock process and IPC listener.
func setupDaemonServer(t *testing.T) (*daemonServer, *mockDaemonProcess, string) {
	t.Helper()
	dir := shortSocketDir(t)
	ipcPath := IPCPath(dir, "t")

	listener, err := Listen(ipcPath)
	require.NoError(t, err)

	proc := newMockProcess(123)

	state := &PodDaemonState{
		PodKey:      "test-pod",
		IPCPath:     ipcPath,
		SandboxPath: dir,
		Cols:        80,
		Rows:        24,
	}

	d := &daemonServer{
		proc:     proc,
		listener: listener,
		exitDone: make(chan struct{}),
		orphanCh: make(chan struct{}),
		log:      slog.Default(),
		state:    state,
	}

	t.Cleanup(func() {
		listener.Close()
	})

	return d, proc, ipcPath
}
