package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Aider specific scenario tests
// Aider is a popular CLI coding assistant with distinctive prompts

func TestAider_Prompt(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 400, []string{
		"Aider v0.50.0",
		"Main model: gpt-4-turbo with diff edit format",
		"Git repo: /home/user/project",
		"Repo-map: using 1024 tokens",
		"",
		"aider> ",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Aider 'aider>' prompt should be Waiting")
}

func TestAider_AddingFiles(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 200, []string{
		"aider> /add src/main.py",
		"",
		"Added src/main.py to the chat.",
		"",
		"aider> ",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "After adding files should be Waiting")
}

func TestAider_GeneratingDiff(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateContinuousOutput(d, 5, 100, []string{
		"aider> Add error handling to the API",
		"",
		"Thinking...",
		"",
		"src/api.py",
		"<<<<<<< SEARCH",
	})

	assert.Equal(t, StateExecuting, d.GetState(), "Generating diff should be Executing")
}

func TestAider_ApplyDiff(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 300, []string{
		"src/api.py",
		"<<<<<<< SEARCH",
		"def handle_request():",
		"=======",
		"def handle_request():",
		"    try:",
		">>>>>>> REPLACE",
		"",
		"Apply this change? [y/n/a(ll)/d(on't)]:",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Apply diff prompt should be Waiting")
}

func TestAider_RunningTests(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateContinuousOutput(d, 4, 80, []string{
		"aider> /test",
		"",
		"Running: pytest tests/ -v",
		"===== test session starts =====",
	})

	assert.Equal(t, StateExecuting, d.GetState(), "Running tests should be Executing")
}

func TestAider_CommitConfirm(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 200, []string{
		"Commit message: Add error handling to API",
		"",
		"Commit this change? [y/n]:",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Commit confirm should be Waiting")
}

func TestAider_UndoPrompt(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 200, []string{
		"aider> /undo",
		"",
		"This will undo the last commit:",
		"  abc1234 - Add error handling",
		"",
		"Proceed? [y/n]:",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Undo prompt should be Waiting")
}
