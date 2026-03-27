package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// fetchGitHubPrimaryEmail fetches the primary verified email from GitHub
// /user/emails API when the public profile email is empty.
func fetchGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub emails API returned status %d", resp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	// Prefer primary + verified email
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	// Fall back to any verified email
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}
	return "", nil
}

// parseGitHubUserInfo parses GitHub user info
func parseGitHubUserInfo(body []byte) (*UserInfo, error) {
	var data struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return &UserInfo{
		ID:        fmt.Sprintf("%d", data.ID),
		Username:  data.Login,
		Email:     data.Email,
		Name:      data.Name,
		AvatarURL: data.AvatarURL,
	}, nil
}

// parseGoogleUserInfo parses Google user info
func parseGoogleUserInfo(body []byte) (*UserInfo, error) {
	var data struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		VerifiedEmail bool   `json:"verified_email"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	if !data.VerifiedEmail {
		return nil, fmt.Errorf("google email %s is not verified", data.Email)
	}
	// Generate username from email
	username := strings.Split(data.Email, "@")[0]
	return &UserInfo{
		ID:        data.ID,
		Username:  username,
		Email:     data.Email,
		Name:      data.Name,
		AvatarURL: data.Picture,
	}, nil
}

// parseGitLabUserInfo parses GitLab user info
func parseGitLabUserInfo(body []byte) (*UserInfo, error) {
	var data struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return &UserInfo{
		ID:        fmt.Sprintf("%d", data.ID),
		Username:  data.Username,
		Email:     data.Email,
		Name:      data.Name,
		AvatarURL: data.AvatarURL,
	}, nil
}

// parseGiteeUserInfo parses Gitee user info
func parseGiteeUserInfo(body []byte) (*UserInfo, error) {
	var data struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return &UserInfo{
		ID:        fmt.Sprintf("%d", data.ID),
		Username:  data.Login,
		Email:     data.Email,
		Name:      data.Name,
		AvatarURL: data.AvatarURL,
	}, nil
}
