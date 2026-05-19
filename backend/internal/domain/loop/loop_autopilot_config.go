package loop

import "encoding/json"

type AutopilotConfigValues struct {
	MaxIterations       int32 `json:"max_iterations,omitempty"`
	IterationTimeoutSec int32 `json:"iteration_timeout_sec,omitempty"`
	NoProgressThreshold int32 `json:"no_progress_threshold,omitempty"`
	SameErrorThreshold  int32 `json:"same_error_threshold,omitempty"`
	ApprovalTimeoutMin  int32 `json:"approval_timeout_min,omitempty"`
}

func (l *Loop) ParseAutopilotConfig() AutopilotConfigValues {
	var cfg AutopilotConfigValues
	if l.AutopilotConfig != nil {
		_ = json.Unmarshal(l.AutopilotConfig, &cfg)
	}
	return cfg
}
