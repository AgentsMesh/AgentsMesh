package git

import "context"

func (p *GiteeProvider) GetJob(ctx context.Context, projectID string, jobID int) (*Job, error) {
	return nil, ErrGiteePipelineNotSupported
}

func (p *GiteeProvider) ListPipelineJobs(ctx context.Context, projectID string, pipelineID int) ([]*Job, error) {
	return nil, ErrGiteePipelineNotSupported
}

func (p *GiteeProvider) RetryJob(ctx context.Context, projectID string, jobID int) (*Job, error) {
	return nil, ErrGiteePipelineNotSupported
}

func (p *GiteeProvider) CancelJob(ctx context.Context, projectID string, jobID int) (*Job, error) {
	return nil, ErrGiteePipelineNotSupported
}

func (p *GiteeProvider) GetJobTrace(ctx context.Context, projectID string, jobID int) (string, error) {
	return "", ErrGiteePipelineNotSupported
}

func (p *GiteeProvider) GetJobArtifact(ctx context.Context, projectID string, jobID int, artifactPath string) ([]byte, error) {
	return nil, ErrGiteePipelineNotSupported
}

func (p *GiteeProvider) DownloadJobArtifacts(ctx context.Context, projectID string, jobID int) ([]byte, error) {
	return nil, ErrGiteePipelineNotSupported
}
