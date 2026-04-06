package detector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Claude Code specific scenario tests

func TestClaude_Prompt(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 500, []string{
		"╭──────────────────────────────────────────────────────╮",
		"│  Claude Code                                         │",
		"│  Ready to help with your coding tasks.              │",
		"╰──────────────────────────────────────────────────────╯",
		"",
		">",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Claude Code '>' prompt should be Waiting")
}

func TestClaude_ExecutingTask(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateContinuousOutput(d, 5, 100, []string{
		"Working on task...",
		"Reading file: src/main.go",
		"Analyzing code structure...",
	})

	assert.Equal(t, StateExecuting, d.GetState(), "Continuous output should be Executing")
}

func TestClaude_PermissionRequest(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 100, []string{
		"╭─ Allow Edit ──────────────────────────────────────────╮",
		"│  File: src/auth/login.go                              │",
		"│  [Allow] [Deny] [Allow All]                          │",
		"╰───────────────────────────────────────────────────────╯",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Permission request should be Waiting")
}

func TestClaude_YesNoConfirmation(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 150, []string{
		"This will delete 15 files.",
		"",
		"Do you want to continue? (y/n)",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "y/n confirmation should be Waiting")
}

func TestClaude_ThinkingPhase(t *testing.T) {
	d := NewMultiSignalDetector(MultiSignalConfig{
		IdleThreshold:    100 * time.Millisecond,
		ConfirmThreshold: 100 * time.Millisecond,
		MinStableTime:    100 * time.Millisecond,
		WaitingThreshold: 0.6,
	})

	d.OnOutput(100)
	d.OnScreenUpdate([]string{"Understanding your request...", "", "⠋ Thinking..."})
	time.Sleep(50 * time.Millisecond)
	d.OnScreenUpdate([]string{"Understanding your request...", "", "⠙ Thinking..."})

	assert.Equal(t, StateExecuting, d.DetectState(), "Thinking with spinner should be Executing")
}

func TestClaude_TaskComplete(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 200, []string{
		"✓ Successfully created src/utils/helper.go",
		"✓ Added tests in src/utils/helper_test.go",
		"",
		"Task completed!",
		"",
		">",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Task complete with '>' should be Waiting")
}

func TestClaude_BoxPrompt(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 200, []string{
		"│ What would you like me to help you with?             │",
		"│                                                      │",
		"│ >                                                    │",
		"╰──────────────────────────────────────────────────────╯",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Box prompt should be Waiting")
}

func TestClaude_ToolApproval(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 300, []string{
		"╭─ Tool Use Request ────────────────────────────────────╮",
		"│  Claude wants to use: Write File                      │",
		"│  Allow this tool use?                                 │",
		"│  [y] Yes  [n] No  [a] Always allow                   │",
		"╰───────────────────────────────────────────────────────╯",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Tool approval should be Waiting")
}
