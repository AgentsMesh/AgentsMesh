// Package envfilter provides environment variable filtering for child processes.
// Runner-internal variables (gRPC debug, internal config) are removed to prevent
// accidental leakage to PTY terminals and MCP server processes.
package envfilter

import "strings"

// internalPrefixes are Runner-internal env var prefixes that should
// not leak to child processes (MCP servers, PTY terminals).
// This is a conservative denylist — user env vars, cloud credentials,
// and tool configs are preserved to avoid breaking agent functionality.
var internalPrefixes = []string{
	"AGENTSMESH_", // Runner config internals
	"GRPC_GO_",    // gRPC library debug vars
}

// FilterEnv returns a copy of env with Runner-internal variables removed.
func FilterEnv(env []string) []string {
	result := make([]string, 0, len(env))
	for _, e := range env {
		if shouldFilter(e) {
			continue
		}
		result = append(result, e)
	}
	return result
}

// shouldFilter returns true if the env entry matches any internal prefix.
func shouldFilter(entry string) bool {
	for _, prefix := range internalPrefixes {
		if strings.HasPrefix(entry, prefix) {
			return true
		}
	}
	return false
}
