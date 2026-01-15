package git

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// GetProject returns repository details
func (p *GiteeProvider) GetProject(ctx context.Context, projectID string) (*Project, error) {
	// projectID format: owner/repo
	resp, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s", projectID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gtRepo struct {
		ID            int       `json:"id"`
		Name          string    `json:"name"`
		FullName      string    `json:"full_name"`
		Description   string    `json:"description"`
		DefaultBranch string    `json:"default_branch"`
		HTMLURL       string    `json:"html_url"`
		CloneURL      string    `json:"clone_url,omitempty"`
		SSHURL        string    `json:"ssh_url"`
		Public        bool      `json:"public"`
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"updated_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gtRepo); err != nil {
		return nil, err
	}

	visibility := "private"
	if gtRepo.Public {
		visibility = "public"
	}

	return &Project{
		ID:            strconv.Itoa(gtRepo.ID),
		Name:          gtRepo.Name,
		FullPath:      gtRepo.FullName,
		Description:   gtRepo.Description,
		DefaultBranch: gtRepo.DefaultBranch,
		WebURL:        gtRepo.HTMLURL,
		CloneURL:      gtRepo.CloneURL,
		SSHCloneURL:   gtRepo.SSHURL,
		Visibility:    visibility,
		CreatedAt:     gtRepo.CreatedAt,
		UpdatedAt:     gtRepo.UpdatedAt,
	}, nil
}

// ListProjects returns user's repositories
func (p *GiteeProvider) ListProjects(ctx context.Context, page, perPage int) ([]*Project, error) {
	path := fmt.Sprintf("/user/repos?page=%d&per_page=%d&sort=updated", page, perPage)
	resp, err := p.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gtRepos []struct {
		ID            int       `json:"id"`
		Name          string    `json:"name"`
		FullName      string    `json:"full_name"`
		Description   string    `json:"description"`
		DefaultBranch string    `json:"default_branch"`
		HTMLURL       string    `json:"html_url"`
		CloneURL      string    `json:"clone_url,omitempty"`
		SSHURL        string    `json:"ssh_url"`
		Public        bool      `json:"public"`
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"updated_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gtRepos); err != nil {
		return nil, err
	}

	projects := make([]*Project, len(gtRepos))
	for i, gtr := range gtRepos {
		visibility := "private"
		if gtr.Public {
			visibility = "public"
		}
		projects[i] = &Project{
			ID:            strconv.Itoa(gtr.ID),
			Name:          gtr.Name,
			FullPath:      gtr.FullName,
			Description:   gtr.Description,
			DefaultBranch: gtr.DefaultBranch,
			WebURL:        gtr.HTMLURL,
			CloneURL:      gtr.CloneURL,
			SSHCloneURL:   gtr.SSHURL,
			Visibility:    visibility,
			CreatedAt:     gtr.CreatedAt,
			UpdatedAt:     gtr.UpdatedAt,
		}
	}

	return projects, nil
}

// SearchProjects searches for repositories
func (p *GiteeProvider) SearchProjects(ctx context.Context, query string, page, perPage int) ([]*Project, error) {
	path := fmt.Sprintf("/search/repositories?q=%s&page=%d&per_page=%d", url.QueryEscape(query), page, perPage)
	resp, err := p.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gtRepos []struct {
		ID            int       `json:"id"`
		Name          string    `json:"name"`
		FullName      string    `json:"full_name"`
		Description   string    `json:"description"`
		DefaultBranch string    `json:"default_branch"`
		HTMLURL       string    `json:"html_url"`
		CloneURL      string    `json:"clone_url,omitempty"`
		SSHURL        string    `json:"ssh_url"`
		Public        bool      `json:"public"`
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"updated_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gtRepos); err != nil {
		return nil, err
	}

	projects := make([]*Project, len(gtRepos))
	for i, gtr := range gtRepos {
		visibility := "private"
		if gtr.Public {
			visibility = "public"
		}
		projects[i] = &Project{
			ID:            strconv.Itoa(gtr.ID),
			Name:          gtr.Name,
			FullPath:      gtr.FullName,
			Description:   gtr.Description,
			DefaultBranch: gtr.DefaultBranch,
			WebURL:        gtr.HTMLURL,
			CloneURL:      gtr.CloneURL,
			SSHCloneURL:   gtr.SSHURL,
			Visibility:    visibility,
			CreatedAt:     gtr.CreatedAt,
			UpdatedAt:     gtr.UpdatedAt,
		}
	}

	return projects, nil
}
