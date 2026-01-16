package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// runWebConsole handles the "webconsole" subcommand.
// It opens the web console in the default browser.
func runWebConsole(args []string) {
	fs := flag.NewFlagSet("webconsole", flag.ExitOnError)
	port := fs.Int("port", DefaultConsolePort, "Web console port")

	fs.Usage = func() {
		fmt.Println(`Open the AgentsMesh Runner web console in browser.

Usage:
  runner webconsole [options]

Options:`)
		fs.PrintDefaults()
		fmt.Println(`
The web console provides a web-based interface to:
  - View runner status and connection state
  - Monitor active pods
  - View logs
  - Manage runner configuration

Note: The runner must be running ('runner run') for the web console to be accessible.`)
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	url := fmt.Sprintf("http://127.0.0.1:%d", *port)

	// Check if the console is accessible
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url + "/api/status")
	if err != nil {
		fmt.Println("Error: Web console is not accessible.")
		fmt.Println("")
		fmt.Println("Make sure the runner is running with 'runner run' first.")
		fmt.Printf("The web console should be available at %s\n", url)
		os.Exit(1)
	}
	resp.Body.Close()

	// Open browser
	fmt.Printf("Opening web console at %s\n", url)
	if err := openBrowser(url); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
		fmt.Printf("Please open %s manually in your browser.\n", url)
		os.Exit(1)
	}
}

// openBrowser opens the specified URL in the default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
