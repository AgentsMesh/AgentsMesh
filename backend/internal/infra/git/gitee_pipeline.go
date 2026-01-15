package git

import (
	"context"
	"fmt"
)

// ErrGiteePipelineNotSupported indicates that Gitee pipeline API is not fully supported
var ErrGiteePipelineNotSupported = fmt.Errorf("gitee pipeline API not fully supported")

// TriggerPipeline is not fully supported by Gitee API
func (p *GiteeProvider) TriggerPipeline(ctx context.Context, projectID string, req *TriggerPipelineRequest) (*Pipeline, error) {
	return nil, ErrGiteePipelineNotSupported
}

// GetPipeline is not fully supported by Gitee API
func (p *GiteeProvider) GetPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error) {
	return nil, ErrGiteePipelineNotSupported
}

// ListPipelines is not fully supported by Gitee API
func (p *GiteeProvider) ListPipelines(ctx context.Context, projectID string, ref, status string, page, perPage int) ([]*Pipeline, error) {
	return nil, ErrGiteePipelineNotSupported
}

// CancelPipeline is not fully supported by Gitee API
func (p *GiteeProvider) CancelPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error) {
	return nil, ErrGiteePipelineNotSupported
}

// RetryPipeline is not fully supported by Gitee API
func (p *GiteeProvider) RetryPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error) {
	return nil, ErrGiteePipelineNotSupported
}
