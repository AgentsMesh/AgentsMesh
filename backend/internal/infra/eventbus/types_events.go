package eventbus

type PodStatusChangedData struct {
	PodKey         string `json:"pod_key"`
	Status         string `json:"status"`
	PreviousStatus string `json:"previous_status,omitempty"`
	AgentStatus    string `json:"agent_status,omitempty"`
	ErrorCode      string `json:"error_code,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

type PodCreatedData struct {
	PodKey      string `json:"pod_key"`
	Status      string `json:"status"`
	AgentStatus string `json:"agent_status,omitempty"`
	RunnerID    int64  `json:"runner_id"`
	TicketID    *int64 `json:"ticket_id,omitempty"`
	TicketSlug  string `json:"ticket_slug,omitempty"`
	CreatedByID int64  `json:"created_by_id"`
}

type RunnerStatusData struct {
	RunnerID      int64  `json:"runner_id"`
	NodeID        string `json:"node_id"`
	Status        string `json:"status"`
	CurrentPods   int    `json:"current_pods,omitempty"`
	LastHeartbeat string `json:"last_heartbeat,omitempty"`
}

type TicketStatusChangedData struct {
	Slug           string `json:"slug"`
	Status         string `json:"status"`
	PreviousStatus string `json:"previous_status,omitempty"`
}

type PodTitleChangedData struct {
	PodKey string `json:"pod_key"`
	Title  string `json:"title"`
}

type PodAliasChangedData struct {
	PodKey string `json:"pod_key"`
	Alias  *string `json:"alias"`
}

type SenderPodAgent struct {
	Name string `json:"name"`
}

type SenderPodInfo struct {
	PodKey string          `json:"pod_key"`
	Alias  *string         `json:"alias,omitempty"`
	Agent  *SenderPodAgent `json:"agent,omitempty"`
}

type ChannelMessageData struct {
	ID            int64          `json:"id"`
	ChannelID     int64          `json:"channel_id"`
	SenderPod     *string        `json:"sender_pod,omitempty"`
	SenderUserID  *int64         `json:"sender_user_id,omitempty"`
	SenderName    string         `json:"sender_name,omitempty"`
	SenderPodInfo *SenderPodInfo `json:"sender_pod_info,omitempty"`
	MessageType   string         `json:"message_type"`
	Body          string         `json:"body"`
	Content       interface{}    `json:"content,omitempty"`
	Mentions      interface{}    `json:"mentions,omitempty"`
	ReplyTo       *int64         `json:"reply_to,omitempty"`
	CreatedAt     string         `json:"created_at"`
}

type ChannelMessageEditedData struct {
	ID        int64       `json:"id"`
	ChannelID int64       `json:"channel_id"`
	Body      string      `json:"body"`
	Content   interface{} `json:"content,omitempty"`
	Mentions  interface{} `json:"mentions,omitempty"`
	EditedAt  string      `json:"edited_at"`
}

type ChannelMessageDeletedData struct {
	ID        int64 `json:"id"`
	ChannelID int64 `json:"channel_id"`
}

type PodInitProgressData struct {
	PodKey   string `json:"pod_key"`
	Phase    string `json:"phase"`    // pending, cloning, preparing, starting_pty, ready
	Progress int    `json:"progress"` // 0-100
	Message  string `json:"message"`
}

type PodRestartingData struct {
	PodKey       string `json:"pod_key"`
	ExitCode     int32  `json:"exit_code"`
	RestartCount int32  `json:"restart_count"`
}

type AutopilotStatusChangedData struct {
	AutopilotControllerKey string `json:"autopilot_controller_key"`
	PodKey                 string `json:"pod_key"`
	Phase                  string `json:"phase"`
	CurrentIteration       int32  `json:"current_iteration"`
	MaxIterations          int32  `json:"max_iterations"`
	CircuitBreakerState    string `json:"circuit_breaker_state"`
	CircuitBreakerReason   string `json:"circuit_breaker_reason,omitempty"`
	UserTakeover           bool   `json:"user_takeover"`
}

type AutopilotIterationData struct {
	AutopilotControllerKey string   `json:"autopilot_controller_key"`
	Iteration              int32    `json:"iteration"`
	Phase                  string   `json:"phase"`
	Summary                string   `json:"summary,omitempty"`
	FilesChanged           []string `json:"files_changed,omitempty"`
	DurationMs             int64    `json:"duration_ms,omitempty"`
}

type AutopilotCreatedData struct {
	AutopilotControllerKey string `json:"autopilot_controller_key"`
	PodKey                 string `json:"pod_key"`
}

type AutopilotTerminatedData struct {
	AutopilotControllerKey string `json:"autopilot_controller_key"`
	Phase                  string `json:"phase,omitempty"`
	Reason                 string `json:"reason,omitempty"`
}

type AutopilotThinkingData struct {
	AutopilotControllerKey string                    `json:"autopilot_controller_key"`
	Iteration              int32                     `json:"iteration"`
	DecisionType           string                    `json:"decision_type"` // continue, completed, need_help, give_up
	Reasoning              string                    `json:"reasoning"`
	Confidence             float64                   `json:"confidence"`
	Action                 *AutopilotActionData      `json:"action,omitempty"`
	Progress               *AutopilotProgressData    `json:"progress,omitempty"`
	HelpRequest            *AutopilotHelpRequestData `json:"help_request,omitempty"`
}

type AutopilotActionData struct {
	Type    string `json:"type"`    // observe, send_input, wait, none
	Content string `json:"content"`
	Reason  string `json:"reason"`
}

type AutopilotProgressData struct {
	Summary        string   `json:"summary"`
	CompletedSteps []string `json:"completed_steps,omitempty"`
	RemainingSteps []string `json:"remaining_steps,omitempty"`
	Percent        int32    `json:"percent"`
}

type AutopilotHelpRequestData struct {
	Reason          string                        `json:"reason"`
	Context         string                        `json:"context"`
	TerminalExcerpt string                        `json:"terminal_excerpt,omitempty"`
	Suggestions     []AutopilotHelpSuggestionData `json:"suggestions,omitempty"`
}

type AutopilotHelpSuggestionData struct {
	Action string `json:"action"`
	Label  string `json:"label"`
}

type MREventData struct {
	MRID           int64  `json:"mr_id"`
	MRIID          int    `json:"mr_iid"`
	MRURL          string `json:"mr_url"`
	SourceBranch   string `json:"source_branch"`
	TargetBranch   string `json:"target_branch,omitempty"`
	Title          string `json:"title,omitempty"`
	State          string `json:"state"`
	Action         string `json:"action,omitempty"` // opened, updated, merged, closed
	TicketID       *int64 `json:"ticket_id,omitempty"`
	TicketSlug     string `json:"ticket_slug,omitempty"`
	PodID          *int64 `json:"pod_id,omitempty"`
	RepositoryID   int64  `json:"repository_id"`
	PipelineStatus string `json:"pipeline_status,omitempty"`
}

type LoopRunWarningData struct {
	LoopID    int64  `json:"loop_id"`
	RunID     int64  `json:"run_id"`
	RunNumber int    `json:"run_number"`
	Warning   string `json:"warning"`
	Detail    string `json:"detail,omitempty"`
}

type PipelineEventData struct {
	MRID           int64  `json:"mr_id,omitempty"`
	PipelineID     int64  `json:"pipeline_id"`
	PipelineStatus string `json:"pipeline_status"`
	PipelineURL    string `json:"pipeline_url,omitempty"`
	SourceBranch   string `json:"source_branch,omitempty"`
	TicketID       *int64 `json:"ticket_id,omitempty"`
	TicketSlug     string `json:"ticket_slug,omitempty"`
	PodID          *int64 `json:"pod_id,omitempty"`
	RepositoryID   int64  `json:"repository_id"`
}

type ChannelMemberChangedData struct {
	ChannelID int64  `json:"channel_id"`
	UserID    int64  `json:"user_id"`
	Role      string `json:"role,omitempty"`
}
