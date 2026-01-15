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

// TriggerPipeline triggers a new pipeline
func (p *GitLabProvider) TriggerPipeline(ctx context.Context, projectID string, req *TriggerPipelineRequest) (*Pipeline, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/pipeline", encodedID)

	bodyData := map[string]interface{}{
		"ref": req.Ref,
	}
	if len(req.Variables) > 0 {
		vars := make([]map[string]string, 0, len(req.Variables))
		for k, v := range req.Variables {
			vars = append(vars, map[string]string{"key": k, "value": v})
		}
		bodyData["variables"] = vars
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

	return p.parseGitLabPipeline(resp.Body, projectID)
}

// GetPipeline returns a specific pipeline
func (p *GitLabProvider) GetPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/pipelines/%d", encodedID, pipelineID)

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseGitLabPipeline(resp.Body, projectID)
}

// ListPipelines returns pipelines for a project
func (p *GitLabProvider) ListPipelines(ctx context.Context, projectID string, ref, status string, page, perPage int) ([]*Pipeline, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/pipelines?page=%d&per_page=%d", encodedID, page, perPage)
	if ref != "" {
		path += "&ref=" + url.QueryEscape(ref)
	}
	if status != "" {
		path += "&status=" + status
	}

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glPipelines []struct {
		ID         int        `json:"id"`
		IID        int        `json:"iid"`
		Ref        string     `json:"ref"`
		SHA        string     `json:"sha"`
		Status     string     `json:"status"`
		Source     string     `json:"source"`
		WebURL     string     `json:"web_url"`
		CreatedAt  time.Time  `json:"created_at"`
		UpdatedAt  time.Time  `json:"updated_at"`
		StartedAt  *time.Time `json:"started_at"`
		FinishedAt *time.Time `json:"finished_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glPipelines); err != nil {
		return nil, err
	}

	pipelines := make([]*Pipeline, len(glPipelines))
	for i, glp := range glPipelines {
		pipelines[i] = &Pipeline{
			ID:         glp.ID,
			IID:        glp.IID,
			ProjectID:  projectID,
			Ref:        glp.Ref,
			SHA:        glp.SHA,
			Status:     glp.Status,
			Source:     glp.Source,
			WebURL:     glp.WebURL,
			CreatedAt:  glp.CreatedAt,
			UpdatedAt:  glp.UpdatedAt,
			StartedAt:  glp.StartedAt,
			FinishedAt: glp.FinishedAt,
		}
	}

	return pipelines, nil
}

// CancelPipeline cancels a pipeline
func (p *GitLabProvider) CancelPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/pipelines/%d/cancel", encodedID, pipelineID)

	resp, err := p.doRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseGitLabPipeline(resp.Body, projectID)
}

// RetryPipeline retries a pipeline
func (p *GitLabProvider) RetryPipeline(ctx context.Context, projectID string, pipelineID int) (*Pipeline, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/pipelines/%d/retry", encodedID, pipelineID)

	resp, err := p.doRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseGitLabPipeline(resp.Body, projectID)
}

func (p *GitLabProvider) parseGitLabPipeline(r io.Reader, projectID string) (*Pipeline, error) {
	var glp struct {
		ID         int        `json:"id"`
		IID        int        `json:"iid"`
		Ref        string     `json:"ref"`
		SHA        string     `json:"sha"`
		Status     string     `json:"status"`
		Source     string     `json:"source"`
		WebURL     string     `json:"web_url"`
		CreatedAt  time.Time  `json:"created_at"`
		UpdatedAt  time.Time  `json:"updated_at"`
		StartedAt  *time.Time `json:"started_at"`
		FinishedAt *time.Time `json:"finished_at"`
	}

	if err := json.NewDecoder(r).Decode(&glp); err != nil {
		return nil, err
	}

	return &Pipeline{
		ID:         glp.ID,
		IID:        glp.IID,
		ProjectID:  projectID,
		Ref:        glp.Ref,
		SHA:        glp.SHA,
		Status:     glp.Status,
		Source:     glp.Source,
		WebURL:     glp.WebURL,
		CreatedAt:  glp.CreatedAt,
		UpdatedAt:  glp.UpdatedAt,
		StartedAt:  glp.StartedAt,
		FinishedAt: glp.FinishedAt,
	}, nil
}
