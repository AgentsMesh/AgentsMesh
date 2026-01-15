package git

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ListBranches returns branches for a repository
func (p *GitHubProvider) ListBranches(ctx context.Context, projectID string) ([]*Branch, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repos/%s/branches", projectID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghBranches []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
		Protected bool `json:"protected"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghBranches); err != nil {
		return nil, err
	}

	project, _ := p.GetProject(ctx, projectID)
	defaultBranch := ""
	if project != nil {
		defaultBranch = project.DefaultBranch
	}

	branches := make([]*Branch, len(ghBranches))
	for i, ghb := range ghBranches {
		branches[i] = &Branch{
			Name:      ghb.Name,
			CommitSHA: ghb.Commit.SHA,
			Protected: ghb.Protected,
			Default:   ghb.Name == defaultBranch,
		}
	}

	return branches, nil
}

// GetBranch returns a specific branch
func (p *GitHubProvider) GetBranch(ctx context.Context, projectID, branchName string) (*Branch, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repos/%s/branches/%s", projectID, url.PathEscape(branchName)), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghBranch struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
		Protected bool `json:"protected"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghBranch); err != nil {
		return nil, err
	}

	project, _ := p.GetProject(ctx, projectID)
	isDefault := project != nil && ghBranch.Name == project.DefaultBranch

	return &Branch{
		Name:      ghBranch.Name,
		CommitSHA: ghBranch.Commit.SHA,
		Protected: ghBranch.Protected,
		Default:   isDefault,
	}, nil
}

// CreateBranch creates a new branch
func (p *GitHubProvider) CreateBranch(ctx context.Context, projectID, branchName, ref string) (*Branch, error) {
	refResp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repos/%s/git/refs/heads/%s", projectID, ref), nil)
	if err != nil {
		return nil, err
	}

	var refData struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	json.NewDecoder(refResp.Body).Decode(&refData)
	refResp.Body.Close()

	body := fmt.Sprintf(`{"ref":"refs/heads/%s","sha":"%s"}`, branchName, refData.Object.SHA)
	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/repos/%s/git/refs", projectID), strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return &Branch{
		Name:      branchName,
		CommitSHA: refData.Object.SHA,
		Protected: false,
		Default:   false,
	}, nil
}

// DeleteBranch deletes a branch
func (p *GitHubProvider) DeleteBranch(ctx context.Context, projectID, branchName string) error {
	resp, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/repos/%s/git/refs/heads/%s", projectID, url.PathEscape(branchName)), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
