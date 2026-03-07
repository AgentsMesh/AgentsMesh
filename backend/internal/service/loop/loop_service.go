package loop

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	loopDomain "github.com/anthropics/agentsmesh/backend/internal/domain/loop"
	"github.com/robfig/cron/v3"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)


var (
	ErrLoopNotFound        = errors.New("loop not found")
	ErrDuplicateSlug       = errors.New("loop slug already exists in this organization")
	ErrLoopDisabled        = errors.New("loop is disabled")
	ErrInvalidCron         = errors.New("invalid cron expression")
	ErrInvalidSlug         = errors.New("slug must be lowercase alphanumeric with hyphens, 3-100 chars")
	ErrInvalidEnumValue    = errors.New("invalid enum value")
	ErrInvalidCallbackURL  = errors.New("invalid callback URL")
)

// validateCallbackURL validates a webhook callback URL to prevent SSRF.
// Only allows http/https schemes and blocks loopback/private addresses.
func validateCallbackURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCallbackURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: scheme must be http or https", ErrInvalidCallbackURL)
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("%w: missing host", ErrInvalidCallbackURL)
	}
	// Block loopback and well-known private hostnames
	blockedHosts := []string{"localhost", "127.0.0.1", "::1", "0.0.0.0", "[::1]"}
	for _, blocked := range blockedHosts {
		if strings.EqualFold(host, blocked) {
			return fmt.Errorf("%w: callback URL must not target localhost", ErrInvalidCallbackURL)
		}
	}
	// Block private IP ranges (basic SSRF protection)
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("%w: callback URL must not target private/internal networks", ErrInvalidCallbackURL)
		}
	}
	return nil
}

var validExecutionModes = map[string]bool{
	loopDomain.ExecutionModeAutopilot: true,
	loopDomain.ExecutionModeDirect:    true,
}

var validSandboxStrategies = map[string]bool{
	loopDomain.SandboxStrategyPersistent: true,
	loopDomain.SandboxStrategyFresh:      true,
}

func validateEnumFields(executionMode, sandboxStrategy, concurrencyPolicy string) error {
	if executionMode != "" && !validExecutionModes[executionMode] {
		return fmt.Errorf("%w: execution_mode must be 'autopilot' or 'direct'", ErrInvalidEnumValue)
	}
	if sandboxStrategy != "" && !validSandboxStrategies[sandboxStrategy] {
		return fmt.Errorf("%w: sandbox_strategy must be 'persistent' or 'fresh'", ErrInvalidEnumValue)
	}
	if concurrencyPolicy != "" && concurrencyPolicy != loopDomain.ConcurrencyPolicySkip {
		return fmt.Errorf("%w: concurrency_policy currently only supports 'skip'", ErrInvalidEnumValue)
	}
	return nil
}

