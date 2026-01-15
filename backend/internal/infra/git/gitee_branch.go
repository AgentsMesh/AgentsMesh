package git

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ListBranches returns branches for a repository
func (p *GiteeProvider) ListBranches(ctx context.Context, projectID string) ([]*Branch, error) {
	resp, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/branches", projectID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gtBranches []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
		Protected bool `json:"protected"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gtBranches); err != nil {
		return nil, err
	}

	project, _ := p.GetProject(ctx, projectID)
	defaultBranch := ""
	if project != nil {
		defaultBranch = project.DefaultBranch
	}

	branches := make([]*Branch, len(gtBranches))
	for i, gtb := range gtBranches {
		branches[i] = &Branch{
			Name:      gtb.Name,
			CommitSHA: gtb.Commit.SHA,
			Protected: gtb.Protected,
			Default:   gtb.Name == defaultBranch,
		}
	}

	return branches, nil
}

// GetBranch returns a specific branch
func (p *GiteeProvider) GetBranch(ctx context.Context, projectID, branchName string) (*Branch, error) {
	resp, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/branches/%s", projectID, url.PathEscape(branchName)), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gtBranch struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
		Protected bool `json:"protected"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gtBranch); err != nil {
		return nil, err
	}

	project, _ := p.GetProject(ctx, projectID)
	isDefault := project != nil && gtBranch.Name == project.DefaultBranch

	return &Branch{
		Name:      gtBranch.Name,
		CommitSHA: gtBranch.Commit.SHA,
		Protected: gtBranch.Protected,
		Default:   isDefault,
	}, nil
}

// CreateBranch creates a new branch
func (p *GiteeProvider) CreateBranch(ctx context.Context, projectID, branchName, ref string) (*Branch, error) {
	body := fmt.Sprintf(`{"refs":"%s","branch_name":"%s"}`, ref, branchName)
	resp, err := p.doRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/branches", projectID), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gtBranch struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gtBranch); err != nil {
		return nil, err
	}

	return &Branch{
		Name:      gtBranch.Name,
		CommitSHA: gtBranch.Commit.SHA,
		Protected: false,
		Default:   false,
	}, nil
}

// DeleteBranch deletes a branch
func (p *GiteeProvider) DeleteBranch(ctx context.Context, projectID, branchName string) error {
	resp, err := p.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/repos/%s/branches/%s", projectID, url.PathEscape(branchName)), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
