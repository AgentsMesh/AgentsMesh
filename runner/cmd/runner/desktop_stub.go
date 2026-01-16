//go:build !desktop

package main

import (
	"fmt"
	"os"
)

// runDesktop is a stub when built without desktop tag.
// CLI builds don't include desktop/tray support.
func runDesktop(args []string) {
	fmt.Println("Desktop mode is not available in this build.")
	fmt.Println("")
	fmt.Println("This is the CLI version of AgentsMesh Runner.")
	fmt.Println("Desktop mode with system tray requires the Desktop build.")
	fmt.Println("")
	fmt.Println("For CLI usage:")
	fmt.Println("  runner run         - Start the runner")
	fmt.Println("  runner webconsole  - Open web console in browser")
	os.Exit(1)
}