var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,98}[a-z0-9]$`)

// LoopService handles Loop CRUD operations
type LoopService struct {
	repo loopDomain.LoopRepository
}

// NewLoopService creates a new LoopService
func NewLoopService(repo loopDomain.LoopRepository) *LoopService {
	return &LoopService{
		repo: repo,
	}
}

// CreateLoopRequest represents a loop creation request
type CreateLoopRequest struct {
	OrganizationID int64
	CreatedByID    int64
	Name           string
	Slug           string
	Description    *string

	// Agent configuration
	AgentTypeID       *int64
	CustomAgentTypeID *int64
	PermissionMode    string
	PromptTemplate    string
	PromptVariables   []byte // JSON

	// Resource bindings
	RepositoryID        *int64
	RunnerID            *int64
	BranchName          *string
	TicketID            *int64
	CredentialProfileID *int64
	ConfigOverrides     []byte // JSON

	// Execution
	ExecutionMode   string
	CronExpression  *string // Optional — if set, cron scheduling is enabled
	AutopilotConfig []byte  // JSON
	CallbackURL     *string

	// Policies
	SandboxStrategy    string
	SessionPersistence bool
	ConcurrencyPolicy  string
	MaxConcurrentRuns  int
	MaxRetainedRuns    int // 0 = unlimited (keep all runs)
	TimeoutMinutes     int
}

// UpdateLoopRequest represents a loop update request
type UpdateLoopRequest struct {
	Name              *string
	Description       *string
	AgentTypeID       *int64
	CustomAgentTypeID *int64
	PermissionMode    *string
	PromptTemplate    *string
	PromptVariables   []byte

	RepositoryID        *int64
	RunnerID            *int64
	BranchName          *string
	TicketID            *int64
	CredentialProfileID *int64
	ConfigOverrides     []byte

	ExecutionMode   *string
	CronExpression  *string
	AutopilotConfig []byte
	CallbackURL     *string

	SandboxStrategy    *string
	SessionPersistence *bool
	ConcurrencyPolicy  *string
	MaxConcurrentRuns  *int
	MaxRetainedRuns    *int
	TimeoutMinutes     *int
}

// ListLoopsFilter represents filters for listing loops (service-level alias)
type ListLoopsFilter = loopDomain.ListFilter

// generateSlug creates a URL-friendly slug from a name.
// Handles non-ASCII names (e.g. Chinese) by falling back to a timestamp-based slug.
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 100 {
		slug = slug[:100]
		slug = strings.TrimRight(slug, "-")
	}
	// If the slug is empty or too short (e.g. non-ASCII name with no alphanumeric chars),
	// generate a fallback using "loop-" prefix + timestamp
	if len(slug) == 0 {
		slug = fmt.Sprintf("loop-%d", time.Now().UnixMilli())
	} else if len(slug) < 3 {
		slug = slug + "-loop"
	}
	return slug
}

// Create creates a new Loop
func (s *LoopService) Create(ctx context.Context, req *CreateLoopRequest) (*loopDomain.Loop, error) {
	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Name)
	}
	if !slugRegex.MatchString(slug) {
		return nil, ErrInvalidSlug
	}

	// Set defaults
	if req.PermissionMode == "" {
		req.PermissionMode = "bypassPermissions"
	}
	if req.ExecutionMode == "" {
		req.ExecutionMode = loopDomain.ExecutionModeAutopilot
	}
	if req.SandboxStrategy == "" {
		req.SandboxStrategy = loopDomain.SandboxStrategyPersistent
	}
	if req.ConcurrencyPolicy == "" {
		req.ConcurrencyPolicy = loopDomain.ConcurrencyPolicySkip
	}
	if req.MaxConcurrentRuns == 0 {
		req.MaxConcurrentRuns = 1
	}
	if req.TimeoutMinutes == 0 {
		req.TimeoutMinutes = 60
	}
	if req.AutopilotConfig == nil {
		req.AutopilotConfig = []byte("{}")
	}
	if req.ConfigOverrides == nil {
		req.ConfigOverrides = []byte("{}")
	}
	if req.PromptVariables == nil {
		req.PromptVariables = []byte("{}")
	}

	// Validate enum fields
	if err := validateEnumFields(req.ExecutionMode, req.SandboxStrategy, req.ConcurrencyPolicy); err != nil {
		return nil, err
	}

	// Validate callback URL (SSRF protection)
	if req.CallbackURL != nil {
		if err := validateCallbackURL(*req.CallbackURL); err != nil {
			return nil, err
		}
	}

	// Calculate initial next_run_at for cron loops
	var nextRunAt *time.Time
	if req.CronExpression != nil && *req.CronExpression != "" {
		schedule, err := cronParser.Parse(*req.CronExpression)
		if err != nil {
			return nil, ErrInvalidCron
		}
		next := schedule.Next(time.Now())
		nextRunAt = &next
	}

	loop := &loopDomain.Loop{
		OrganizationID:      req.OrganizationID,
		Name:                req.Name,
		Slug:                slug,
		Description:         req.Description,
		AgentTypeID:         req.AgentTypeID,
		CustomAgentTypeID:   req.CustomAgentTypeID,
		PermissionMode:      req.PermissionMode,
		PromptTemplate:      req.PromptTemplate,
		PromptVariables:     req.PromptVariables,
		RepositoryID:        req.RepositoryID,
		RunnerID:            req.RunnerID,
		BranchName:          req.BranchName,
		TicketID:            req.TicketID,
		CredentialProfileID: req.CredentialProfileID,
		ConfigOverrides:     req.ConfigOverrides,
		ExecutionMode:       req.ExecutionMode,
		CronExpression:      req.CronExpression,
		AutopilotConfig:     req.AutopilotConfig,
		CallbackURL:         req.CallbackURL,
		Status:              loopDomain.StatusEnabled,
		SandboxStrategy:     req.SandboxStrategy,
		SessionPersistence:  req.SessionPersistence,
		ConcurrencyPolicy:   req.ConcurrencyPolicy,
		MaxConcurrentRuns:   req.MaxConcurrentRuns,
		MaxRetainedRuns:     req.MaxRetainedRuns,
		TimeoutMinutes:      req.TimeoutMinutes,
		CreatedByID:         req.CreatedByID,
		NextRunAt:           nextRunAt,
	}

	if err := s.repo.Create(ctx, loop); err != nil {
		if strings.Contains(err.Error(), "idx_loops_org_slug") {
			return nil, ErrDuplicateSlug
		}
		return nil, err
	}

	return loop, nil
}

// GetBySlug retrieves a Loop by organization ID and slug
func (s *LoopService) GetBySlug(ctx context.Context, orgID int64, slug string) (*loopDomain.Loop, error) {
	loop, err := s.repo.GetBySlug(ctx, orgID, slug)
	if err != nil {
		if errors.Is(err, loopDomain.ErrNotFound) {
			return nil, ErrLoopNotFound
		}
		return nil, err
	}
	return loop, nil
}

// GetByID retrieves a Loop by ID
func (s *LoopService) GetByID(ctx context.Context, id int64) (*loopDomain.Loop, error) {
	loop, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, loopDomain.ErrNotFound) {
			return nil, ErrLoopNotFound
		}
		return nil, err
	}
	return loop, nil
}

// List lists Loops with filters
func (s *LoopService) List(ctx context.Context, filter *ListLoopsFilter) ([]*loopDomain.Loop, int64, error) {
	return s.repo.List(ctx, filter)
}

// Update updates a Loop
func (s *LoopService) Update(ctx context.Context, orgID int64, slug string, req *UpdateLoopRequest) (*loopDomain.Loop, error) {
	loop, err := s.GetBySlug(ctx, orgID, slug)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.AgentTypeID != nil {
		updates["agent_type_id"] = *req.AgentTypeID
	}
	if req.CustomAgentTypeID != nil {
		updates["custom_agent_type_id"] = *req.CustomAgentTypeID
	}
	if req.PermissionMode != nil {
		updates["permission_mode"] = *req.PermissionMode
	}
	if req.PromptTemplate != nil {
		updates["prompt_template"] = *req.PromptTemplate
	}
	if req.RepositoryID != nil {
		updates["repository_id"] = *req.RepositoryID
	}
	if req.RunnerID != nil {
		updates["runner_id"] = *req.RunnerID
	}
	if req.BranchName != nil {
		updates["branch_name"] = *req.BranchName
	}
	if req.TicketID != nil {
		updates["ticket_id"] = *req.TicketID
	}
	if req.CredentialProfileID != nil {
		updates["credential_profile_id"] = *req.CredentialProfileID
	}
	if req.ConfigOverrides != nil {
		updates["config_overrides"] = req.ConfigOverrides
	}
	if req.ExecutionMode != nil {
		updates["execution_mode"] = *req.ExecutionMode
	}
	if req.CronExpression != nil {
		updates["cron_expression"] = *req.CronExpression
		// Recalculate next_run_at when cron expression changes
		if *req.CronExpression != "" {
			schedule, err := cronParser.Parse(*req.CronExpression)
			if err != nil {
				return nil, ErrInvalidCron
			}
			next := schedule.Next(time.Now())
			updates["next_run_at"] = next
		} else {
			// Cron cleared — remove next_run_at
			updates["next_run_at"] = nil
		}
	}
	if req.AutopilotConfig != nil {
		updates["autopilot_config"] = req.AutopilotConfig
	}
	if req.PromptVariables != nil {
		updates["prompt_variables"] = req.PromptVariables
	}
	if req.CallbackURL != nil {
		if *req.CallbackURL == "" {
			// Empty string clears the callback URL (set to NULL)
			updates["callback_url"] = nil
		} else {
			if err := validateCallbackURL(*req.CallbackURL); err != nil {
				return nil, err
			}
			updates["callback_url"] = *req.CallbackURL
		}
	}
	if req.SandboxStrategy != nil {
		updates["sandbox_strategy"] = *req.SandboxStrategy
	}
	if req.SessionPersistence != nil {
		updates["session_persistence"] = *req.SessionPersistence
	}
	if req.ConcurrencyPolicy != nil {
		updates["concurrency_policy"] = *req.ConcurrencyPolicy
	}
	if req.MaxConcurrentRuns != nil {
		updates["max_concurrent_runs"] = *req.MaxConcurrentRuns
	}
	if req.MaxRetainedRuns != nil {
		updates["max_retained_runs"] = *req.MaxRetainedRuns
	}
	if req.TimeoutMinutes != nil {
		updates["timeout_minutes"] = *req.TimeoutMinutes
	}

	// H1: When runner changes on a persistent-sandbox loop, break the resume chain.
	// The old sandbox lives on the old Runner, so it can't be resumed from a different Runner.
	if req.RunnerID != nil {
		effectiveRunnerID := *req.RunnerID
		currentRunnerID := int64(0)
		if loop.RunnerID != nil {
			currentRunnerID = *loop.RunnerID
		}
		if effectiveRunnerID != currentRunnerID && loop.IsPersistent() && loop.LastPodKey != nil {
			updates["last_pod_key"] = nil
			updates["sandbox_path"] = nil
		}
	}

	// When switching from persistent to fresh, clear runtime state
	if req.SandboxStrategy != nil && *req.SandboxStrategy == loopDomain.SandboxStrategyFresh &&
		loop.SandboxStrategy == loopDomain.SandboxStrategyPersistent {
		updates["last_pod_key"] = nil
		updates["sandbox_path"] = nil
	}

	// Validate enum fields if present in updates
	execMode := ""
	if req.ExecutionMode != nil {
		execMode = *req.ExecutionMode
	}
	sandboxStrat := ""
	if req.SandboxStrategy != nil {
		sandboxStrat = *req.SandboxStrategy
	}
	concPolicy := ""
	if req.ConcurrencyPolicy != nil {
		concPolicy = *req.ConcurrencyPolicy
	}
	if err := validateEnumFields(execMode, sandboxStrat, concPolicy); err != nil {
		return nil, err
	}

	if len(updates) > 0 {
		if err := s.repo.Update(ctx, loop.ID, updates); err != nil {
			return nil, err
		}
	}

	return s.GetBySlug(ctx, orgID, slug)
}

var ErrHasActiveRuns = errors.New("loop has active runs")

// Delete deletes a Loop (hard delete).
// Atomically checks for active runs — returns ErrHasActiveRuns if any exist.
func (s *LoopService) Delete(ctx context.Context, orgID int64, slug string) error {
	affected, err := s.repo.Delete(ctx, orgID, slug)
	if err != nil {
		if errors.Is(err, loopDomain.ErrHasActiveRuns) {
			return ErrHasActiveRuns
		}
		return err
	}
	if affected == 0 {
		return ErrLoopNotFound
	}
	return nil
}

var validStatuses = map[string]bool{
	loopDomain.StatusEnabled:  true,
	loopDomain.StatusDisabled: true,
}

// SetStatus updates the status of a Loop.
// When re-enabling a cron loop, recalculates next_run_at so cron scheduling resumes immediately.
func (s *LoopService) SetStatus(ctx context.Context, orgID int64, slug string, status string) (*loopDomain.Loop, error) {
	if !validStatuses[status] {
		return nil, fmt.Errorf("%w: status must be 'enabled' or 'disabled'", ErrInvalidEnumValue)
	}

	loop, err := s.GetBySlug(ctx, orgID, slug)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"status": status,
	}

	// When re-enabling a cron loop, recalculate next_run_at so scheduling resumes
	if status == loopDomain.StatusEnabled && loop.HasCron() {
		schedule, err := cronParser.Parse(*loop.CronExpression)
		if err == nil {
			next := schedule.Next(time.Now())
			updates["next_run_at"] = next
		}
	}

	if err := s.repo.Update(ctx, loop.ID, updates); err != nil {
		return nil, err
	}

	return s.GetBySlug(ctx, orgID, slug)
}

// UpdateRunStats updates the run statistics on a Loop (incremental)
func (s *LoopService) UpdateRunStats(ctx context.Context, loopID int64, status string, lastRunAt time.Time) error {
	return s.repo.IncrementRunStats(ctx, loopID, status, lastRunAt)
}

// UpdateStats sets the run statistics on a Loop to absolute values.
// Used by LoopOrchestrator.RefreshLoopStats() which computes stats from Pod status (SSOT).
func (s *LoopService) UpdateStats(ctx context.Context, loopID int64, total, successful, failed int) error {
	return s.repo.Update(ctx, loopID, map[string]interface{}{
		"total_runs":      total,
		"successful_runs": successful,
		"failed_runs":     failed,
	})
}

// ClearRuntimeState clears sandbox_path and last_pod_key (sets to NULL).
func (s *LoopService) ClearRuntimeState(ctx context.Context, loopID int64) error {
	return s.repo.Update(ctx, loopID, map[string]interface{}{
		"sandbox_path": nil,
		"last_pod_key": nil,
	})
}

// UpdateRuntimeState updates sandbox_path and last_pod_key
func (s *LoopService) UpdateRuntimeState(ctx context.Context, loopID int64, sandboxPath *string, lastPodKey *string) error {
	updates := map[string]interface{}{}
	if sandboxPath != nil {
		updates["sandbox_path"] = *sandboxPath
	}
	if lastPodKey != nil {
		updates["last_pod_key"] = *lastPodKey
	}
	if len(updates) == 0 {
		return nil
	}
	return s.repo.Update(ctx, loopID, updates)
}

// UpdateNextRunAt updates the next scheduled run time
func (s *LoopService) UpdateNextRunAt(ctx context.Context, loopID int64, nextRunAt *time.Time) error {
	return s.repo.Update(ctx, loopID, map[string]interface{}{
		"next_run_at": nextRunAt,
	})
}

// GetDueCronLoops returns enabled loops with cron scheduling that are due for execution.
// orgIDs filters to specific organizations; nil means all orgs (single-instance mode).
func (s *LoopService) GetDueCronLoops(ctx context.Context, orgIDs []int64) ([]*loopDomain.Loop, error) {
	return s.repo.GetDueCronLoops(ctx, orgIDs)
}

// ClaimCronLoop atomically claims a cron loop with SKIP LOCKED and advances next_run_at.
func (s *LoopService) ClaimCronLoop(ctx context.Context, loopID int64, nextRunAt *time.Time) (bool, error) {
	return s.repo.ClaimCronLoop(ctx, loopID, nextRunAt)
}

// FindLoopsNeedingNextRun returns enabled cron loops with next_run_at IS NULL.
// orgIDs filters to specific organizations; nil means all orgs (single-instance mode).
func (s *LoopService) FindLoopsNeedingNextRun(ctx context.Context, orgIDs []int64) ([]*loopDomain.Loop, error) {
	return s.repo.FindLoopsNeedingNextRun(ctx, orgIDs)
}
