package git

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

// TriggerPipeline triggers a workflow run (GitHub Actions)
func (p *GitHubProvider) TriggerPipeline(ctx context.Context, projectID string, req *TriggerPipelineRequest) (*Pipeline, error) {
	path := fmt.Sprintf("/repos/%s/actions/workflows/ci.yml/dispatches", projectID)

	bodyData := map[string]interface{}{
		"ref": req.Ref,
	}
	if len(req.Variables) > 0 {
		bodyData["inputs"] = req.Variables
	}

	bodyBytes, err := json.Marshal(bodyData)
	if err != nil {
		return nil, err
	}

	resp, err := p.doRequest(ctx, "POST", path, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	runs, err := p.ListPipelines(ctx, projectID, req.Ref, "", 1, 1)
	if err != nil {
		return nil, err
	}
	if len(runs) > 0 {
		return runs[0], nil
	}

	return &Pipeline{
		ProjectID: projectID,
		Ref:       req.Ref,
		Status:    PipelineStatusPending,
	}, nil
}

// GetPipeline returns a specific workflow run
func (p *GitHubProvider) GetPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error) {
	path := fmt.Sprintf("/repos/%s/actions/runs/%d", projectID, pipelineID)

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseWorkflowRun(resp.Body, projectID)
}

// ListPipelines returns workflow runs for a repository
func (p *GitHubProvider) ListPipelines(ctx context.Context, projectID string, ref, status string, page, perPage int) ([]*Pipeline, error) {
	path := fmt.Sprintf("/repos/%s/actions/runs?page=%d&per_page=%d", projectID, page, perPage)
	if ref != "" {
		path += "&branch=" + url.QueryEscape(ref)
	}
	if status != "" {
		path += "&status=" + status
	}

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		WorkflowRuns []struct {
			ID           int        `json:"id"`
			RunNumber    int        `json:"run_number"`
			HeadBranch   string     `json:"head_branch"`
			HeadSHA      string     `json:"head_sha"`
			Status       string     `json:"status"`
			Conclusion   string     `json:"conclusion"`
			Event        string     `json:"event"`
			HTMLURL      string     `json:"html_url"`
			CreatedAt    time.Time  `json:"created_at"`
			UpdatedAt    time.Time  `json:"updated_at"`
			RunStartedAt *time.Time `json:"run_started_at"`
		} `json:"workflow_runs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	pipelines := make([]*Pipeline, len(result.WorkflowRuns))
	for i, run := range result.WorkflowRuns {
		pipelines[i] = &Pipeline{
			ID:        run.ID,
			IID:       run.RunNumber,
			ProjectID: projectID,
			Ref:       run.HeadBranch,
			SHA:       run.HeadSHA,
			Status:    p.mapGitHubStatus(run.Status, run.Conclusion),
			Source:    run.Event,
			WebURL:    run.HTMLURL,
			CreatedAt: run.CreatedAt,
			UpdatedAt: run.UpdatedAt,
			StartedAt: run.RunStartedAt,
		}
	}

	return pipelines, nil
}

// CancelPipeline cancels a workflow run
func (p *GitHubProvider) CancelPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error) {
	path := fmt.Sprintf("/repos/%s/actions/runs/%d/cancel", projectID, pipelineID)

	resp, err := p.doRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	return p.GetPipeline(ctx, projectID, pipelineID)
}

// RetryPipeline re-runs a workflow
func (p *GitHubProvider) RetryPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error) {
	path := fmt.Sprintf("/repos/%s/actions/runs/%d/rerun", projectID, pipelineID)

	resp, err := p.doRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	return p.GetPipeline(ctx, projectID, pipelineID)
}

func (p *GitHubProvider) parseWorkflowRun(r io.Reader, projectID string) (*Pipeline, error) {
	var run struct {
		ID           int        `json:"id"`
		RunNumber    int        `json:"run_number"`
		HeadBranch   string     `json:"head_branch"`
		HeadSHA      string     `json:"head_sha"`
		Status       string     `json:"status"`
		Conclusion   string     `json:"conclusion"`
		Event        string     `json:"event"`
		HTMLURL      string     `json:"html_url"`
		CreatedAt    time.Time  `json:"created_at"`
		UpdatedAt    time.Time  `json:"updated_at"`
		RunStartedAt *time.Time `json:"run_started_at"`
	}

	if err := json.NewDecoder(r).Decode(&run); err != nil {
		return nil, err
	}

	return &Pipeline{
		ID:        run.ID,
		IID:       run.RunNumber,
		ProjectID: projectID,
		Ref:       run.HeadBranch,
		SHA:       run.HeadSHA,
		Status:    p.mapGitHubStatus(run.Status, run.Conclusion),
		Source:    run.Event,
		WebURL:    run.HTMLURL,
		CreatedAt: run.CreatedAt,
		UpdatedAt: run.UpdatedAt,
		StartedAt: run.RunStartedAt,
	}, nil
}

func (p *GitHubProvider) mapGitHubStatus(status, conclusion string) string {
	if status == "completed" {
		switch conclusion {
		case "success":
			return PipelineStatusSuccess
		case "failure":
			return PipelineStatusFailed
		case "cancelled":
			return PipelineStatusCanceled
		case "skipped":
			return PipelineStatusSkipped
		default:
			return PipelineStatusFailed
		}
	}
	switch status {
	case "queued":
		return PipelineStatusPending
	case "in_progress":
		return PipelineStatusRunning
	case "waiting":
		return PipelineStatusManual
	default:
		return PipelineStatusPending
	}
}
