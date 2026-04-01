package loop

import (
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
	ErrLoopNotFound       = errors.New("loop not found")
	ErrDuplicateSlug      = errors.New("loop slug already exists in this organization")
	ErrLoopDisabled       = errors.New("loop is disabled")
	ErrInvalidCron        = errors.New("invalid cron expression")
	ErrInvalidSlug        = errors.New("slug must be lowercase alphanumeric with hyphens, 3-100 chars")
	ErrInvalidEnumValue   = errors.New("invalid enum value")
	ErrInvalidCallbackURL = errors.New("invalid callback URL")
)

// validateCallbackURL validates a webhook callback URL to prevent SSRF.
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
	blockedHosts := []string{"localhost", "127.0.0.1", "::1", "0.0.0.0", "[::1]"}
	for _, blocked := range blockedHosts {
		if strings.EqualFold(host, blocked) {
			return fmt.Errorf("%w: callback URL must not target localhost", ErrInvalidCallbackURL)
		}
	}
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

func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 100 {
		slug = slug[:100]
		slug = strings.TrimRight(slug, "-")
	}
	if len(slug) == 0 {
		slug = fmt.Sprintf("loop-%d", time.Now().UnixMilli())
	} else if len(slug) < 3 {
		slug = slug + "-loop"
	}
	return slug
}

// CreateLoopRequest represents a loop creation request.
type CreateLoopRequest struct {
	OrganizationID int64
	CreatedByID    int64
	Name           string
	Slug           string
	Description    *string

	AgentSlug       string
	PermissionMode  string
	PromptTemplate  string
	PromptVariables []byte

	RepositoryID        *int64
	RunnerID            *int64
	BranchName          *string
	TicketID            *int64
	CredentialProfileID *int64
	ConfigOverrides     []byte

	ExecutionMode   string
	CronExpression  *string
	AutopilotConfig []byte
	CallbackURL     *string

	SandboxStrategy    string
	SessionPersistence bool
	ConcurrencyPolicy  string
	MaxConcurrentRuns  int
	MaxRetainedRuns    int
	TimeoutMinutes     int
	IdleTimeoutSec     int
}

// UpdateLoopRequest represents a loop update request.
type UpdateLoopRequest struct {
	Name            *string
	Description     *string
	AgentSlug       string
	PermissionMode  *string
	PromptTemplate  *string
	PromptVariables []byte

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
	IdleTimeoutSec     *int
}

type ListLoopsFilter = loopDomain.ListFilter
