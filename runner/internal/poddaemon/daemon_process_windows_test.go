//go:build windows

package poddaemon

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartDaemonProcessWindows(t *testing.T) {
	proc, err := startDaemonProcess("cmd.exe", []string{"/c", "echo hello"}, "", nil, 80, 24)
	require.NoError(t, err)
	defer proc.Close()

	assert.Greater(t, proc.Pid(), 0)

	// Read output — ConPTY may include extra whitespace or ANSI sequences.
	buf := make([]byte, 4096)
	var output strings.Builder
	for {
		n, err := proc.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
		}
		if err != nil || strings.Contains(output.String(), "hello") {
			break
		}
	}
	assert.Contains(t, output.String(), "hello")

	exitCode, err := proc.Wait()
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}

func TestStartDaemonProcessNotFound(t *testing.T) {
	_, err := startDaemonProcess("nonexistent-binary-xyz-12345", nil, "", nil, 80, 24)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command not found")
}

func TestWindowsDaemonProcessResize(t *testing.T) {
	// Use cmd.exe to wrap ping — direct ping can crash ConPTY on some CI runners.
	proc, err := startDaemonProcess("cmd.exe", []string{"/c", "ping -n 10 127.0.0.1"}, "", nil, 80, 24)
	require.NoError(t, err)
	defer func() {
		// Kill first, then Close (single cleanup path to avoid double-close).
		_ = proc.Kill()
	}()

	// Give ConPTY time to fully initialize before Resize.
	time.Sleep(200 * time.Millisecond)

	// Resize should not error on a running ConPTY process.
	err = proc.Resize(120, 40)
	assert.NoError(t, err)
}

func TestWindowsDaemonProcessGracefulStop(t *testing.T) {
	proc, err := startDaemonProcess("cmd.exe", []string{"/c", "ping -n 30 127.0.0.1"}, "", nil, 80, 24)
	require.NoError(t, err)
	defer proc.Close()

	// GracefulStop sends Ctrl+C.
	err = proc.GracefulStop()
	assert.NoError(t, err)

	// Process should exit (possibly with non-zero code due to Ctrl+C).
	_, err = proc.Wait()
	// We don't assert on the exit code — Ctrl+C may produce various codes.
	_ = err
}

func TestWindowsDaemonProcessKill(t *testing.T) {
	proc, err := startDaemonProcess("cmd.exe", []string{"/c", "ping -n 30 127.0.0.1"}, "", nil, 80, 24)
	require.NoError(t, err)

	pid := proc.Pid()
	assert.Greater(t, pid, 0)

	err = proc.Kill()
	assert.NoError(t, err)
}
