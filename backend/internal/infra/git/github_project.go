package git

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// GetProject returns repository details
func (p *GitHubProvider) GetProject(ctx context.Context, projectID string) (*Project, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repos/%s", projectID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghRepo struct {
		ID            int       `json:"id"`
		Name          string    `json:"name"`
		FullName      string    `json:"full_name"`
		Description   string    `json:"description"`
		DefaultBranch string    `json:"default_branch"`
		HTMLURL       string    `json:"html_url"`
		CloneURL      string    `json:"clone_url"`
		SSHURL        string    `json:"ssh_url"`
		Visibility    string    `json:"visibility"`
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"updated_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghRepo); err != nil {
		return nil, err
	}

	return &Project{
		ID:            strconv.Itoa(ghRepo.ID),
		Name:          ghRepo.Name,
		Slug:          ghRepo.FullName,
		Description:   ghRepo.Description,
		DefaultBranch: ghRepo.DefaultBranch,
		WebURL:        ghRepo.HTMLURL,
		CloneURL:      ghRepo.CloneURL,
		SSHCloneURL:   ghRepo.SSHURL,
		Visibility:    ghRepo.Visibility,
		CreatedAt:     ghRepo.CreatedAt,
		UpdatedAt:     ghRepo.UpdatedAt,
	}, nil
}

// ListProjects returns user's repositories
func (p *GitHubProvider) ListProjects(ctx context.Context, page, perPage int) ([]*Project, error) {
	path := fmt.Sprintf("/user/repos?page=%d&per_page=%d&sort=updated", page, perPage)
	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghRepos []struct {
		ID            int       `json:"id"`
		Name          string    `json:"name"`
		FullName      string    `json:"full_name"`
		Description   string    `json:"description"`
		DefaultBranch string    `json:"default_branch"`
		HTMLURL       string    `json:"html_url"`
		CloneURL      string    `json:"clone_url"`
		SSHURL        string    `json:"ssh_url"`
		Visibility    string    `json:"visibility"`
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"updated_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghRepos); err != nil {
		return nil, err
	}

	projects := make([]*Project, len(ghRepos))
	for i, ghr := range ghRepos {
		projects[i] = &Project{
			ID:            strconv.Itoa(ghr.ID),
			Name:          ghr.Name,
			Slug:          ghr.FullName,
			Description:   ghr.Description,
			DefaultBranch: ghr.DefaultBranch,
			WebURL:        ghr.HTMLURL,
			CloneURL:      ghr.CloneURL,
			SSHCloneURL:   ghr.SSHURL,
			Visibility:    ghr.Visibility,
			CreatedAt:     ghr.CreatedAt,
			UpdatedAt:     ghr.UpdatedAt,
		}
	}

	return projects, nil
}

// SearchProjects searches for repositories
func (p *GitHubProvider) SearchProjects(ctx context.Context, query string, page, perPage int) ([]*Project, error) {
	path := fmt.Sprintf("/search/repositories?q=%s&page=%d&per_page=%d", url.QueryEscape(query), page, perPage)
	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID            int       `json:"id"`
			Name          string    `json:"name"`
			FullName      string    `json:"full_name"`
			Description   string    `json:"description"`
			DefaultBranch string    `json:"default_branch"`
			HTMLURL       string    `json:"html_url"`
			CloneURL      string    `json:"clone_url"`
			SSHURL        string    `json:"ssh_url"`
			Visibility    string    `json:"visibility"`
			CreatedAt     time.Time `json:"created_at"`
			UpdatedAt     time.Time `json:"updated_at"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	projects := make([]*Project, len(result.Items))
	for i, ghr := range result.Items {
		projects[i] = &Project{
			ID:            strconv.Itoa(ghr.ID),
			Name:          ghr.Name,
			Slug:          ghr.FullName,
			Description:   ghr.Description,
			DefaultBranch: ghr.DefaultBranch,
			WebURL:        ghr.HTMLURL,
			CloneURL:      ghr.CloneURL,
			SSHCloneURL:   ghr.SSHURL,
			Visibility:    ghr.Visibility,
			CreatedAt:     ghr.CreatedAt,
			UpdatedAt:     ghr.UpdatedAt,
		}
	}

	return projects, nil
}
