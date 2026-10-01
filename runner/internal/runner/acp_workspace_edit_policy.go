package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

// acpWorkspaceEditPolicy answers ACP file-edit permission requests that can only
// touch the pod workspace. ACP runs Claude Code non-interactively (`-p`), so a
// request needing a human only resolves while a browser is attached to the relay;
// with the CLI default permission mode every Edit waited forever and surfaced as
// denied in headless pods, while PTY mode kept working (issue #241).
type acpWorkspaceEditPolicy struct {
	enabled   bool
	editable  bool
	mode      string
	workspace string
}

var acpEditTools = map[string]bool{
	"Edit": true, "Write": true, "MultiEdit": true, "NotebookEdit": true,
}

var acpEditPathKeys = []string{"file_path", "filePath", "path", "notebook_path"}

func newACPWorkspaceEditPolicy(workspace string, args []string) acpWorkspaceEditPolicy {
	p := acpWorkspaceEditPolicy{workspace: workspace}
	if !isClaudeACPSession(args) {
		// codex/gemini/opencode advertise no mode we can interpret.
		return p
	}
	p.enabled = true
	p.mode = parseClaudeInitialConfig(args).PermissionMode
	// "" = no --permission-mode flag, i.e. Claude's interactive default prompt
	// policy, which a non-interactive ACP session cannot answer. acceptEdits is
	// documented as auto-approving edits. "default" (what ACP rewrites "plan"
	// to), "plan", "dontAsk" and "bypassPermissions" stay untouched.
	p.editable = p.mode == "" || p.mode == "acceptEdits"
	return p
}

func isClaudeACPSession(args []string) bool {
	printMode, streamJSON := false, false
	for i, a := range args {
		switch a {
		case "-p", "--print":
			printMode = true
		case "--input-format":
			streamJSON = streamJSON || (i+1 < len(args) && args[i+1] == "stream-json")
		case "--input-format=stream-json":
			streamJSON = true
		}
	}
	return printMode && streamJSON
}

func (p acpWorkspaceEditPolicy) shouldAutoApprove(req acp.PermissionRequest) bool {
	if !p.enabled || !p.editable || p.workspace == "" || !acpEditTools[req.ToolName] {
		return false
	}
	paths := acpEditPaths(req.ArgumentsJSON)
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		if !pathInsideWorkspace(p.workspace, path) {
			return false
		}
	}
	return true
}

func acpEditPaths(raw string) []string {
	var args map[string]any
	if raw == "" || json.Unmarshal([]byte(raw), &args) != nil {
		return nil
	}
	var paths []string
	for _, key := range acpEditPathKeys {
		if v, ok := args[key].(string); ok && v != "" {
			paths = append(paths, v)
		}
	}
	return paths
}

func pathInsideWorkspace(root, target string) bool {
	if !filepath.IsAbs(target) {
		return false
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(rootAbs), filepath.Clean(target))
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
