package git

import "context"

// GetJob is not fully supported by Gitee API
func (p *GiteeProvider) GetJob(ctx context.Context, projectID string, jobID int) (*Job, error) {
	return nil, ErrGiteePipelineNotSupported
}

// ListPipelineJobs is not fully supported by Gitee API
func (p *GiteeProvider) ListPipelineJobs(ctx context.Context, projectID string, pipelineID int) ([]*Job, error) {
	return nil, ErrGiteePipelineNotSupported
}

// RetryJob is not fully supported by Gitee API
func (p *GiteeProvider) RetryJob(ctx context.Context, projectID string, jobID int) (*Job, error) {
	return nil, ErrGiteePipelineNotSupported
}

// CancelJob is not fully supported by Gitee API
func (p *GiteeProvider) CancelJob(ctx context.Context, projectID string, jobID int) (*Job, error) {
	return nil, ErrGiteePipelineNotSupported
}

// GetJobTrace is not fully supported by Gitee API
func (p *GiteeProvider) GetJobTrace(ctx context.Context, projectID string, jobID int) (string, error) {
	return "", ErrGiteePipelineNotSupported
}

// GetJobArtifact is not fully supported by Gitee API
func (p *GiteeProvider) GetJobArtifact(ctx context.Context, projectID string, jobID int, artifactPath string) ([]byte, error) {
	return nil, ErrGiteePipelineNotSupported
}

// DownloadJobArtifacts is not fully supported by Gitee API
func (p *GiteeProvider) DownloadJobArtifacts(ctx context.Context, projectID string, jobID int) ([]byte, error) {
	return nil, ErrGiteePipelineNotSupported
}
