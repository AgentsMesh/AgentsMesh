package git

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func (p *GitLabProvider) RegisterWebhook(ctx context.Context, projectID string, config *WebhookConfig) (string, error) {
	encodedID := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/hooks", encodedID)

	pushEvents := false
	mrEvents := false
	pipelineEvents := false
	for _, event := range config.Events {
		switch event {
		case "push":
			pushEvents = true
		case "merge_request":
			mrEvents = true
		case "pipeline":
			pipelineEvents = true
		}
	}

	body := fmt.Sprintf(`{"url":"%s","token":"%s","push_events":%t,"merge_requests_events":%t,"pipeline_events":%t}`,
		config.URL, config.Secret, pushEvents, mrEvents, pipelineEvents)

	resp, err := p.doRequest(ctx, "POST", path, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ID int `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return strconv.Itoa(result.ID), nil
}

func (p *GitLabProvider) DeleteWebhook(ctx context.Context, projectID, webhookID string) error {
	encodedID := url.PathEscape(projectID)
	resp, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/projects/%s/hooks/%s", encodedID, webhookID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
