package git

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// GetProject returns project details
func (p *GitLabProvider) GetProject(ctx context.Context, projectID string) (*Project, error) {
	encodedID := url.PathEscape(projectID)
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s", encodedID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glProject struct {
		ID                int       `json:"id"`
		Name              string    `json:"name"`
		PathWithNamespace string    `json:"path_with_namespace"`
		Description       string    `json:"description"`
		DefaultBranch     string    `json:"default_branch"`
		WebURL            string    `json:"web_url"`
		HTTPURLToRepo     string    `json:"http_url_to_repo"`
		SSHURLToRepo      string    `json:"ssh_url_to_repo"`
		Visibility        string    `json:"visibility"`
		CreatedAt         time.Time `json:"created_at"`
		LastActivityAt    time.Time `json:"last_activity_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glProject); err != nil {
		return nil, err
	}

	return &Project{
		ID:            strconv.Itoa(glProject.ID),
		Name:          glProject.Name,
		Slug:      glProject.PathWithNamespace,
		Description:   glProject.Description,
		DefaultBranch: glProject.DefaultBranch,
		WebURL:        glProject.WebURL,
		HttpCloneURL:  glProject.HTTPURLToRepo,
		SSHCloneURL:   glProject.SSHURLToRepo,
		Visibility:    glProject.Visibility,
		CreatedAt:     glProject.CreatedAt,
		UpdatedAt:     glProject.LastActivityAt,
	}, nil
}

// ListProjects returns user's projects
func (p *GitLabProvider) ListProjects(ctx context.Context, page, perPage int) ([]*Project, error) {
	path := fmt.Sprintf("/projects?membership=true&page=%d&per_page=%d", page, perPage)
	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glProjects []struct {
		ID                int       `json:"id"`
		Name              string    `json:"name"`
		PathWithNamespace string    `json:"path_with_namespace"`
		Description       string    `json:"description"`
		DefaultBranch     string    `json:"default_branch"`
		WebURL            string    `json:"web_url"`
		HTTPURLToRepo     string    `json:"http_url_to_repo"`
		SSHURLToRepo      string    `json:"ssh_url_to_repo"`
		Visibility        string    `json:"visibility"`
		CreatedAt         time.Time `json:"created_at"`
		LastActivityAt    time.Time `json:"last_activity_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glProjects); err != nil {
		return nil, err
	}

	projects := make([]*Project, len(glProjects))
	for i, glp := range glProjects {
		projects[i] = &Project{
			ID:            strconv.Itoa(glp.ID),
			Name:          glp.Name,
			Slug:      glp.PathWithNamespace,
			Description:   glp.Description,
			DefaultBranch: glp.DefaultBranch,
			WebURL:        glp.WebURL,
			HttpCloneURL:  glp.HTTPURLToRepo,
			SSHCloneURL:   glp.SSHURLToRepo,
			Visibility:    glp.Visibility,
			CreatedAt:     glp.CreatedAt,
			UpdatedAt:     glp.LastActivityAt,
		}
	}

	return projects, nil
}

// SearchProjects searches for projects
func (p *GitLabProvider) SearchProjects(ctx context.Context, query string, page, perPage int) ([]*Project, error) {
	path := fmt.Sprintf("/projects?search=%s&page=%d&per_page=%d", url.QueryEscape(query), page, perPage)
	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glProjects []struct {
		ID                int       `json:"id"`
		Name              string    `json:"name"`
		PathWithNamespace string    `json:"path_with_namespace"`
		Description       string    `json:"description"`
		DefaultBranch     string    `json:"default_branch"`
		WebURL            string    `json:"web_url"`
		HTTPURLToRepo     string    `json:"http_url_to_repo"`
		SSHURLToRepo      string    `json:"ssh_url_to_repo"`
		Visibility        string    `json:"visibility"`
		CreatedAt         time.Time `json:"created_at"`
		LastActivityAt    time.Time `json:"last_activity_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glProjects); err != nil {
		return nil, err
	}

	projects := make([]*Project, len(glProjects))
	for i, glp := range glProjects {
		projects[i] = &Project{
			ID:            strconv.Itoa(glp.ID),
			Name:          glp.Name,
			Slug:      glp.PathWithNamespace,
			Description:   glp.Description,
			DefaultBranch: glp.DefaultBranch,
			WebURL:        glp.WebURL,
			HttpCloneURL:  glp.HTTPURLToRepo,
			SSHCloneURL:   glp.SSHURLToRepo,
			Visibility:    glp.Visibility,
			CreatedAt:     glp.CreatedAt,
			UpdatedAt:     glp.LastActivityAt,
		}
	}

	return projects, nil
}
