package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// listResources retrieves available resources from the server
func (s *Server) listResources(ctx context.Context) error {
	resp, err := s.call(ctx, "resources/list", nil)
	if err != nil {
		return err
	}

	var result struct {
		Resources []Resource `json:"resources"`
	}

	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to parse resources list: %w", err)
	}

	s.mu.Lock()
	for _, res := range result.Resources {
		r := res
		s.resources[res.URI] = &r
	}
	s.mu.Unlock()

	return nil
}

// GetResources returns available resources
func (s *Server) GetResources() []*Resource {
	s.mu.Lock()
	defer s.mu.Unlock()

	resources := make([]*Resource, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, r)
	}
	return resources
}

// ReadResource reads an MCP resource
func (s *Server) ReadResource(ctx context.Context, uri string) ([]byte, string, error) {
	params := map[string]interface{}{
		"uri": uri,
	}

	resp, err := s.call(ctx, "resources/read", params)
	if err != nil {
		return nil, "", err
	}

	if resp.Error != nil {
		return nil, "", fmt.Errorf("resource read failed: %s", resp.Error.Message)
	}

	var result struct {
		Contents []struct {
			URI      string `json:"uri"`
			MimeType string `json:"mimeType,omitempty"`
			Text     string `json:"text,omitempty"`
			Blob     string `json:"blob,omitempty"`
		} `json:"contents"`
	}

	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, "", fmt.Errorf("failed to parse resource: %w", err)
	}

	if len(result.Contents) == 0 {
		return nil, "", fmt.Errorf("no content returned")
	}

	content := result.Contents[0]
	if content.Text != "" {
		return []byte(content.Text), content.MimeType, nil
	}

	// Handle blob (base64 encoded)
	// In a real implementation, you'd decode the base64
	return []byte(content.Blob), content.MimeType, nil
}
