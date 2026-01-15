package git

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// GetCommit returns a specific commit
func (p *GitHubProvider) GetCommit(ctx context.Context, projectID, sha string) (*Commit, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/repos/%s/commits/%s", projectID, sha), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghCommit struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghCommit); err != nil {
		return nil, err
	}

	return &Commit{
		SHA:         ghCommit.SHA,
		Message:     ghCommit.Commit.Message,
		Author:      ghCommit.Commit.Author.Name,
		AuthorEmail: ghCommit.Commit.Author.Email,
		CreatedAt:   ghCommit.Commit.Author.Date,
	}, nil
}

// ListCommits returns commits for a branch
func (p *GitHubProvider) ListCommits(ctx context.Context, projectID, branch string, page, perPage int) ([]*Commit, error) {
	path := fmt.Sprintf("/repos/%s/commits?sha=%s&page=%d&per_page=%d", projectID, url.QueryEscape(branch), page, perPage)

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghCommits []struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			} `json:"author"`
		} `json:"commit"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghCommits); err != nil {
		return nil, err
	}

	commits := make([]*Commit, len(ghCommits))
	for i, ghc := range ghCommits {
		commits[i] = &Commit{
			SHA:         ghc.SHA,
			Message:     ghc.Commit.Message,
			Author:      ghc.Commit.Author.Name,
			AuthorEmail: ghc.Commit.Author.Email,
			CreatedAt:   ghc.Commit.Author.Date,
		}
	}

	return commits, nil
}
