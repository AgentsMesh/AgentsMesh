package git

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
)

// GetFileContent returns file content
func (p *GiteeProvider) GetFileContent(ctx context.Context, projectID, filePath, ref string) ([]byte, error) {
	path := fmt.Sprintf("/repos/%s/contents/%s?ref=%s", projectID, filePath, url.QueryEscape(ref))

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Encoding == "base64" {
		return base64.StdEncoding.DecodeString(result.Content)
	}

	return []byte(result.Content), nil
}
