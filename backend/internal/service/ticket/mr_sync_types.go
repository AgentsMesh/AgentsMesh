package ticket

import (
	"errors"
	"regexp"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
	"gorm.io/gorm"
)

var (
	ErrMRNotFound       = errors.New("merge request not found")
	ErrNoGitProvider    = errors.New("git provider not available")
	ErrNoRepositoryLink = errors.New("ticket has no repository linked")
)

// ticketSlugRegex matches patterns like "AM-123" in branch names
var ticketSlugRegex = regexp.MustCompile(`([A-Z]+-\d+)`)

// MRSyncService handles MR synchronization with git providers
type MRSyncService struct {
	db          *gorm.DB
	gitProvider git.Provider
}

// NewMRSyncService creates a new MR sync service
func NewMRSyncService(db *gorm.DB, gitProvider git.Provider) *MRSyncService {
	return &MRSyncService{
		db:          db,
		gitProvider: gitProvider,
	}
}

// MRData represents MR data from git provider
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
