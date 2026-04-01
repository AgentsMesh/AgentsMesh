package loop

import "encoding/json"

// AutopilotConfigValues is the typed representation of Loop.AutopilotConfig JSON.
// All fields are optional — zero values mean "use domain defaults" (applied by agentpod.ApplyDefaults).
type AutopilotConfigValues struct {
	MaxIterations       int32 `json:"max_iterations,omitempty"`
	IterationTimeoutSec int32 `json:"iteration_timeout_sec,omitempty"`
	NoProgressThreshold int32 `json:"no_progress_threshold,omitempty"`
	SameErrorThreshold  int32 `json:"same_error_threshold,omitempty"`
	ApprovalTimeoutMin  int32 `json:"approval_timeout_min,omitempty"`
}

// ParseAutopilotConfig deserializes the AutopilotConfig JSON into a typed struct.
// Returns zero-value struct if AutopilotConfig is nil or invalid (all zeros → domain defaults apply).
func (l *Loop) ParseAutopilotConfig() AutopilotConfigValues {
	var cfg AutopilotConfigValues
	if l.AutopilotConfig != nil {
		_ = json.Unmarshal(l.AutopilotConfig, &cfg)
	}
	return cfg
}
