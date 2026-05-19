package git

import (
	"context"
	"errors"
	"time"
)

const (
	ProviderTypeGitLab = "gitlab"
	ProviderTypeGitHub = "github"
	ProviderTypeGitee  = "gitee"
)

var (
	ErrProviderNotSupported = errors.New("git provider not supported")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrNotFound             = errors.New("resource not found")
	ErrRateLimited          = errors.New("rate limited")
)

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type Project struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description"`
	DefaultBranch string    `json:"default_branch"`
	WebURL        string    `json:"web_url"`
	HttpCloneURL  string    `json:"http_clone_url"`
	SSHCloneURL   string    `json:"ssh_clone_url"`
	Visibility    string    `json:"visibility"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Branch struct {
	Name      string `json:"name"`
	CommitSHA string `json:"commit_sha"`
	Protected bool   `json:"protected"`
	Default   bool   `json:"default"`
}

type MergeRequest struct {
	ID           int       `json:"id"`
	IID          int       `json:"iid"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	SourceBranch string    `json:"source_branch"`
	TargetBranch string    `json:"target_branch"`
	State        string    `json:"state"`
	WebURL       string    `json:"web_url"`
	Author       *User     `json:"author"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MergedAt     *time.Time `json:"merged_at,omitempty"`

	PipelineStatus string `json:"pipeline_status,omitempty"`
	PipelineID     int    `json:"pipeline_id,omitempty"`
	PipelineURL    string `json:"pipeline_url,omitempty"`

	MergeCommitSHA string `json:"merge_commit_sha,omitempty"`
}

type CreateMRRequest struct {
	ProjectID    string `json:"project_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
}

type Commit struct {
	SHA         string    `json:"sha"`
	Message     string    `json:"message"`
	Author      string    `json:"author"`
	AuthorEmail string    `json:"author_email"`
	CreatedAt   time.Time `json:"created_at"`
}

const (
	PipelineStatusPending  = "pending"
	PipelineStatusRunning  = "running"
	PipelineStatusSuccess  = "success"
	PipelineStatusFailed   = "failed"
	PipelineStatusCanceled = "canceled"
	PipelineStatusSkipped  = "skipped"
	PipelineStatusManual   = "manual"
)

const (
	JobStatusCreated  = "created"
	JobStatusPending  = "pending"
	JobStatusRunning  = "running"
	JobStatusSuccess  = "success"
	JobStatusFailed   = "failed"
	JobStatusCanceled = "canceled"
	JobStatusSkipped  = "skipped"
	JobStatusManual   = "manual"
)

type Pipeline struct {
	ID        int       `json:"id"`
	IID       int       `json:"iid"`
	ProjectID string    `json:"project_id"`
	Ref       string    `json:"ref"`
	SHA       string    `json:"sha"`
	Status    string    `json:"status"`
	Source    string    `json:"source"`
	WebURL    string    `json:"web_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

type Job struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	Stage        string     `json:"stage"`
	Status       string     `json:"status"`
	Ref          string     `json:"ref"`
	PipelineID   int        `json:"pipeline_id"`
	WebURL       string     `json:"web_url"`
	AllowFailure bool       `json:"allow_failure"`
	Duration     float64    `json:"duration"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

type TriggerPipelineRequest struct {
	Ref       string            `json:"ref"`
	Variables map[string]string `json:"variables,omitempty"`
}

type WebhookConfig struct {
	URL    string   `json:"url"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
}

type Provider interface {
	GetCurrentUser(ctx context.Context) (*User, error)

	GetProject(ctx context.Context, projectID string) (*Project, error)
	ListProjects(ctx context.Context, page, perPage int) ([]*Project, error)
	SearchProjects(ctx context.Context, query string, page, perPage int) ([]*Project, error)

	ListBranches(ctx context.Context, projectID string) ([]*Branch, error)
	GetBranch(ctx context.Context, projectID, branchName string) (*Branch, error)
	CreateBranch(ctx context.Context, projectID, branchName, ref string) (*Branch, error)
	DeleteBranch(ctx context.Context, projectID, branchName string) error

	GetMergeRequest(ctx context.Context, projectID string, mrIID int) (*MergeRequest, error)
	ListMergeRequests(ctx context.Context, projectID string, state string, page, perPage int) ([]*MergeRequest, error)
	ListMergeRequestsByBranch(ctx context.Context, projectID, sourceBranch, state string) ([]*MergeRequest, error)
	CreateMergeRequest(ctx context.Context, req *CreateMRRequest) (*MergeRequest, error)
	UpdateMergeRequest(ctx context.Context, projectID string, mrIID int, title, description string) (*MergeRequest, error)
	MergeMergeRequest(ctx context.Context, projectID string, mrIID int) (*MergeRequest, error)
	CloseMergeRequest(ctx context.Context, projectID string, mrIID int) (*MergeRequest, error)

	GetCommit(ctx context.Context, projectID, sha string) (*Commit, error)
	ListCommits(ctx context.Context, projectID, branch string, page, perPage int) ([]*Commit, error)

	RegisterWebhook(ctx context.Context, projectID string, config *WebhookConfig) (string, error)
	DeleteWebhook(ctx context.Context, projectID, webhookID string) error

	GetFileContent(ctx context.Context, projectID, filePath, ref string) ([]byte, error)

	TriggerPipeline(ctx context.Context, projectID string, req *TriggerPipelineRequest) (*Pipeline, error)
	GetPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error)
	ListPipelines(ctx context.Context, projectID string, ref, status string, page, perPage int) ([]*Pipeline, error)
	CancelPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error)
	RetryPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error)

	GetJob(ctx context.Context, projectID string, jobID int) (*Job, error)
	ListPipelineJobs(ctx context.Context, projectID string, pipelineID int) ([]*Job, error)
	RetryJob(ctx context.Context, projectID string, jobID int) (*Job, error)
	CancelJob(ctx context.Context, projectID string, jobID int) (*Job, error)
	GetJobTrace(ctx context.Context, projectID string, jobID int) (string, error)
	GetJobArtifact(ctx context.Context, projectID string, jobID int, artifactPath string) ([]byte, error)
	DownloadJobArtifacts(ctx context.Context, projectID string, jobID int) ([]byte, error)
}

func NewProvider(providerType, baseURL, accessToken string) (Provider, error) {
	switch providerType {
	case ProviderTypeGitLab:
		return NewGitLabProvider(baseURL, accessToken)
	case ProviderTypeGitHub:
		return NewGitHubProvider(baseURL, accessToken)
	case ProviderTypeGitee:
		return NewGiteeProvider(baseURL, accessToken)
	default:
		return nil, ErrProviderNotSupported
	}
}
