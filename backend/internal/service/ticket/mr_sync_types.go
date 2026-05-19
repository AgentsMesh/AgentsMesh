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

var ticketSlugRegex = regexp.MustCompile(`([A-Z]+-\d+)`)

// MRSyncService handles MR synchronization with git providers.
type MRSyncService struct {
	repo        ticket.MRSyncRepository
	gitProvider git.Provider
}

func NewMRSyncService(repo ticket.MRSyncRepository, gitProvider git.Provider) *MRSyncService {
	return &MRSyncService{
		repo:        repo,
		gitProvider: gitProvider,
	}
}

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
