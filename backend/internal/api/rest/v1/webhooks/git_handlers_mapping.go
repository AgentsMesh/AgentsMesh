package webhooks

import (
	"github.com/gin-gonic/gin"
)

func (r *WebhookRouter) extractObjectKind(payload map[string]interface{}, provider string, c *gin.Context) string {
	switch provider {
	case "gitlab":
		if kind, ok := payload["object_kind"].(string); ok {
			return kind
		}
	case "github":
		event := c.GetHeader("X-GitHub-Event")
		if event != "" {
			return r.mapGitHubEventToKind(event)
		}
	case "gitee":
		event := c.GetHeader("X-Gitee-Event")
		if event != "" {
			return r.mapGiteeEventToKind(event)
		}
		if hookName, ok := payload["hook_name"].(string); ok {
			return r.mapGiteeEventToKind(hookName)
		}
	}

	return ""
}

func (r *WebhookRouter) mapGitHubEventToKind(event string) string {
	mapping := map[string]string{
		"push":                "push",
		"pull_request":        "merge_request",
		"check_run":           "pipeline",
		"check_suite":         "pipeline",
		"workflow_run":        "pipeline",
		"status":              "pipeline",
		"issues":              "issue",
		"issue_comment":       "note",
		"pull_request_review": "note",
	}

	if kind, ok := mapping[event]; ok {
		return kind
	}
	return event
}

func (r *WebhookRouter) mapGiteeEventToKind(event string) string {
	mapping := map[string]string{
		"push_hooks":          "push",
		"Push Hook":           "push",
		"merge_request_hooks": "merge_request",
		"Merge Request Hook":  "merge_request",
		"issue_hooks":         "issue",
		"Issue Hook":          "issue",
		"note_hooks":          "note",
		"Note Hook":           "note",
	}

	if kind, ok := mapping[event]; ok {
		return kind
	}
	return event
}
