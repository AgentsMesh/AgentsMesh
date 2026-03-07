package git

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GetMergeRequest returns a specific pull request
func (p *GiteeProvider) GetMergeRequest(ctx context.Context, projectID string, mrIID int) (*MergeRequest, error) {
	resp, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/pulls/%d", projectID, mrIID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parsePullRequest(resp.Body)
}

// ListMergeRequests returns pull requests for a repository
func (p *GiteeProvider) ListMergeRequests(ctx context.Context, projectID string, state string, page, perPage int) ([]*MergeRequest, error) {
	gtState := "all"
	switch state {
	case "opened":
		gtState = "open"
	case "merged":
		gtState = "merged"
	case "closed":
		gtState = "closed"
	}

	path := fmt.Sprintf("/repos/%s/pulls?state=%s&page=%d&per_page=%d", projectID, gtState, page, perPage)
	resp, err := p.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gtPRs []struct {
		ID        int    `json:"id"`
		Number    int    `json:"number"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		Head      struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		State   string `json:"state"`
		HTMLURL string `json:"html_url"`
		User    struct {
			ID        int    `json:"id"`
			Login     string `json:"login"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
		CreatedAt time.Time  `json:"created_at"`
		UpdatedAt time.Time  `json:"updated_at"`
		MergedAt  *time.Time `json:"merged_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gtPRs); err != nil {
		return nil, err
	}

	mrs := make([]*MergeRequest, len(gtPRs))
	for i, gtPR := range gtPRs {
		mrs[i] = &MergeRequest{
			ID:           gtPR.ID,
			IID:          gtPR.Number,
			Title:        gtPR.Title,
			Description:  gtPR.Body,
			SourceBranch: gtPR.Head.Ref,
			TargetBranch: gtPR.Base.Ref,
			State:        gtPR.State,
			WebURL:       gtPR.HTMLURL,
			Author: &User{
				ID:        strconv.Itoa(gtPR.User.ID),
				Username:  gtPR.User.Login,
				Name:      gtPR.User.Name,
				AvatarURL: gtPR.User.AvatarURL,
			},
			CreatedAt: gtPR.CreatedAt,
			UpdatedAt: gtPR.UpdatedAt,
			MergedAt:  gtPR.MergedAt,
		}
	}

	return mrs, nil
}

// ListMergeRequestsByBranch returns pull requests filtered by source branch
func (p *GiteeProvider) ListMergeRequestsByBranch(ctx context.Context, projectID, sourceBranch, state string) ([]*MergeRequest, error) {
	gtState := "all"
	switch state {
	case "opened":
		gtState = "open"
	case "merged":
		gtState = "merged"
	case "closed":
		gtState = "closed"
	}

	// Gitee supports head parameter for filtering by source branch
	path := fmt.Sprintf("/repos/%s/pulls?state=%s&head=%s", projectID, gtState, url.QueryEscape(sourceBranch))
	resp, err := p.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gtPRs []struct {
		ID        int    `json:"id"`
		Number    int    `json:"number"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		Head      struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		State          string     `json:"state"`
		HTMLURL        string     `json:"html_url"`
		MergeCommitSHA string     `json:"merge_commit_sha"`
		MergedAt       *time.Time `json:"merged_at"`
		User           struct {
			ID        int    `json:"id"`
			Login     string `json:"login"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gtPRs); err != nil {
		return nil, err
	}

	mrs := make([]*MergeRequest, len(gtPRs))
	for i, gtPR := range gtPRs {
		mrs[i] = &MergeRequest{
			ID:             gtPR.ID,
			IID:            gtPR.Number,
			Title:          gtPR.Title,
			Description:    gtPR.Body,
			SourceBranch:   gtPR.Head.Ref,
			TargetBranch:   gtPR.Base.Ref,
			State:          gtPR.State,
			WebURL:         gtPR.HTMLURL,
			MergeCommitSHA: gtPR.MergeCommitSHA,
			MergedAt:       gtPR.MergedAt,
			Author: &User{
				ID:        strconv.Itoa(gtPR.User.ID),
				Username:  gtPR.User.Login,
				Name:      gtPR.User.Name,
				AvatarURL: gtPR.User.AvatarURL,
			},
			CreatedAt: gtPR.CreatedAt,
			UpdatedAt: gtPR.UpdatedAt,
		}
	}

	return mrs, nil
}

// CreateMergeRequest creates a new pull request
func (p *GiteeProvider) CreateMergeRequest(ctx context.Context, req *CreateMRRequest) (*MergeRequest, error) {
	body := fmt.Sprintf(`{"title":"%s","body":"%s","head":"%s","base":"%s"}`,
		req.Title, req.Description, req.SourceBranch, req.TargetBranch)

	resp, err := p.doRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/pulls", req.ProjectID), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parsePullRequest(resp.Body)
}

// UpdateMergeRequest updates a pull request
func (p *GiteeProvider) UpdateMergeRequest(ctx context.Context, projectID string, mrIID int, title, description string) (*MergeRequest, error) {
	body := fmt.Sprintf(`{"title":"%s","body":"%s"}`, title, description)

	resp, err := p.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/repos/%s/pulls/%d", projectID, mrIID), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parsePullRequest(resp.Body)
}

// MergeMergeRequest merges a pull request
func (p *GiteeProvider) MergeMergeRequest(ctx context.Context, projectID string, mrIID int) (*MergeRequest, error) {
	resp, err := p.doRequest(ctx, http.MethodPut, fmt.Sprintf("/repos/%s/pulls/%d/merge", projectID, mrIID), nil)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	return p.GetMergeRequest(ctx, projectID, mrIID)
}

// CloseMergeRequest closes a pull request
func (p *GiteeProvider) CloseMergeRequest(ctx context.Context, projectID string, mrIID int) (*MergeRequest, error) {
	body := `{"state":"closed"}`

	resp, err := p.doRequest(ctx, http.MethodPatch, fmt.Sprintf("/repos/%s/pulls/%d", projectID, mrIID), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parsePullRequest(resp.Body)
}

func (p *GiteeProvider) parsePullRequest(r io.Reader) (*MergeRequest, error) {
	var gtPR struct {
		ID        int    `json:"id"`
		Number    int    `json:"number"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		Head      struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		State   string `json:"state"`
		HTMLURL string `json:"html_url"`
		User    struct {
			ID        int    `json:"id"`
			Login     string `json:"login"`
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
		CreatedAt time.Time  `json:"created_at"`
		UpdatedAt time.Time  `json:"updated_at"`
		MergedAt  *time.Time `json:"merged_at"`
	}

	if err := json.NewDecoder(r).Decode(&gtPR); err != nil {
		return nil, err
	}

	return &MergeRequest{
		ID:           gtPR.ID,
		IID:          gtPR.Number,
		Title:        gtPR.Title,
		Description:  gtPR.Body,
		SourceBranch: gtPR.Head.Ref,
		TargetBranch: gtPR.Base.Ref,
		State:        gtPR.State,
		WebURL:       gtPR.HTMLURL,
		Author: &User{
			ID:        strconv.Itoa(gtPR.User.ID),
			Username:  gtPR.User.Login,
			Name:      gtPR.User.Name,
			AvatarURL: gtPR.User.AvatarURL,
		},
		CreatedAt: gtPR.CreatedAt,
		UpdatedAt: gtPR.UpdatedAt,
		MergedAt:  gtPR.MergedAt,
	}, nil
}
