package git

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// RegisterWebhook registers a webhook
func (p *GiteeProvider) RegisterWebhook(ctx context.Context, projectID string, config *WebhookConfig) (string, error) {
	pushEvents := false
	prEvents := false
	for _, event := range config.Events {
		switch event {
		case "push":
			pushEvents = true
		case "merge_request":
			prEvents = true
		}
	}

	body := fmt.Sprintf(`{"url":"%s","password":"%s","push_events":%t,"pull_request_events":%t}`,
		config.URL, config.Secret, pushEvents, prEvents)

	resp, err := p.doRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/hooks", projectID), strings.NewReader(body))
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
func (p *GiteeProvider) DeleteWebhook(ctx context.Context, projectID, webhookID string) error {
	resp, err := p.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/repos/%s/hooks/%s", projectID, webhookID), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
