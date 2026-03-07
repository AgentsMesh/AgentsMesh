package autopilot

import (
	"encoding/json"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/textutil"
)

// ExtractResultFromJSON extracts the "result" field from Claude Code JSON output.
func ExtractResultFromJSON(output string) string {
	var jsonResult struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(output), &jsonResult); err == nil && jsonResult.Result != "" {
		return jsonResult.Result
	}
	return ""
}

// FindDecisionMarker searches for a decision marker at the start of a line.
// Returns the DecisionType if found, or empty string if not found.
// Markers must appear at the beginning of a line (after optional whitespace).
func FindDecisionMarker(output string) DecisionType {
	lines := textutil.SplitLines(output)

	// Search from the end of the output, as the decision is typically at the end
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		upperLine := strings.ToUpper(line)

		// Check if line starts with a decision marker
		if strings.HasPrefix(upperLine, "TASK_COMPLETED") {
			return DecisionCompleted
		}
		if strings.HasPrefix(upperLine, "NEED_HUMAN_HELP") {
			return DecisionNeedHumanHelp
		}
		if strings.HasPrefix(upperLine, "GIVE_UP") {
			return DecisionGiveUp
		}
		if strings.HasPrefix(upperLine, "CONTINUE") {
			return DecisionContinue
		}
	}

	return "" // No marker found, will use default (Continue)
}

// ExtractSummary extracts a brief summary from the output.
// It looks for content after decision markers that appear at line start.
func ExtractSummary(output string) string {
	lines := textutil.SplitLines(output)
	markers := []string{"TASK_COMPLETED", "CONTINUE", "NEED_HUMAN_HELP", "GIVE_UP"}

	// Find the line with a decision marker at start
	markerLineIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		trimmedLine := strings.TrimSpace(lines[i])
		upperLine := strings.ToUpper(trimmedLine)
		for _, marker := range markers {
			if strings.HasPrefix(upperLine, marker) {
				markerLineIdx = i
				break
			}
		}
		if markerLineIdx >= 0 {
			break
		}
	}

	// Extract summary from lines after the marker
	if markerLineIdx >= 0 {
		var summaryLines []string
		for i := markerLineIdx + 1; i < len(lines) && i <= markerLineIdx+3; i++ {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed != "" && !strings.HasPrefix(trimmed, "{") {
				summaryLines = append(summaryLines, trimmed)
			}
		}
		if len(summaryLines) > 0 {
			summary := strings.Join(summaryLines, " ")
			if len(summary) > 200 {
				summary = summary[:200] + "..."
			}
			return summary
		}
	}

	// Fallback: Take last few non-empty lines as summary
	var summaryLines []string
	for i := len(lines) - 1; i >= 0 && len(summaryLines) < 3; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" && !strings.HasPrefix(trimmed, "{") {
			summaryLines = append([]string{trimmed}, summaryLines...)
		}
	}

	summary := strings.Join(summaryLines, " ")
	if len(summary) > 200 {
		summary = summary[:200] + "..."
	}
	return summary
}

// ExtractJSONBlock tries to find and parse a JSON block in the output.
func ExtractJSONBlock(output string) map[string]interface{} {
	// Find JSON block between { and }
	start := strings.Index(output, "{")
	if start == -1 {
		return nil
	}

	// Find matching closing brace
	depth := 0
	end := -1
	for i := start; i < len(output); i++ {
		switch output[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
		if end > 0 {
			break
		}
	}

	if end == -1 {
		return nil
	}

	jsonStr := output[start:end]
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil
	}

	return data
}

// ExtractSessionID extracts session_id from Claude's JSON output.
func ExtractSessionID(output string) string {
	// Try to parse as JSON to get session_id
	var result struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(output), &result); err == nil && result.SessionID != "" {
		return result.SessionID
	}

	// Try to find session_id in the output text
	if idx := strings.Index(output, `"session_id"`); idx != -1 {
		// Try to extract value after session_id
		remaining := output[idx:]
		if colonIdx := strings.Index(remaining, ":"); colonIdx != -1 {
			afterColon := strings.TrimSpace(remaining[colonIdx+1:])
			if len(afterColon) > 0 && afterColon[0] == '"' {
				endQuote := strings.Index(afterColon[1:], `"`)
				if endQuote != -1 {
					return afterColon[1 : endQuote+1]
				}
			}
		}
	}

	return ""
}

// mapDecisionType maps string type to DecisionType.
func mapDecisionType(typeStr string) DecisionType {
	switch strings.ToLower(typeStr) {
	case "completed", "task_completed":
		return DecisionCompleted
	case "continue":
		return DecisionContinue
	case "need_help", "need_human_help":
		return DecisionNeedHumanHelp
	case "give_up", "giveup":
		return DecisionGiveUp
	default:
		return DecisionContinue
	}
}

// extractJSONString extracts the first complete JSON object from content.
func extractJSONString(content string) string {
	start := strings.Index(content, "{")
	if start == -1 {
		return ""
	}

	depth := 0
	end := -1
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
		if end > 0 {
			break
		}
	}

	if end == -1 {
		return ""
	}

	return content[start:end]
}

// truncateSummary truncates a string to maxLen characters.
func truncateSummary(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
