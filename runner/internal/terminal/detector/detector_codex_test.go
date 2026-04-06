package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Codex CLI (OpenAI) specific scenario tests
// Codex CLI has a minimalist interface with simple prompts

func TestCodex_Prompt(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 200, []string{
		"Codex CLI v1.0.0",
		"",
		"Enter your coding request:",
		"> ",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Codex '>' prompt should be Waiting")
}

func TestCodex_GeneratingCode(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateContinuousOutput(d, 5, 100, []string{
		"> Create a binary search function",
		"",
		"Generating code...",
		"[████████░░░░░░░░░░░░] 40%",
	})

	assert.Equal(t, StateExecuting, d.GetState(), "Code generation should be Executing")
}

func TestCodex_CodeComplete(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 400, []string{
		"Generated code:",
		"```python",
		"def binary_search(arr, target):",
		"    left, right = 0, len(arr) - 1",
		"    while left <= right:",
		"        mid = (left + right) // 2",
		"        if arr[mid] == target:",
		"            return mid",
		"    return -1",
		"```",
		"",
		"> ",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "After code complete should be Waiting")
}

func TestCodex_ApplyChanges(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 200, []string{
		"Ready to apply changes to:",
		"  src/search.py",
		"",
		"Apply changes? (y/n): ",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Apply changes prompt should be Waiting")
}

func TestCodex_ExecutingCommand(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateContinuousOutput(d, 4, 80, []string{
		"Running: python -m pytest tests/",
		"",
		"test_search.py::test_binary_search PASSED",
		"test_search.py::test_edge_cases PASSED",
	})

	assert.Equal(t, StateExecuting, d.GetState(), "Running tests should be Executing")
}

func TestCodex_ErrorRecovery(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 300, []string{
		"Error: SyntaxError in generated code",
		"",
		"Would you like me to fix this? (y/n): ",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Error recovery prompt should be Waiting")
}

func TestCodex_FileSelection(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 250, []string{
		"Select files to modify:",
		"  [1] src/main.py",
		"  [2] src/utils.py",
		"  [3] tests/test_main.py",
		"",
		"Enter selection (1-3): ",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "File selection should be Waiting")
}
