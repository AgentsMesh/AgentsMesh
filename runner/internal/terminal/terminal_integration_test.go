//go:build integration

package terminal

import (
	"bytes"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTerminal_PTYLifecycle_Integration creates a short-lived process,
// verifies PID > 0, waits for exit, and confirms exit handler fires with code 0.
func TestTerminal_PTYLifecycle_Integration(t *testing.T) {
	var exitCode atomic.Int32
	exitCode.Store(-1)
	exitCh := make(chan struct{})

	term, err := New(Options{
		Command: "echo",
		Args:    []string{"lifecycle"},
		WorkDir: t.TempDir(),
		Rows:    24,
		Cols:    80,
		Label:   "lifecycle-test",
	})
	require.NoError(t, err)

	term.SetExitHandler(func(code int) {
		exitCode.Store(int32(code))
		close(exitCh)
	})

	require.NoError(t, term.Start())
	assert.Greater(t, term.PID(), 0, "PID should be positive after Start")

	select {
	case <-exitCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for exit handler")
	}

	assert.Equal(t, int32(0), exitCode.Load(), "echo should exit with code 0")
	term.Stop()
	assert.True(t, term.IsClosed(), "terminal should be closed after Stop")
}

// TestTerminal_OutputHandler_Integration runs `echo "output test"` and
// verifies the output handler receives the expected string.
func TestTerminal_OutputHandler_Integration(t *testing.T) {
	var mu sync.Mutex
	var collected []byte
	outputCh := make(chan struct{}, 1)

	term, err := New(Options{
		Command: "echo",
		Args:    []string{"output test"},
		WorkDir: t.TempDir(),
		Rows:    24,
		Cols:    80,
		Label:   "output-test",
	})
	require.NoError(t, err)

	term.SetOutputHandler(func(data []byte) {
		mu.Lock()
		collected = append(collected, data...)
		if bytes.Contains(collected, []byte("output test")) {
			select {
			case outputCh <- struct{}{}:
			default:
			}
		}
		mu.Unlock()
	})

	require.NoError(t, term.Start())
	defer term.Stop()

	select {
	case <-outputCh:
	case <-time.After(5 * time.Second):
		mu.Lock()
		t.Fatalf("timeout waiting for output; collected so far: %q", string(collected))
		mu.Unlock()
	}
}

// TestTerminal_Resize_Integration starts `cat` (stays alive), resizes the
// terminal, and verifies the operation succeeds without error.
func TestTerminal_Resize_Integration(t *testing.T) {
	term, err := New(Options{
		Command: "cat",
		WorkDir: t.TempDir(),
		Rows:    24,
		Cols:    80,
		Label:   "resize-test",
	})
	require.NoError(t, err)

	// Drain output so the read loop doesn't block
	term.SetOutputHandler(func([]byte) {})

	require.NoError(t, term.Start())
	defer term.Stop()

	// Give the PTY a moment to initialise
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, term.Resize(120, 40), "Resize should succeed on a running terminal")
}

// TestTerminal_InputWrite_Integration starts `cat`, writes data to stdin,
// and reads the echoed output back via the output handler.
func TestTerminal_InputWrite_Integration(t *testing.T) {
	var mu sync.Mutex
	var collected []byte
	echoCh := make(chan struct{}, 1)
	needle := []byte("hello-from-test")

	term, err := New(Options{
		Command: "cat",
		WorkDir: t.TempDir(),
		Rows:    24,
		Cols:    80,
		Label:   "input-test",
	})
	require.NoError(t, err)

	term.SetOutputHandler(func(data []byte) {
		mu.Lock()
		collected = append(collected, data...)
		if bytes.Contains(collected, needle) {
			select {
			case echoCh <- struct{}{}:
			default:
			}
		}
		mu.Unlock()
	})

	require.NoError(t, term.Start())
	defer term.Stop()

	// Give cat a moment to start
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, term.Write(append(needle, '\n')))

	select {
	case <-echoCh:
	case <-time.After(5 * time.Second):
		mu.Lock()
		t.Fatalf("timeout waiting for echo; collected so far: %q", string(collected))
		mu.Unlock()
	}
}
