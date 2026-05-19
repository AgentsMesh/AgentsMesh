package git

import (
	"context"
	"fmt"
)

var ErrGiteePipelineNotSupported = fmt.Errorf("gitee pipeline API not fully supported")

func (p *GiteeProvider) TriggerPipeline(ctx context.Context, projectID string, req *TriggerPipelineRequest) (*Pipeline, error) {
	return nil, ErrGiteePipelineNotSupported
}

func (p *GiteeProvider) GetPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error) {
	return nil, ErrGiteePipelineNotSupported
}

func (p *GiteeProvider) ListPipelines(ctx context.Context, projectID string, ref, status string, page, perPage int) ([]*Pipeline, error) {
	return nil, ErrGiteePipelineNotSupported
}

func (p *GiteeProvider) CancelPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error) {
	return nil, ErrGiteePipelineNotSupported
}

func (p *GiteeProvider) RetryPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error) {
	return nil, ErrGiteePipelineNotSupported
}
