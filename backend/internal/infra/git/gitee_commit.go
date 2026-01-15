package git

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// GetCommit returns a specific commit
func (p *GiteeProvider) GetCommit(ctx context.Context, projectID, sha string) (*Commit, error) {
	resp, err := p.doRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/commits/%s", projectID, sha), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gtCommit struct {
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

	if err := json.NewDecoder(resp.Body).Decode(&gtCommit); err != nil {
		return nil, err
	}

	return &Commit{
		SHA:         gtCommit.SHA,
		Message:     gtCommit.Commit.Message,
		Author:      gtCommit.Commit.Author.Name,
		AuthorEmail: gtCommit.Commit.Author.Email,
		CreatedAt:   gtCommit.Commit.Author.Date,
	}, nil
}

// ListCommits returns commits for a branch
func (p *GiteeProvider) ListCommits(ctx context.Context, projectID, branch string, page, perPage int) ([]*Commit, error) {
	path := fmt.Sprintf("/repos/%s/commits?sha=%s&page=%d&per_page=%d", projectID, url.QueryEscape(branch), page, perPage)

	resp, err := p.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gtCommits []struct {
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

	if err := json.NewDecoder(resp.Body).Decode(&gtCommits); err != nil {
		return nil, err
	}

	commits := make([]*Commit, len(gtCommits))
	for i, gtc := range gtCommits {
		commits[i] = &Commit{
			SHA:         gtc.SHA,
			Message:     gtc.Commit.Message,
			Author:      gtc.Commit.Author.Name,
			AuthorEmail: gtc.Commit.Author.Email,
			CreatedAt:   gtc.Commit.Author.Date,
		}
	}

	return commits, nil
}
