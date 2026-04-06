package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// OpenCode specific scenario tests
// OpenCode has a TUI interface similar to Claude Code

func TestOpenCode_Prompt(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 300, []string{
		"┌─────────────────────────────────────────────────────────┐",
		"│ OpenCode - AI Coding Assistant                          │",
		"└─────────────────────────────────────────────────────────┘",
		"",
		"What would you like to build today?",
		"",
		"❯ ",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "OpenCode '❯' prompt should be Waiting")
}

func TestOpenCode_AnalyzingCode(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateContinuousOutput(d, 5, 100, []string{
		"❯ Analyze this codebase",
		"",
		"🔍 Scanning files...",
		"   Found 42 source files",
		"   Analyzing dependencies...",
	})

	assert.Equal(t, StateExecuting, d.GetState(), "Analyzing should be Executing")
}

func TestOpenCode_FileEditConfirm(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 250, []string{
		"┌─ Proposed Changes ──────────────────────────────────────┐",
		"│ File: src/api/handler.go                                │",
		"│ +15 lines, -3 lines                                     │",
		"└─────────────────────────────────────────────────────────┘",
		"",
		"Apply these changes? [y/n/d(iff)]:",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Edit confirm should be Waiting")
}

func TestOpenCode_RunningTask(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateContinuousOutput(d, 4, 80, []string{
		"⚡ Running task: Build project",
		"",
		"   [1/3] Compiling...",
		"   [2/3] Linking...",
	})

	assert.Equal(t, StateExecuting, d.GetState(), "Running task should be Executing")
}

func TestOpenCode_TaskComplete(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 200, []string{
		"✅ Task completed successfully!",
		"",
		"   Created: src/new_feature.go",
		"   Modified: src/main.go",
		"",
		"❯ ",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Task complete should be Waiting")
}

func TestOpenCode_MultiStepWizard(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 200, []string{
		"Step 2/4: Configure database",
		"",
		"Select database type:",
		"  (1) PostgreSQL",
		"  (2) MySQL",
		"  (3) SQLite",
		"",
		"Enter choice [1-3]:",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Wizard step should be Waiting")
}

func TestOpenCode_ErrorWithRetry(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 200, []string{
		"❌ Error: Failed to connect to API",
		"",
		"Retry? [y/n]:",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Error retry should be Waiting")
}
