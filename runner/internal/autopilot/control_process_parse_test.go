package autopilot

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
)

// Tests for decision parsing in controller

func TestParseDecision_TaskCompleted(t *testing.T) {
	config := &runnerv1.AutopilotConfig{
		Prompt: "Test",
	}

	workerCtrl := &MockPodController{}

	rp := NewAutopilotController(Config{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		ProtoConfig:  config,
		PodCtrl:      workerCtrl,
		Reporter:     &MockEventReporter{},
	})

	output := `Analysis complete.
TASK_COMPLETED
Successfully implemented the user authentication feature.
All tests passing.`

	decision := rp.parseDecision(output)

	assert.Equal(t, DecisionCompleted, decision.Type)
	assert.Contains(t, decision.Summary, "Successfully implemented")
}

func TestParseDecision_Continue(t *testing.T) {
	config := &runnerv1.AutopilotConfig{
		Prompt: "Test",
	}

	workerCtrl := &MockPodController{}

	rp := NewAutopilotController(Config{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		ProtoConfig:  config,
		PodCtrl:      workerCtrl,
		Reporter:     &MockEventReporter{},
	})

	output := `Current progress: 50%
CONTINUE
Need to implement the login form next.`

	decision := rp.parseDecision(output)

	assert.Equal(t, DecisionContinue, decision.Type)
	assert.Contains(t, decision.Summary, "Need to implement")
}

func TestParseDecision_NeedHumanHelp(t *testing.T) {
	config := &runnerv1.AutopilotConfig{
		Prompt: "Test",
	}

	workerCtrl := &MockPodController{}

	rp := NewAutopilotController(Config{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		ProtoConfig:  config,
		PodCtrl:      workerCtrl,
		Reporter:     &MockEventReporter{},
	})

	output := `I've encountered a problem.
NEED_HUMAN_HELP
The API credentials are missing from the environment.
Cannot proceed without them.`

	decision := rp.parseDecision(output)

	assert.Equal(t, DecisionNeedHumanHelp, decision.Type)
	assert.Contains(t, decision.Summary, "API credentials")
}

func TestParseDecision_GiveUp(t *testing.T) {
	config := &runnerv1.AutopilotConfig{
		Prompt: "Test",
	}

	workerCtrl := &MockPodController{}

	rp := NewAutopilotController(Config{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		ProtoConfig:  config,
		PodCtrl:      workerCtrl,
		Reporter:     &MockEventReporter{},
	})

	output := `After multiple attempts, I cannot complete this task.
GIVE_UP
The codebase architecture is incompatible with the requested feature.
This would require a complete rewrite.`

	decision := rp.parseDecision(output)

	assert.Equal(t, DecisionGiveUp, decision.Type)
	assert.Contains(t, decision.Summary, "codebase architecture")
}

func TestParseDecision_DefaultToContinue(t *testing.T) {
	config := &runnerv1.AutopilotConfig{
		Prompt: "Test",
	}

	workerCtrl := &MockPodController{}

	rp := NewAutopilotController(Config{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		ProtoConfig:  config,
		PodCtrl:      workerCtrl,
		Reporter:     &MockEventReporter{},
	})

	// Output without any decision marker
	output := `Working on the task...
Still processing.`

	decision := rp.parseDecision(output)

	// Should default to CONTINUE
	assert.Equal(t, DecisionContinue, decision.Type)
}

func TestParseDecision_WithJSONBlock(t *testing.T) {
	config := &runnerv1.AutopilotConfig{
		Prompt: "Test",
	}

	workerCtrl := &MockPodController{}

	rp := NewAutopilotController(Config{
		AutopilotKey: "autopilot-123",
		PodKey:       "worker-123",
		ProtoConfig:  config,
		PodCtrl:      workerCtrl,
		Reporter:     &MockEventReporter{},
	})

	output := `Task analysis complete.
TASK_COMPLETED
{"files_changed": ["auth.go", "user.go", "test.go"]}
Summary: All files updated.`

	decision := rp.parseDecision(output)

	assert.Equal(t, DecisionCompleted, decision.Type)
	assert.Len(t, decision.FilesChanged, 3)
	assert.Contains(t, decision.FilesChanged, "auth.go")
}

func TestDecisionType_Constants(t *testing.T) {
	// Verify decision type constants match expected values
	assert.Equal(t, DecisionType("TASK_COMPLETED"), DecisionCompleted)
	assert.Equal(t, DecisionType("CONTINUE"), DecisionContinue)
	assert.Equal(t, DecisionType("NEED_HUMAN_HELP"), DecisionNeedHumanHelp)
	assert.Equal(t, DecisionType("GIVE_UP"), DecisionGiveUp)
}

// Test DecisionParser directly
func TestDecisionParser_ParseDecision(t *testing.T) {
	parser := NewDecisionParser()

	output := `Analysis complete.
TASK_COMPLETED
Successfully implemented the feature.`

	decision := parser.ParseDecision(output)

	assert.Equal(t, DecisionCompleted, decision.Type)
	assert.Contains(t, decision.Summary, "Successfully implemented")
}
