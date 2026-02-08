// Package main provides the entry point for the AgentsMesh Runner CLI.
package main

import (
	"fmt"
	"os"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "login", "register":
		runRegister(os.Args[2:])
	case "run", "start":
		runRunner(os.Args[2:])
	case "service":
		runService(os.Args[2:])
	case "desktop":
		runDesktop(os.Args[2:])
	case "webconsole", "console":
		runWebConsole(os.Args[2:])
	case "reactivate":
		runReactivate(os.Args[2:])
	case "update":
		runUpdate(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("AgentsMesh Runner %s (built %s)\n", version, buildTime)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`AgentsMesh Runner

Usage:
  runner <command> [options]

Commands:
  login       Login to AgentsMesh server (alias for register)
  register    Register this runner with the AgentsMesh server (gRPC/mTLS)
  run         Start the runner in CLI mode (requires prior registration)
  webconsole  Open the web console in browser
  service     Manage runner as a system service (install/start/stop)
  desktop     Start runner in desktop mode with system tray
  reactivate  Reactivate runner with expired certificate
  update      Check and install updates
  version     Show version information
  help        Show this help message

Login Examples:
  runner login
      Opens browser for authorization (uses https://agentsmesh.ai)

  runner login --headless
      Print URL only, don't open browser (for SSH/remote sessions)

  runner login --token <token>
      Login using a pre-generated token

  runner login --server https://self-hosted.example.com
      Login to a self-hosted AgentsMesh server

Use "runner <command> --help" for more information about a command.`)
}
