package autopilot

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
)

// Tests for prompt building

func TestBuildPrompt_Default(t *testing.T) {
	config := &runnerv1.AutopilotConfig{
		Prompt: "Implement user authentication",
		MaxIterations: 10,
	}

	workerCtrl := &MockPodController{
		workDir: t.TempDir(),
		podKey:  "worker-123",
	}

	rp := NewAutopilotController(Config{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		ProtoConfig:  config,
		PodCtrl:      workerCtrl,
		Reporter:     &MockEventReporter{},
		MCPPort:      19000,
	})

	prompt := rp.buildPrompt()

	// Should contain the original task
	assert.Contains(t, prompt, "Implement user authentication")
	// Should contain the JSON decision type instructions (new format)
	assert.Contains(t, prompt, `"completed"`)
	assert.Contains(t, prompt, `"continue"`)
	assert.Contains(t, prompt, `"need_help"`)
	assert.Contains(t, prompt, `"give_up"`)
	// Should contain MCP tool instructions
	assert.Contains(t, prompt, "get_pod_snapshot")
	assert.Contains(t, prompt, "send_pod_input")
	assert.Contains(t, prompt, "get_pod_status")
	// Should contain pod key
	assert.Contains(t, prompt, "worker-123")
	// Should contain the important restrictions
	assert.Contains(t, prompt, "重要限制")
	assert.Contains(t, prompt, "你不能直接完成任务")
}

func TestBuildPrompt_CustomTemplate(t *testing.T) {
	customTemplate := "Custom prompt template for {{task}}"
	config := &runnerv1.AutopilotConfig{
		Prompt:                "Test task",
		ControlPromptTemplate: customTemplate,
	}

	workerCtrl := &MockPodController{
		workDir: t.TempDir(),
		podKey:  "worker-123",
	}

	rp := NewAutopilotController(Config{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		ProtoConfig:  config,
		PodCtrl:      workerCtrl,
		Reporter:     &MockEventReporter{},
	})

	prompt := rp.buildPrompt()

	// Should use custom template
	assert.Equal(t, customTemplate, prompt)
}

func TestBuildResumePrompt(t *testing.T) {
	config := &runnerv1.AutopilotConfig{
		Prompt: "Test task",
		MaxIterations: 10,
	}

	workerCtrl := &MockPodController{
		workDir: t.TempDir(),
		podKey:  "worker-123",
	}

	rp := NewAutopilotController(Config{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		ProtoConfig:  config,
		PodCtrl:      workerCtrl,
		Reporter:     &MockEventReporter{},
	})

	prompt := rp.buildResumePrompt(5)

	// Should contain iteration info
	assert.Contains(t, prompt, "5")
	assert.Contains(t, prompt, "10")
	// Should contain instructions
	assert.Contains(t, prompt, "观察 Pod 终端")
	assert.Contains(t, prompt, "判断任务是否完成")
}
