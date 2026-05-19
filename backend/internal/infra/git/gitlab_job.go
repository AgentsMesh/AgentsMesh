package git

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"
)

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
