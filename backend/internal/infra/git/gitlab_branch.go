package git

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// ListBranches returns branches for a project
func (p *GitLabProvider) ListBranches(ctx context.Context, projectID string) ([]*Branch, error) {
	encodedID := url.PathEscape(projectID)
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s/repository/branches", encodedID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glBranches []struct {
		Name   string `json:"name"`
		Commit struct {
			ID string `json:"id"`
		} `json:"commit"`
		Protected bool `json:"protected"`
		Default   bool `json:"default"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glBranches); err != nil {
		return nil, err
	}

	branches := make([]*Branch, len(glBranches))
	for i, glb := range glBranches {
		branches[i] = &Branch{
			Name:      glb.Name,
			CommitSHA: glb.Commit.ID,
			Protected: glb.Protected,
			Default:   glb.Default,
		}
	}

	return branches, nil
}

// GetBranch returns a specific branch
func (p *GitLabProvider) GetBranch(ctx context.Context, projectID, branchName string) (*Branch, error) {
	encodedID := url.PathEscape(projectID)
	encodedBranch := url.PathEscape(branchName)
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s/repository/branches/%s", encodedID, encodedBranch), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glBranch struct {
		Name   string `json:"name"`
		Commit struct {
			ID string `json:"id"`
		} `json:"commit"`
		Protected bool `json:"protected"`
		Default   bool `json:"default"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glBranch); err != nil {
		return nil, err
	}

	return &Branch{
		Name:      glBranch.Name,
		CommitSHA: glBranch.Commit.ID,
		Protected: glBranch.Protected,
		Default:   glBranch.Default,
	}, nil
}

// CreateBranch creates a new branch
func (p *GitLabProvider) CreateBranch(ctx context.Context, projectID, branchName, ref string) (*Branch, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/repository/branches?branch=%s&ref=%s",
		encodedID, url.QueryEscape(branchName), url.QueryEscape(ref))

	resp, err := p.doRequest(ctx, "POST", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glBranch struct {
		Name   string `json:"name"`
		Commit struct {
			ID string `json:"id"`
		} `json:"commit"`
		Protected bool `json:"protected"`
		Default   bool `json:"default"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glBranch); err != nil {
		return nil, err
	}

	return &Branch{
		Name:      glBranch.Name,
		CommitSHA: glBranch.Commit.ID,
		Protected: glBranch.Protected,
		Default:   glBranch.Default,
	}, nil
}

// DeleteBranch deletes a branch
func (p *GitLabProvider) DeleteBranch(ctx context.Context, projectID, branchName string) error {
	encodedID := url.PathEscape(projectID)
	encodedBranch := url.PathEscape(branchName)
	resp, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/projects/%s/repository/branches/%s", encodedID, encodedBranch), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
