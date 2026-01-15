package git

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

// GetFileContent returns file content
func (p *GitLabProvider) GetFileContent(ctx context.Context, projectID, filePath, ref string) ([]byte, error) {
	encodedID := url.PathEscape(projectID)
	encodedPath := url.PathEscape(filePath)
	path := fmt.Sprintf("/projects/%s/repository/files/%s/raw?ref=%s", encodedID, encodedPath, url.QueryEscape(ref))

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
