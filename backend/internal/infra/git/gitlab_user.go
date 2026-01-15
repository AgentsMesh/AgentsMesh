package git

import (
	"context"
	"encoding/json"
	"strconv"
)

// GetCurrentUser returns the authenticated user
func (p *GitLabProvider) GetCurrentUser(ctx context.Context) (*User, error) {
	resp, err := p.doRequest(ctx, "GET", "/user", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var glUser struct {
		ID        int    `json:"id"`
		Username  string `json:"username"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glUser); err != nil {
		return nil, err
	}

	return &User{
		ID:        strconv.Itoa(glUser.ID),
		Username:  glUser.Username,
		Name:      glUser.Name,
		Email:     glUser.Email,
		AvatarURL: glUser.AvatarURL,
	}, nil
}
