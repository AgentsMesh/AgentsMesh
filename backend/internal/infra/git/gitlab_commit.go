package git

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// GetCommit returns a specific commit
func (p *GitLabProvider) GetCommit(ctx context.Context, projectID, sha string) (*Commit, error) {
	encodedID := url.PathEscape(projectID)
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/projects/%s/repository/commits/%s", encodedID, sha), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glCommit struct {
		ID          string    `json:"id"`
		Message     string    `json:"message"`
		AuthorName  string    `json:"author_name"`
		AuthorEmail string    `json:"author_email"`
		CreatedAt   time.Time `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glCommit); err != nil {
		return nil, err
	}

	return &Commit{
		SHA:         glCommit.ID,
		Message:     glCommit.Message,
		Author:      glCommit.AuthorName,
		AuthorEmail: glCommit.AuthorEmail,
		CreatedAt:   glCommit.CreatedAt,
	}, nil
}

// ListCommits returns commits for a branch
func (p *GitLabProvider) ListCommits(ctx context.Context, projectID, branch string, page, perPage int) ([]*Commit, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/repository/commits?ref_name=%s&page=%d&per_page=%d",
		encodedID, url.QueryEscape(branch), page, perPage)

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glCommits []struct {
		ID          string    `json:"id"`
		Message     string    `json:"message"`
		AuthorName  string    `json:"author_name"`
		AuthorEmail string    `json:"author_email"`
		CreatedAt   time.Time `json:"created_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glCommits); err != nil {
		return nil, err
	}

	commits := make([]*Commit, len(glCommits))
	for i, glc := range glCommits {
		commits[i] = &Commit{
			SHA:         glc.ID,
			Message:     glc.Message,
			Author:      glc.AuthorName,
			AuthorEmail: glc.AuthorEmail,
			CreatedAt:   glc.CreatedAt,
		}
	}

	return commits, nil
}
