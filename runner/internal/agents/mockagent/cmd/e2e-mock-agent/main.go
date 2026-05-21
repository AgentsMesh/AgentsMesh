// Command e2e-mock-agent is a programmable mock agent used for e2e tests of
// PTY-mode and ACP-mode pods. Its sole purpose is to give tests a deterministic
// agent process that emits a chosen scenario over stdout/stderr (PTY) or
// JSON-RPC 2.0 (ACP), so that runner+relay+web behavior can be validated
// without depending on real LLM CLIs.
//
// Selection:
//
//	--mode=pty|acp           (env: E2E_MOCK_MODE, default "pty")
//	--scenario=<name>        (env: E2E_MOCK_SCENARIO, default "echo")
//
// CLI flags take precedence over env vars. PATH-discoverable so the
// e2e-echo agentfile can use EXECUTABLE e2e-mock-agent directly.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/anthropics/agentsmesh/runner/internal/agents/mockagent"
)

func main() {
	mode := flag.String("mode", envDefault("E2E_MOCK_MODE", "pty"), "runtime mode: pty | acp")
	scenario := flag.String("scenario", envDefault("E2E_MOCK_SCENARIO", "echo"), "behavior scenario name")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	switch *mode {
	case "pty":
		os.Exit(mockagent.RunPTY(*scenario, logger))
	case "acp":
		os.Exit(mockagent.RunACP(*scenario, logger))
	default:
		fmt.Fprintf(os.Stderr, "e2e-mock-agent: unknown mode %q (want: pty|acp)\n", *mode)
		os.Exit(2)
	}
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
