package git

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"
)

// GetJob returns a specific job
func (p *GitLabProvider) GetJob(ctx context.Context, projectID string, jobID int) (*Job, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/jobs/%d", encodedID, jobID)

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseGitLabJob(resp.Body)
}

// ListPipelineJobs returns jobs for a pipeline
func (p *GitLabProvider) ListPipelineJobs(ctx context.Context, projectID string, pipelineID int) ([]*Job, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/pipelines/%d/jobs", encodedID, pipelineID)

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glJobs []struct {
		ID           int        `json:"id"`
		Name         string     `json:"name"`
		Stage        string     `json:"stage"`
		Status       string     `json:"status"`
		Ref          string     `json:"ref"`
		WebURL       string     `json:"web_url"`
		AllowFailure bool       `json:"allow_failure"`
		Duration     float64    `json:"duration"`
		Pipeline     struct {
			ID int `json:"id"`
		} `json:"pipeline"`
		CreatedAt  time.Time  `json:"created_at"`
		StartedAt  *time.Time `json:"started_at"`
		FinishedAt *time.Time `json:"finished_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glJobs); err != nil {
		return nil, err
	}

	jobs := make([]*Job, len(glJobs))
	for i, glj := range glJobs {
		jobs[i] = &Job{
			ID:           glj.ID,
			Name:         glj.Name,
			Stage:        glj.Stage,
			Status:       glj.Status,
			Ref:          glj.Ref,
			PipelineID:   glj.Pipeline.ID,
			WebURL:       glj.WebURL,
			AllowFailure: glj.AllowFailure,
			Duration:     glj.Duration,
			CreatedAt:    glj.CreatedAt,
			StartedAt:    glj.StartedAt,
			FinishedAt:   glj.FinishedAt,
		}
	}

	return jobs, nil
}

// RetryJob retries a job
func (p *GitLabProvider) RetryJob(ctx context.Context, projectID string, jobID int) (*Job, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/jobs/%d/retry", encodedID, jobID)

	resp, err := p.doRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseGitLabJob(resp.Body)
}

// CancelJob cancels a job
func (p *GitLabProvider) CancelJob(ctx context.Context, projectID string, jobID int) (*Job, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/jobs/%d/cancel", encodedID, jobID)

	resp, err := p.doRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseGitLabJob(resp.Body)
}

// GetJobTrace returns the job log (trace)
func (p *GitLabProvider) GetJobTrace(ctx context.Context, projectID string, jobID int) (string, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/jobs/%d/trace", encodedID, jobID)

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// GetJobArtifact downloads a specific artifact file from a job
func (p *GitLabProvider) GetJobArtifact(ctx context.Context, projectID string, jobID int, artifactPath string) ([]byte, error) {
	encodedID := url.PathEscape(projectID)
	encodedPath := url.PathEscape(artifactPath)
	path := fmt.Sprintf("/projects/%s/jobs/%d/artifacts/%s", encodedID, jobID, encodedPath)

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// DownloadJobArtifacts downloads the complete artifacts archive from a job
func (p *GitLabProvider) DownloadJobArtifacts(ctx context.Context, projectID string, jobID int) ([]byte, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/jobs/%d/artifacts", encodedID, jobID)

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (p *GitLabProvider) parseGitLabJob(r io.Reader) (*Job, error) {
	var glj struct {
		ID           int        `json:"id"`
		Name         string     `json:"name"`
		Stage        string     `json:"stage"`
		Status       string     `json:"status"`
		Ref          string     `json:"ref"`
		WebURL       string     `json:"web_url"`
		AllowFailure bool       `json:"allow_failure"`
		Duration     float64    `json:"duration"`
		Pipeline     struct {
			ID int `json:"id"`
		} `json:"pipeline"`
		CreatedAt  time.Time  `json:"created_at"`
		StartedAt  *time.Time `json:"started_at"`
		FinishedAt *time.Time `json:"finished_at"`
	}

	if err := json.NewDecoder(r).Decode(&glj); err != nil {
		return nil, err
	}

	return &Job{
		ID:           glj.ID,
		Name:         glj.Name,
		Stage:        glj.Stage,
		Status:       glj.Status,
		Ref:          glj.Ref,
		PipelineID:   glj.Pipeline.ID,
		WebURL:       glj.WebURL,
		AllowFailure: glj.AllowFailure,
		Duration:     glj.Duration,
		CreatedAt:    glj.CreatedAt,
		StartedAt:    glj.StartedAt,
		FinishedAt:   glj.FinishedAt,
	}, nil
}
