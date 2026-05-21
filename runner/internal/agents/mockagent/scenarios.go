package mockagent

import (
	"encoding/json"
	"log/slog"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// scenario describes the ACP behavior for one named test profile.
// Add new scenarios by registering them in registerScenarios() and
// implementing the handler in a dedicated scenario_*.go file.
//
// The handler receives the shared runtimeState (writer + pending response
// registry), the prompt request id (to be answered with end_turn at the end),
// and the raw params (extract user text via extractPromptText).
type scenario struct {
	name         string
	handlePrompt func(state *runtimeState, id int64, params json.RawMessage, logger *slog.Logger) error
}

var registeredScenarios = registerScenarios()

func registerScenarios() map[string]scenario {
	scenarios := map[string]scenario{}
	for _, s := range []scenario{
		{name: "echo", handlePrompt: scenarioEcho},
		{name: "streaming_3", handlePrompt: scenarioStreaming3},
		{name: "thinking_then_answer", handlePrompt: scenarioThinkingThenAnswer},
		{name: "tool_call_edit", handlePrompt: scenarioToolCallEdit},
		{name: "permission_request_edit", handlePrompt: scenarioPermissionRequestEdit},
		{name: "config_change_plan", handlePrompt: scenarioConfigChangePlan},
		{name: "fail_after_1s", handlePrompt: scenarioFailAfter1s},
		{name: "malformed_json", handlePrompt: scenarioMalformedJSON},
		{name: "tool_call_failed", handlePrompt: scenarioToolCallFailed},
		{name: "log_warnings", handlePrompt: scenarioLogWarnings},
	} {
		scenarios[s.name] = s
	}
	return scenarios
}

func lookupScenario(name string) (scenario, bool) {
	if name == "" {
		name = "echo"
	}
	s, ok := registeredScenarios[name]
	return s, ok
}

// confirm scenario handlers do not accidentally use the bare acp writer —
// keeps the linter from flagging acp as unused after the scenario file lift.
var _ = (*acp.Writer)(nil)
