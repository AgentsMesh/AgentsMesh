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

// GetMergeRequest returns a specific pull request
func (p *GitHubProvider) GetMergeRequest(ctx context.Context, projectID string, mrIID int) (*MergeRequest, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repos/%s/pulls/%d", projectID, mrIID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parsePullRequest(resp.Body)
}

// ListMergeRequests returns pull requests for a repository
func (p *GitHubProvider) ListMergeRequests(ctx context.Context, projectID string, state string, page, perPage int) ([]*MergeRequest, error) {
	ghState := "all"
	if state == "opened" {
		ghState = "open"
	} else if state == "merged" || state == "closed" {
		ghState = "closed"
	}

	path := fmt.Sprintf("/repos/%s/pulls?state=%s&page=%d&per_page=%d", projectID, ghState, page, perPage)
	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghPRs []struct {
		ID     int    `json:"id"`
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		Head   struct {
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
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
		CreatedAt time.Time  `json:"created_at"`
		UpdatedAt time.Time  `json:"updated_at"`
		MergedAt  *time.Time `json:"merged_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghPRs); err != nil {
		return nil, err
	}

	mrs := make([]*MergeRequest, len(ghPRs))
	for i, ghPR := range ghPRs {
		prState := ghPR.State
		if ghPR.MergedAt != nil {
			prState = "merged"
		}

		mrs[i] = &MergeRequest{
			ID:           ghPR.ID,
			IID:          ghPR.Number,
			Title:        ghPR.Title,
			Description:  ghPR.Body,
			SourceBranch: ghPR.Head.Ref,
			TargetBranch: ghPR.Base.Ref,
			State:        prState,
			WebURL:       ghPR.HTMLURL,
			Author: &User{
				ID:        strconv.Itoa(ghPR.User.ID),
				Username:  ghPR.User.Login,
				AvatarURL: ghPR.User.AvatarURL,
			},
			CreatedAt: ghPR.CreatedAt,
			UpdatedAt: ghPR.UpdatedAt,
			MergedAt:  ghPR.MergedAt,
		}
	}

	return mrs, nil
}

// ListMergeRequestsByBranch returns pull requests filtered by head branch
func (p *GitHubProvider) ListMergeRequestsByBranch(ctx context.Context, projectID, sourceBranch, state string) ([]*MergeRequest, error) {
	ghState := "all"
	if state == "opened" {
		ghState = "open"
	} else if state == "merged" || state == "closed" {
		ghState = "closed"
	}

	path := fmt.Sprintf("/repos/%s/pulls?state=%s&head=%s", projectID, ghState, url.QueryEscape(sourceBranch))
	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghPRs []struct {
		ID     int    `json:"id"`
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		Head   struct {
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
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghPRs); err != nil {
		return nil, err
	}

	mrs := make([]*MergeRequest, len(ghPRs))
	for i, ghPR := range ghPRs {
		prState := ghPR.State
		if ghPR.MergedAt != nil {
			prState = "merged"
		}

		mrs[i] = &MergeRequest{
			ID:             ghPR.ID,
			IID:            ghPR.Number,
			Title:          ghPR.Title,
			Description:    ghPR.Body,
			SourceBranch:   ghPR.Head.Ref,
			TargetBranch:   ghPR.Base.Ref,
			State:          prState,
			WebURL:         ghPR.HTMLURL,
			MergeCommitSHA: ghPR.MergeCommitSHA,
			MergedAt:       ghPR.MergedAt,
			Author: &User{
				ID:        strconv.Itoa(ghPR.User.ID),
				Username:  ghPR.User.Login,
				AvatarURL: ghPR.User.AvatarURL,
			},
			CreatedAt: ghPR.CreatedAt,
			UpdatedAt: ghPR.UpdatedAt,
		}
	}

	return mrs, nil
}

// CreateMergeRequest creates a new pull request
func (p *GitHubProvider) CreateMergeRequest(ctx context.Context, req *CreateMRRequest) (*MergeRequest, error) {
	body := fmt.Sprintf(`{"title":"%s","body":"%s","head":"%s","base":"%s"}`,
		req.Title, req.Description, req.SourceBranch, req.TargetBranch)

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/repos/%s/pulls", req.ProjectID), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parsePullRequest(resp.Body)
}

// UpdateMergeRequest updates a pull request
func (p *GitHubProvider) UpdateMergeRequest(ctx context.Context, projectID string, mrIID int, title, description string) (*MergeRequest, error) {
	body := fmt.Sprintf(`{"title":"%s","body":"%s"}`, title, description)

	resp, err := p.doRequest(ctx, "PATCH", fmt.Sprintf("/repos/%s/pulls/%d", projectID, mrIID), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parsePullRequest(resp.Body)
}

// MergeMergeRequest merges a pull request
func (p *GitHubProvider) MergeMergeRequest(ctx context.Context, projectID string, mrIID int) (*MergeRequest, error) {
	resp, err := p.doRequest(ctx, "PUT", fmt.Sprintf("/repos/%s/pulls/%d/merge", projectID, mrIID), nil)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	return p.GetMergeRequest(ctx, projectID, mrIID)
}

// CloseMergeRequest closes a pull request
func (p *GitHubProvider) CloseMergeRequest(ctx context.Context, projectID string, mrIID int) (*MergeRequest, error) {
	body := `{"state":"closed"}`

	resp, err := p.doRequest(ctx, "PATCH", fmt.Sprintf("/repos/%s/pulls/%d", projectID, mrIID), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return p.parsePullRequest(resp.Body)
}

func (p *GitHubProvider) parsePullRequest(r io.Reader) (*MergeRequest, error) {
	var ghPR struct {
		ID     int    `json:"id"`
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		Head   struct {
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
			AvatarURL string `json:"avatar_url"`
		} `json:"user"`
		CreatedAt time.Time  `json:"created_at"`
		UpdatedAt time.Time  `json:"updated_at"`
		MergedAt  *time.Time `json:"merged_at"`
	}

	if err := json.NewDecoder(r).Decode(&ghPR); err != nil {
		return nil, err
	}

	state := ghPR.State
	if ghPR.MergedAt != nil {
		state = "merged"
	}

	return &MergeRequest{
		ID:           ghPR.ID,
		IID:          ghPR.Number,
		Title:        ghPR.Title,
		Description:  ghPR.Body,
		SourceBranch: ghPR.Head.Ref,
		TargetBranch: ghPR.Base.Ref,
		State:        state,
		WebURL:       ghPR.HTMLURL,
		Author: &User{
			ID:        strconv.Itoa(ghPR.User.ID),
			Username:  ghPR.User.Login,
			AvatarURL: ghPR.User.AvatarURL,
		},
		CreatedAt: ghPR.CreatedAt,
		UpdatedAt: ghPR.UpdatedAt,
		MergedAt:  ghPR.MergedAt,
	}, nil
}
