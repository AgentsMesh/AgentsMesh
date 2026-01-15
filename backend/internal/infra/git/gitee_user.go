package git

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

// GetCurrentUser returns the authenticated user
func (p *GiteeProvider) GetCurrentUser(ctx context.Context) (*User, error) {
	resp, err := p.doRequest(ctx, http.MethodGet, "/user", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var gtUser struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gtUser); err != nil {
		return nil, err
	}

	return &User{
		ID:        strconv.Itoa(gtUser.ID),
		Username:  gtUser.Login,
		Name:      gtUser.Name,
		Email:     gtUser.Email,
		AvatarURL: gtUser.AvatarURL,
	}, nil
}
