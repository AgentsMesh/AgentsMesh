package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Gemini CLI specific scenario tests
// Gemini CLI uses a different UI style with colored output and specific prompts

func TestGemini_Prompt(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 300, []string{
		"Welcome to Gemini CLI!",
		"",
		"Type your message or /help for commands.",
		"",
		">>> ",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Gemini '>>>' prompt should be Waiting")
}

func TestGemini_ProcessingQuery(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateContinuousOutput(d, 5, 80, []string{
		">>> How do I create a REST API?",
		"",
		"[Thinking...]",
		"",
		"Processing your request...",
	})

	assert.Equal(t, StateExecuting, d.GetState(), "Processing should be Executing")
}

func TestGemini_StreamingResponse(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	// Simulate streaming response
	responses := []string{
		"Here's how to create a REST API:\n",
		"1. First, set up your project structure\n",
		"2. Install the required dependencies\n",
	}

	for _, chunk := range responses {
		d.OnOutput(len(chunk))
		d.DetectState()
	}

	assert.Equal(t, StateExecuting, d.GetState(), "Streaming response should be Executing")
}

func TestGemini_WaitingAfterResponse(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 500, []string{
		"Here's how to create a REST API:",
		"1. First, set up your project structure",
		"2. Install the required dependencies",
		"",
		">>> ",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "After response with '>>>' should be Waiting")
}

func TestGemini_CodeExecution(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateContinuousOutput(d, 4, 100, []string{
		"Executing code...",
		"",
		"[Running] python script.py",
		"Output: Hello, World!",
	})

	assert.Equal(t, StateExecuting, d.GetState(), "Code execution should be Executing")
}

func TestGemini_ConfirmAction(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	simulateOutputAndScreen(d, 200, []string{
		"This will modify 3 files:",
		"  - main.py",
		"  - utils.py",
		"  - config.py",
		"",
		"Proceed? [y/N]",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Confirm action should be Waiting")
}

func TestGemini_MultiTurnChat(t *testing.T) {
	d := NewMultiSignalDetector(testDetectorConfig())

	// First turn complete
	d.OnOutput(200)
	d.OnScreenUpdate([]string{
		">>> What is Go?",
		"",
		"Go is a programming language...",
		"",
		">>> ",
	})

	simulateOutputAndScreen(d, 0, []string{
		">>> What is Go?",
		"",
		"Go is a programming language...",
		"",
		">>> ",
	})

	assert.Equal(t, StateWaiting, d.DetectState(), "Multi-turn waiting should be Waiting")
}
