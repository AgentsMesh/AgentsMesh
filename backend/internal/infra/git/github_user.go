package git

import (
	"context"
	"encoding/json"
	"strconv"
)

// GetCurrentUser returns the authenticated user
func (p *GitHubProvider) GetCurrentUser(ctx context.Context) (*User, error) {
	resp, err := p.doRequest(ctx, "GET", "/user", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ghUser struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, err
	}

	return &User{
		ID:        strconv.Itoa(ghUser.ID),
		Username:  ghUser.Login,
		Name:      ghUser.Name,
		Email:     ghUser.Email,
		AvatarURL: ghUser.AvatarURL,
	}, nil
}
