package v1

type createLoopRequest struct {
	Name              string                 `json:"name" binding:"required,min=1,max=255"`
	Slug              string                 `json:"slug"`
	Description       *string                `json:"description"`
	AgentSlug         string                 `json:"agent_slug"`
	PermissionMode    string                 `json:"permission_mode"`
	PromptTemplate    string                 `json:"prompt_template" binding:"required"`
	PromptVariables   map[string]interface{} `json:"prompt_variables"`
	RepositoryID      *int64                 `json:"repository_id"`
	RunnerID          *int64                 `json:"runner_id"`
	BranchName        *string                `json:"branch_name"`
	TicketID          *int64                 `json:"ticket_id"`
	// CredentialProfileID specifies which credential profile to use
	// - nil (field absent): use user's default profile, fallback to RunnerHost if no default
	// - 0: explicit RunnerHost mode (use Runner's local environment, no credentials injected)
	// - >0: use specified credential profile ID
	CredentialProfileID *int64               `json:"credential_profile_id"`
	ConfigOverrides   map[string]interface{} `json:"config_overrides"`
	ExecutionMode     string                 `json:"execution_mode"`
	CronExpression    *string                `json:"cron_expression"`
	AutopilotConfig   map[string]interface{} `json:"autopilot_config"`
	CallbackURL       *string                `json:"callback_url"`
	SandboxStrategy   string                 `json:"sandbox_strategy"`
	SessionPersistence *bool                 `json:"session_persistence"`
	ConcurrencyPolicy string                 `json:"concurrency_policy"`
	MaxConcurrentRuns *int                   `json:"max_concurrent_runs"`
	MaxRetainedRuns   *int                   `json:"max_retained_runs"`
	TimeoutMinutes    *int                   `json:"timeout_minutes"`
	IdleTimeoutSec    *int                   `json:"idle_timeout_sec"`
}

type updateLoopRequest struct {
	Name              *string                `json:"name"`
	Description       *string                `json:"description"`
	AgentSlug         string                 `json:"agent_slug"`
	PermissionMode    *string                `json:"permission_mode"`
	PromptTemplate    *string                `json:"prompt_template"`
	PromptVariables   map[string]interface{} `json:"prompt_variables"`
	RepositoryID      *int64                 `json:"repository_id"`
	RunnerID          *int64                 `json:"runner_id"`
	BranchName        *string                `json:"branch_name"`
	TicketID          *int64                 `json:"ticket_id"`
	// CredentialProfileID specifies which credential profile to use
	// - nil (field absent): use user's default profile, fallback to RunnerHost if no default
	// - 0: explicit RunnerHost mode (use Runner's local environment, no credentials injected)
	// - >0: use specified credential profile ID
	CredentialProfileID *int64               `json:"credential_profile_id"`
	ConfigOverrides   map[string]interface{} `json:"config_overrides"`
	ExecutionMode     *string                `json:"execution_mode"`
	CronExpression    *string                `json:"cron_expression"`
	AutopilotConfig   map[string]interface{} `json:"autopilot_config"`
	CallbackURL       *string                `json:"callback_url"`
	SandboxStrategy   *string                `json:"sandbox_strategy"`
	SessionPersistence *bool                 `json:"session_persistence"`
	ConcurrencyPolicy *string                `json:"concurrency_policy"`
	MaxConcurrentRuns *int                   `json:"max_concurrent_runs"`
	MaxRetainedRuns   *int                   `json:"max_retained_runs"`
	TimeoutMinutes    *int                   `json:"timeout_minutes"`
	IdleTimeoutSec    *int                   `json:"idle_timeout_sec"`
}

type listLoopsQuery struct {
	Status        string `form:"status"`
	ExecutionMode string `form:"execution_mode"`
	CronEnabled   *bool  `form:"cron_enabled"`
	Query         string `form:"query"`
	Limit         int    `form:"limit"`
	Offset        int    `form:"offset"`
}

type listRunsQuery struct {
	Status string `form:"status"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}
