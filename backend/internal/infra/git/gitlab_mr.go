package git

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GetMergeRequest returns a specific merge request
func (p *GitLabProvider) GetMergeRequest(ctx context.Context, projectID string, mrIID int) (*MergeRequest, error) {
	encodedID := url.PathEscape(projectID)
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s/merge_requests/%d", encodedID, mrIID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseGitLabMergeRequest(resp.Body)
}

// ListMergeRequests returns merge requests for a project
func (p *GitLabProvider) ListMergeRequests(ctx context.Context, projectID string, state string, page, perPage int) ([]*MergeRequest, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/merge_requests?page=%d&per_page=%d", encodedID, page, perPage)
	if state != "" {
		path += "&state=" + state
	}

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glMRs []struct {
		ID           int       `json:"id"`
		IID          int       `json:"iid"`
		Title        string    `json:"title"`
		Description  string    `json:"description"`
		SourceBranch string    `json:"source_branch"`
		TargetBranch string    `json:"target_branch"`
		State        string    `json:"state"`
		WebURL       string    `json:"web_url"`
		Author       struct {
			ID        int    `json:"id"`
			Username  string `json:"username"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"author"`
		CreatedAt time.Time  `json:"created_at"`
		UpdatedAt time.Time  `json:"updated_at"`
		MergedAt  *time.Time `json:"merged_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glMRs); err != nil {
		return nil, err
	}

	mrs := make([]*MergeRequest, len(glMRs))
	for i, glMR := range glMRs {
		mrs[i] = &MergeRequest{
			ID:           glMR.ID,
			IID:          glMR.IID,
			Title:        glMR.Title,
			Description:  glMR.Description,
			SourceBranch: glMR.SourceBranch,
			TargetBranch: glMR.TargetBranch,
			State:        glMR.State,
			WebURL:       glMR.WebURL,
			Author: &User{
				ID:        strconv.Itoa(glMR.Author.ID),
				Username:  glMR.Author.Username,
				Name:      glMR.Author.Name,
				AvatarURL: glMR.Author.AvatarURL,
			},
			CreatedAt: glMR.CreatedAt,
			UpdatedAt: glMR.UpdatedAt,
			MergedAt:  glMR.MergedAt,
		}
	}

	return mrs, nil
}

// ListMergeRequestsByBranch returns merge requests filtered by source branch
func (p *GitLabProvider) ListMergeRequestsByBranch(ctx context.Context, projectID, sourceBranch, state string) ([]*MergeRequest, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/merge_requests?source_branch=%s", encodedID, url.QueryEscape(sourceBranch))
	if state != "" && state != "all" {
		path += "&state=" + state
	}

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glMRs []struct {
		ID             int        `json:"id"`
		IID            int        `json:"iid"`
		Title          string     `json:"title"`
		Description    string     `json:"description"`
		SourceBranch   string     `json:"source_branch"`
		TargetBranch   string     `json:"target_branch"`
		State          string     `json:"state"`
		WebURL         string     `json:"web_url"`
		MergeCommitSHA string     `json:"merge_commit_sha"`
		Pipeline       *struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
			WebURL string `json:"web_url"`
		} `json:"pipeline"`
		Author struct {
			ID        int    `json:"id"`
			Username  string `json:"username"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"author"`
		CreatedAt time.Time  `json:"created_at"`
		UpdatedAt time.Time  `json:"updated_at"`
		MergedAt  *time.Time `json:"merged_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glMRs); err != nil {
		return nil, err
	}

	mrs := make([]*MergeRequest, len(glMRs))
	for i, glMR := range glMRs {
		mr := &MergeRequest{
			ID:             glMR.ID,
			IID:            glMR.IID,
			Title:          glMR.Title,
			Description:    glMR.Description,
			SourceBranch:   glMR.SourceBranch,
			TargetBranch:   glMR.TargetBranch,
			State:          glMR.State,
			WebURL:         glMR.WebURL,
			MergeCommitSHA: glMR.MergeCommitSHA,
			Author: &User{
				ID:        strconv.Itoa(glMR.Author.ID),
				Username:  glMR.Author.Username,
				Name:      glMR.Author.Name,
				AvatarURL: glMR.Author.AvatarURL,
			},
			CreatedAt: glMR.CreatedAt,
			UpdatedAt: glMR.UpdatedAt,
			MergedAt:  glMR.MergedAt,
		}
		if glMR.Pipeline != nil {
			mr.PipelineID = glMR.Pipeline.ID
			mr.PipelineStatus = glMR.Pipeline.Status
			mr.PipelineURL = glMR.Pipeline.WebURL
		}
		mrs[i] = mr
	}

	return mrs, nil
}

// CreateMergeRequest creates a new merge request
func (p *GitLabProvider) CreateMergeRequest(ctx context.Context, req *CreateMRRequest) (*MergeRequest, error) {
	encodedID := url.PathEscape(req.ProjectID)
	path := fmt.Sprintf("/projects/%s/merge_requests", encodedID)

	body := fmt.Sprintf(`{"source_branch":"%s","target_branch":"%s","title":"%s","description":"%s"}`,
		req.SourceBranch, req.TargetBranch, req.Title, req.Description)

	resp, err := p.doRequest(ctx, "POST", path, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseGitLabMergeRequest(resp.Body)
}

// UpdateMergeRequest updates a merge request
func (p *GitLabProvider) UpdateMergeRequest(ctx context.Context, projectID string, mrIID int, title, description string) (*MergeRequest, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/merge_requests/%d", encodedID, mrIID)

	body := fmt.Sprintf(`{"title":"%s","description":"%s"}`, title, description)

	resp, err := p.doRequest(ctx, "PUT", path, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseGitLabMergeRequest(resp.Body)
}

// MergeMergeRequest merges a merge request
func (p *GitLabProvider) MergeMergeRequest(ctx context.Context, projectID string, mrIID int) (*MergeRequest, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/merge_requests/%d/merge", encodedID, mrIID)

	resp, err := p.doRequest(ctx, "PUT", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseGitLabMergeRequest(resp.Body)
}

// CloseMergeRequest closes a merge request
func (p *GitLabProvider) CloseMergeRequest(ctx context.Context, projectID string, mrIID int) (*MergeRequest, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/merge_requests/%d", encodedID, mrIID)

	body := `{"state_event":"close"}`

	resp, err := p.doRequest(ctx, "PUT", path, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parseGitLabMergeRequest(resp.Body)
}

func (p *GitLabProvider) parseGitLabMergeRequest(r io.Reader) (*MergeRequest, error) {
	var glMR struct {
		ID           int       `json:"id"`
		IID          int       `json:"iid"`
		Title        string    `json:"title"`
		Description  string    `json:"description"`
		SourceBranch string    `json:"source_branch"`
		TargetBranch string    `json:"target_branch"`
		State        string    `json:"state"`
		WebURL       string    `json:"web_url"`
		Author       struct {
			ID        int    `json:"id"`
			Username  string `json:"username"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"author"`
		CreatedAt time.Time  `json:"created_at"`
		UpdatedAt time.Time  `json:"updated_at"`
		MergedAt  *time.Time `json:"merged_at"`
	}

	if err := json.NewDecoder(r).Decode(&glMR); err != nil {
		return nil, err
	}

	return &MergeRequest{
		ID:           glMR.ID,
		IID:          glMR.IID,
		Title:        glMR.Title,
		Description:  glMR.Description,
		SourceBranch: glMR.SourceBranch,
		TargetBranch: glMR.TargetBranch,
		State:        glMR.State,
		WebURL:       glMR.WebURL,
		Author: &User{
			ID:        strconv.Itoa(glMR.Author.ID),
			Username:  glMR.Author.Username,
			Name:      glMR.Author.Name,
			AvatarURL: glMR.Author.AvatarURL,
		},
		CreatedAt: glMR.CreatedAt,
		UpdatedAt: glMR.UpdatedAt,
		MergedAt:  glMR.MergedAt,
	}, nil
}
