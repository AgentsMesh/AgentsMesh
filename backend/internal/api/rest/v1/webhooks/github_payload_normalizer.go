package webhooks

func normalizeGitHubMRPayload(payload map[string]interface{}) {
	if _, exists := payload["object_attributes"]; exists {
		return
	}

	pr, ok := payload["pull_request"].(map[string]interface{})
	if !ok {
		return
	}

	objAttrs := map[string]interface{}{}

	if number, ok := pr["number"].(float64); ok {
		objAttrs["iid"] = number
	}
	if htmlURL, ok := pr["html_url"].(string); ok {
		objAttrs["url"] = htmlURL
	}
	if title, ok := pr["title"].(string); ok {
		objAttrs["title"] = title
	}
	if head, ok := pr["head"].(map[string]interface{}); ok {
		if ref, ok := head["ref"].(string); ok {
			objAttrs["source_branch"] = ref
		}
	}
	if base, ok := pr["base"].(map[string]interface{}); ok {
		if ref, ok := base["ref"].(string); ok {
			objAttrs["target_branch"] = ref
		}
	}

	merged, _ := pr["merged"].(bool)
	state, _ := pr["state"].(string)
	objAttrs["state"] = githubMRState(state, merged)

	if action, ok := payload["action"].(string); ok {
		objAttrs["action"] = githubMRAction(action, merged)
	}

	if merged {
		if sha, ok := pr["merge_commit_sha"].(string); ok && sha != "" {
			objAttrs["merge_commit_sha"] = sha
		}
		if mergedAt, ok := pr["merged_at"].(string); ok && mergedAt != "" {
			objAttrs["merged_at"] = mergedAt
		}
	}

	payload["object_attributes"] = objAttrs
}

func githubMRState(state string, merged bool) string {
	switch {
	case merged:
		return "merged"
	case state == "closed":
		return "closed"
	default:
		return "opened"
	}
}

func githubMRAction(action string, merged bool) string {
	switch action {
	case "opened":
		return "open"
	case "reopened":
		return "reopen"
	case "closed":
		if merged {
			return "merge"
		}
		return "close"
	case "synchronize", "edited":
		return "update"
	default:
		return action
	}
}
