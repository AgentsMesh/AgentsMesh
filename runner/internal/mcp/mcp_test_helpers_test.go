package mcp

import "github.com/anthropics/agentsmesh/runner/internal/testutil"

// testDummyCmd returns a platform-appropriate command path for struct-level tests
// that do NOT actually start a process. The returned value is used as the Command
// field in Config to create Server instances for unit testing.
func testDummyCmd() string {
	cmd, _ := testutil.EchoCommand("")
	return cmd
}

// testCatCmd returns a platform-appropriate cat-equivalent command.
func testCatCmd() string {
	cmd, _ := testutil.CatCommand()
	return cmd
}
