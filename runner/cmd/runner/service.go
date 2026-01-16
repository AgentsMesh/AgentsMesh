package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kardianos/service"

	svc "github.com/anthropics/agentsmesh/runner/internal/service"
)

// runService handles the "service" subcommand for system service management.
func runService(args []string) {
	if len(args) < 1 {
		printServiceUsage()
		os.Exit(1)
	}

	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "install":
		runServiceInstall(actionArgs)
	case "uninstall":
		runServiceUninstall()
	case "start":
		runServiceStart()
	case "stop":
		runServiceStop()
	case "restart":
		runServiceRestart()
	case "status":
		runServiceStatus()
	case "help", "-h", "--help":
		printServiceUsage()
	default:
		fmt.Printf("Unknown service action: %s\n", action)
		printServiceUsage()
		os.Exit(1)
	}
}

func printServiceUsage() {
	fmt.Println(`Manage AgentsMesh Runner as a system service.

Usage:
  runner service <action> [options]

Actions:
  install     Install runner as a system service
  uninstall   Remove runner system service
  start       Start the service
  stop        Stop the service
  restart     Restart the service
  status      Show service status

Options for 'install':
  --config    Path to config file (default: ~/.agentsmesh/config.yaml)

Examples:
  runner service install
  runner service install --config /etc/agentsmesh/config.yaml
  runner service start
  runner service status`)
}

func runServiceInstall(args []string) {
	fs := flag.NewFlagSet("service install", flag.ExitOnError)
	configPath := fs.String("config", "", "Path to config file")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Use default config path if not specified
	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = svc.GetDefaultConfigPath()
	}

	// Check if config exists
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		fmt.Printf("Error: Config file not found at %s\n", cfgPath)
		fmt.Println("Please run 'runner register' first to create the configuration.")
		os.Exit(1)
	}

	if err := svc.Install(cfgPath); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Service installed successfully.")
	fmt.Println("Use 'runner service start' to start the service.")
}

func runServiceUninstall() {
	if err := svc.Uninstall(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Service uninstalled successfully.")
}

func runServiceStart() {
	if err := svc.Start(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Service started.")
}

func runServiceStop() {
	if err := svc.Stop(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Service stopped.")
}

func runServiceRestart() {
	if err := svc.Restart(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Service restarted.")
}

func runServiceStatus() {
	status, err := svc.GetStatus()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	var statusText string
	switch status {
	case service.StatusRunning:
		statusText = "Running"
	case service.StatusStopped:
		statusText = "Stopped"
	default:
		statusText = "Unknown"
	}

	fmt.Printf("Service Status: %s\n", statusText)
}
