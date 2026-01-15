package git

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// GetJob returns a specific job
func (p *GitHubProvider) GetJob(ctx context.Context, projectID string, jobID int) (*Job, error) {
	path := fmt.Sprintf("/repos/%s/actions/jobs/%d", projectID, jobID)

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseGitHubJob(resp.Body)
}

// ListPipelineJobs returns jobs for a workflow run
func (p *GitHubProvider) ListPipelineJobs(ctx context.Context, projectID string, pipelineID int) ([]*Job, error) {
	path := fmt.Sprintf("/repos/%s/actions/runs/%d/jobs", projectID, pipelineID)

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Jobs []struct {
			ID          int        `json:"id"`
			Name        string     `json:"name"`
			Status      string     `json:"status"`
			Conclusion  string     `json:"conclusion"`
			HTMLURL     string     `json:"html_url"`
			RunID       int        `json:"run_id"`
			StartedAt   *time.Time `json:"started_at"`
			CompletedAt *time.Time `json:"completed_at"`
		} `json:"jobs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	jobs := make([]*Job, len(result.Jobs))
	for i, ghJob := range result.Jobs {
		var duration float64
		if ghJob.StartedAt != nil && ghJob.CompletedAt != nil {
			duration = ghJob.CompletedAt.Sub(*ghJob.StartedAt).Seconds()
		}

		jobs[i] = &Job{
			ID:         ghJob.ID,
			Name:       ghJob.Name,
			Status:     p.mapGitHubStatus(ghJob.Status, ghJob.Conclusion),
			PipelineID: ghJob.RunID,
			WebURL:     ghJob.HTMLURL,
			CreatedAt:  time.Time{},
			StartedAt:  ghJob.StartedAt,
			FinishedAt: ghJob.CompletedAt,
			Duration:   duration,
		}
	}

	return jobs, nil
}

// RetryJob re-runs a specific job
func (p *GitHubProvider) RetryJob(ctx context.Context, projectID string, jobID int) (*Job, error) {
	path := fmt.Sprintf("/repos/%s/actions/jobs/%d/rerun", projectID, jobID)

	resp, err := p.doRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	return p.GetJob(ctx, projectID, jobID)
}

// CancelJob cancels a workflow run (GitHub doesn't support cancelling individual jobs)
func (p *GitHubProvider) CancelJob(ctx context.Context, projectID string, jobID int) (*Job, error) {
	return p.GetJob(ctx, projectID, jobID)
}

// GetJobTrace returns the job logs
func (p *GitHubProvider) GetJobTrace(ctx context.Context, projectID string, jobID int) (string, error) {
	path := fmt.Sprintf("/repos/%s/actions/jobs/%d/logs", projectID, jobID)

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

// GetJobArtifact downloads a specific artifact
func (p *GitHubProvider) GetJobArtifact(ctx context.Context, projectID string, jobID int, artifactPath string) ([]byte, error) {
	return nil, ErrNotFound
}

// DownloadJobArtifacts downloads artifacts for a workflow run
func (p *GitHubProvider) DownloadJobArtifacts(ctx context.Context, projectID string, jobID int) ([]byte, error) {
	return nil, ErrNotFound
}

func (p *GitHubProvider) parseGitHubJob(r io.Reader) (*Job, error) {
	var ghJob struct {
		ID          int        `json:"id"`
		Name        string     `json:"name"`
		Status      string     `json:"status"`
		Conclusion  string     `json:"conclusion"`
		HTMLURL     string     `json:"html_url"`
		RunID       int        `json:"run_id"`
		StartedAt   *time.Time `json:"started_at"`
		CompletedAt *time.Time `json:"completed_at"`
	}

	if err := json.NewDecoder(r).Decode(&ghJob); err != nil {
		return nil, err
	}

	var duration float64
	if ghJob.StartedAt != nil && ghJob.CompletedAt != nil {
		duration = ghJob.CompletedAt.Sub(*ghJob.StartedAt).Seconds()
	}

	return &Job{
		ID:         ghJob.ID,
		Name:       ghJob.Name,
		Status:     p.mapGitHubStatus(ghJob.Status, ghJob.Conclusion),
		PipelineID: ghJob.RunID,
		WebURL:     ghJob.HTMLURL,
		CreatedAt:  time.Time{},
		StartedAt:  ghJob.StartedAt,
		FinishedAt: ghJob.CompletedAt,
		Duration:   duration,
	}, nil
}
