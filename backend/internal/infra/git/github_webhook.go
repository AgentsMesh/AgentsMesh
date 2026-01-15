package git

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// RegisterWebhook registers a webhook
func (p *GitHubProvider) RegisterWebhook(ctx context.Context, projectID string, config *WebhookConfig) (string, error) {
	events := make([]string, 0, len(config.Events))
	for _, event := range config.Events {
		switch event {
		case "push":
			events = append(events, "push")
		case "merge_request":
			events = append(events, "pull_request")
		}
	}

	eventsJSON, _ := json.Marshal(events)
	body := fmt.Sprintf(`{"name":"web","active":true,"events":%s,"config":{"url":"%s","content_type":"json","secret":"%s"}}`,
		string(eventsJSON), config.URL, config.Secret)

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/repos/%s/hooks", projectID), strings.NewReader(body))
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

// DeleteWebhook deletes a webhook
func (p *GitHubProvider) DeleteWebhook(ctx context.Context, projectID, webhookID string) error {
	resp, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/repos/%s/hooks/%s", projectID, webhookID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
