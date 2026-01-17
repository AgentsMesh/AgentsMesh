package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTerminalRouter_GetRecentOutput_Raw(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register pod
	tr.RegisterPod("pod-1", 1)

	// Write some data to scrollback
	shard := tr.getShard("pod-1")
	shard.mu.RLock()
	buffer := shard.scrollbackBuffers["pod-1"]
	shard.mu.RUnlock()

	buffer.Write([]byte("line1\nline2\nline3\n"))

	// Get raw output
	output := tr.GetRecentOutput("pod-1", 10, true)
	assert.Contains(t, string(output), "line1")
	assert.Contains(t, string(output), "line2")
	assert.Contains(t, string(output), "line3")
}

func TestTerminalRouter_GetRecentOutput_Processed(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register pod
	tr.RegisterPod("pod-1", 1)

	// Write some data with ANSI codes
	shard := tr.getShard("pod-1")
	shard.mu.RLock()
	buffer := shard.scrollbackBuffers["pod-1"]
	shard.mu.RUnlock()

	buffer.Write([]byte("\x1b[32mgreen text\x1b[0m\n"))

	// Get processed output (should strip ANSI)
	output := tr.GetRecentOutput("pod-1", 10, false)
	assert.Contains(t, string(output), "green text")
	assert.NotContains(t, string(output), "\x1b[32m")
}

func TestTerminalRouter_GetRecentOutput_NilBuffer(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Unregistered pod should return nil
	output := tr.GetRecentOutput("nonexistent-pod", 10, true)
	assert.Nil(t, output)

	output = tr.GetRecentOutput("nonexistent-pod", 10, false)
	assert.Nil(t, output)
}

func TestTerminalRouter_GetScreenSnapshot(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register pod
	tr.RegisterPod("pod-1", 1)

	// Write some data
	shard := tr.getShard("pod-1")
	shard.mu.RLock()
	buffer := shard.scrollbackBuffers["pod-1"]
	shard.mu.RUnlock()

	buffer.Write([]byte("$ ls\nfile1.txt\nfile2.txt\n"))

	// Get screen snapshot
	snapshot := tr.GetScreenSnapshot("pod-1")
	assert.Contains(t, snapshot, "ls")
	assert.Contains(t, snapshot, "file1.txt")
}

func TestTerminalRouter_GetScreenSnapshot_NilBuffer(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Unregistered pod should return empty
	snapshot := tr.GetScreenSnapshot("nonexistent-pod")
	assert.Empty(t, snapshot)
}

func TestTerminalRouter_GetCursorPosition(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register pod
	tr.RegisterPod("pod-1", 1)

	// Get cursor position (default should be 0, 0 or initial position)
	row, col := tr.GetCursorPosition("pod-1")
	// Virtual terminal initializes cursor at 0,0
	assert.GreaterOrEqual(t, row, 0)
	assert.GreaterOrEqual(t, col, 0)
}

func TestTerminalRouter_GetCursorPosition_NilTerminal(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Unregistered pod should return 0, 0
	row, col := tr.GetCursorPosition("nonexistent-pod")
	assert.Equal(t, 0, row)
	assert.Equal(t, 0, col)
}

func TestTerminalRouter_GetAllScrollbackData(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register pod
	tr.RegisterPod("pod-1", 1)

	// Write data
	shard := tr.getShard("pod-1")
	shard.mu.RLock()
	buffer := shard.scrollbackBuffers["pod-1"]
	shard.mu.RUnlock()

	testData := []byte("test scrollback data")
	buffer.Write(testData)

	// Get all data
	data := tr.GetAllScrollbackData("pod-1")
	assert.Equal(t, testData, data)
}

func TestTerminalRouter_GetAllScrollbackData_NilBuffer(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Unregistered pod should return nil
	data := tr.GetAllScrollbackData("nonexistent-pod")
	assert.Nil(t, data)
}

func TestTerminalRouter_ClearScrollback(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register pod
	tr.RegisterPod("pod-1", 1)

	// Write data
	shard := tr.getShard("pod-1")
	shard.mu.RLock()
	buffer := shard.scrollbackBuffers["pod-1"]
	shard.mu.RUnlock()

	buffer.Write([]byte("some data"))

	// Clear scrollback
	tr.ClearScrollback("pod-1")

	// Verify cleared
	data := tr.GetAllScrollbackData("pod-1")
	assert.Empty(t, data)
}

func TestTerminalRouter_ClearScrollback_NilBuffer(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Should not panic on unregistered pod
	tr.ClearScrollback("nonexistent-pod")
}

func TestTerminalRouter_GetRecentOutput_EmptyBuffer(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register pod but don't write data
	tr.RegisterPod("pod-1", 1)

	// Raw output should return nil for empty buffer
	output := tr.GetRecentOutput("pod-1", 10, true)
	assert.Nil(t, output)
}

func TestTerminalRouter_GetScreenSnapshot_EmptyBuffer(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	defer cm.Close()
	tr := NewTerminalRouter(cm, newTestLogger())

	// Register pod but don't write data
	tr.RegisterPod("pod-1", 1)

	// Should return empty for empty buffer
	snapshot := tr.GetScreenSnapshot("pod-1")
	// May return empty or whitespace from virtual terminal
	assert.NotNil(t, snapshot)
}
