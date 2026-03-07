package ticket

import (
	"errors"
	"regexp"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
)

var (
	ErrMRNotFound       = errors.New("merge request not found")
	ErrNoGitProvider    = errors.New("git provider not available")
	ErrNoRepositoryLink = errors.New("ticket has no repository linked")
)

// ticketSlugRegex matches patterns like "AM-123" in branch names.
var ticketSlugRegex = regexp.MustCompile(`([A-Z]+-\d+)`)

// MRSyncService handles MR synchronization with git providers.
type MRSyncService struct {
	repo        ticket.MRSyncRepository
	gitProvider git.Provider
}

// NewMRSyncService creates a new MR sync service.
func NewMRSyncService(repo ticket.MRSyncRepository, gitProvider git.Provider) *MRSyncService {
	return &MRSyncService{
		repo:        repo,
		gitProvider: gitProvider,
	}
}

// MRData represents MR data from git provider.
type MRData struct {
	IID            int
	WebURL         string
	Title          string
	SourceBranch   string
	TargetBranch   string
	State          string
	PipelineStatus *string
	PipelineID     *int64
	PipelineURL    *string
	MergeCommitSHA *string
	MergedAt       *time.Time
}
