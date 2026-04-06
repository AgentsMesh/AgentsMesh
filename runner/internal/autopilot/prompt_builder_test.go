package autopilot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPromptBuilder_NewPromptBuilder(t *testing.T) {
	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt:       "Test task",
		CustomTemplate:      "",
		MCPPort:             19000,
		PodKey:              "worker-123",
		GetMaxIterations:    func() int { return 10 },
		GetCurrentIteration: func() int { return 1 },
	})

	assert.NotNil(t, pb)
}

func TestPromptBuilder_DefaultMCPPort(t *testing.T) {
	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt: "Test task",
		MCPPort:       0, // Should use default
		PodKey:        "worker-123",
	})

	prompt := pb.BuildPrompt()
	// Should contain MCP tool instructions (port is no longer in prompt since we use MCP config file)
	assert.Contains(t, prompt, "get_pod_snapshot")
	assert.Contains(t, prompt, "worker-123")
}

func TestPromptBuilder_BuildPrompt(t *testing.T) {
	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt: "Implement user authentication",
		MCPPort:       19000,
		PodKey:        "worker-123",
	})

	prompt := pb.BuildPrompt()

	// Should contain the task
	assert.Contains(t, prompt, "Implement user authentication")

	// Should contain JSON decision types (new format)
	assert.Contains(t, prompt, `"completed"`)
	assert.Contains(t, prompt, `"continue"`)
	assert.Contains(t, prompt, `"need_help"`)
	assert.Contains(t, prompt, `"give_up"`)

	// Should contain MCP tool instructions with pod key
	assert.Contains(t, prompt, "worker-123")
	assert.Contains(t, prompt, "get_pod_snapshot")
	assert.Contains(t, prompt, "send_pod_input")
	assert.Contains(t, prompt, "get_pod_status")
}

func TestPromptBuilder_BuildPrompt_CustomTemplate(t *testing.T) {
	customTemplate := "Custom template: {{task}}"
	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt:  "Test task",
		CustomTemplate: customTemplate,
		MCPPort:        19000,
		PodKey:         "worker-123",
	})

	prompt := pb.BuildPrompt()

	// Should use custom template verbatim
	assert.Equal(t, customTemplate, prompt)
}

func TestPromptBuilder_BuildResumePrompt(t *testing.T) {
	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt:    "Test task",
		MCPPort:          19000,
		PodKey:           "worker-123",
		GetMaxIterations: func() int { return 10 },
	})

	prompt := pb.BuildResumePrompt(5)

	// Should contain iteration info
	assert.Contains(t, prompt, "5")
	assert.Contains(t, prompt, "10")

	// Should contain instructions
	assert.Contains(t, prompt, "观察 Pod 终端")
	assert.Contains(t, prompt, "判断任务是否完成")
}

func TestPromptBuilder_BuildResumePrompt_NilGetMaxIterations(t *testing.T) {
	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt:    "Test task",
		MCPPort:          19000,
		PodKey:           "worker-123",
		GetMaxIterations: nil, // Should use default of 10
	})

	prompt := pb.BuildResumePrompt(5)

	// Should use default max iterations (10)
	assert.Contains(t, prompt, "5")
	assert.Contains(t, prompt, "10")
}

func TestPromptBuilder_BuildResumePrompt_DifferentIterations(t *testing.T) {
	currentIter := 3
	maxIter := 20

	pb := NewPromptBuilder(PromptBuilderConfig{
		Prompt:       "Test task",
		MCPPort:             19000,
		PodKey:              "worker-123",
		GetMaxIterations:    func() int { return maxIter },
		GetCurrentIteration: func() int { return currentIter },
	})

	prompt := pb.BuildResumePrompt(currentIter)

	assert.Contains(t, prompt, "3")
	assert.Contains(t, prompt, "20")
}
