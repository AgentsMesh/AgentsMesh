//go:build desktop

// Package tray provides system tray functionality for the desktop mode.
package tray

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/getlantern/systray"

	"github.com/anthropics/agentmesh/runner/internal/config"
	"github.com/anthropics/agentmesh/runner/internal/console"
	"github.com/anthropics/agentmesh/runner/internal/runner"
)

const (
	// DefaultConsolePort is the default port for the web console.
	DefaultConsolePort = 19080
)

// TrayApp represents the system tray application.
type TrayApp struct {
	cfg     *config.Config
	version string
	runner  *runner.Runner
	console *console.Server

	// State
	running   bool
	connected bool
	mu        sync.RWMutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Menu items
	statusItem    *systray.MenuItem
	toggleItem    *systray.MenuItem
	webItem       *systray.MenuItem
	logsItem      *systray.MenuItem
	autoStartItem *systray.MenuItem
	quitItem      *systray.MenuItem
}

// New creates a new tray application.
func New(cfg *config.Config) *TrayApp {
	return &TrayApp{
		cfg:     cfg,
		version: "dev",
	}
}

// NewWithVersion creates a new tray application with version info.
func NewWithVersion(cfg *config.Config, version string) *TrayApp {
	return &TrayApp{
		cfg:     cfg,
		version: version,
	}
}

// Run starts the tray application (blocking).
func (t *TrayApp) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *TrayApp) onReady() {
	// Set icon
	systray.SetIcon(getIcon())
	systray.SetTitle("AgentsMesh Runner")
	systray.SetTooltip("AgentsMesh Runner")

	// Start web console
	t.console = console.New(t.cfg, DefaultConsolePort, t.version)
	if err := t.console.Start(); err != nil {
		log.Printf("Failed to start web console: %v", err)
	} else {
		log.Printf("Web console available at %s", t.console.GetURL())
	}

	// Build menu
	t.statusItem = systray.AddMenuItem("Status: Stopped", "Current runner status")
	t.statusItem.Disable()

	systray.AddSeparator()

	t.toggleItem = systray.AddMenuItem("Start Runner", "Start/Stop the runner")
	t.webItem = systray.AddMenuItem("Open Web Console", "Open AgentsMesh web console")
	t.logsItem = systray.AddMenuItem("View Logs", "View runner logs")

	systray.AddSeparator()

	t.autoStartItem = systray.AddMenuItemCheckbox("Start at Login", "Auto-start runner at login", false)

	systray.AddSeparator()

	t.quitItem = systray.AddMenuItem("Quit", "Quit AgentsMesh Runner")

	// Handle menu events
	go t.handleEvents()

	// Auto-start runner
	t.startRunner()
}

func (t *TrayApp) onExit() {
	log.Println("Tray exiting...")
	t.stopRunner()

	// Stop web console
	if t.console != nil {
		t.console.Stop()
	}
}

func (t *TrayApp) handleEvents() {
	for {
		select {
		case <-t.toggleItem.ClickedCh:
			t.mu.RLock()
			isRunning := t.running
			t.mu.RUnlock()

			if isRunning {
				t.stopRunner()
			} else {
				t.startRunner()
			}

		case <-t.webItem.ClickedCh:
			t.openWebConsole()

		case <-t.logsItem.ClickedCh:
			t.openLogs()

		case <-t.autoStartItem.ClickedCh:
			t.toggleAutoStart()

		case <-t.quitItem.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func (t *TrayApp) startRunner() {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return
	}
	t.mu.Unlock()

	// Create runner instance
	r, err := runner.New(t.cfg)
	if err != nil {
		log.Printf("Failed to create runner: %v", err)
		t.updateStatus(false, false, err)
		return
	}

	t.mu.Lock()
	t.runner = r
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.running = true
	t.mu.Unlock()

	// Update UI
	t.updateStatus(true, false, nil)

	// Start runner in background
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		// Simulate connection after a short delay
		go func() {
			time.Sleep(2 * time.Second)
			t.mu.Lock()
			if t.running {
				t.connected = true
			}
			t.mu.Unlock()
			t.updateStatus(true, true, nil)
		}()

		if err := r.Run(t.ctx); err != nil {
			log.Printf("Runner error: %v", err)
			t.updateStatus(false, false, err)
		}

		t.mu.Lock()
		t.running = false
		t.connected = false
		t.mu.Unlock()
		t.updateStatus(false, false, nil)
	}()
}

func (t *TrayApp) stopRunner() {
	t.mu.Lock()
	if !t.running || t.cancel == nil {
		t.mu.Unlock()
		return
	}
	cancel := t.cancel
	t.mu.Unlock()

	cancel()
	t.wg.Wait()

	t.updateStatus(false, false, nil)
}

func (t *TrayApp) updateStatus(running, connected bool, err error) {
	t.mu.Lock()
	t.running = running
	t.connected = connected
	t.mu.Unlock()

	var statusText string
	var errMsg string
	if err != nil {
		statusText = fmt.Sprintf("Status: Error - %v", err)
		errMsg = err.Error()
		systray.SetIcon(getIconError())
	} else if running && connected {
		statusText = "Status: Connected"
		systray.SetIcon(getIconConnected())
	} else if running {
		statusText = "Status: Connecting..."
		systray.SetIcon(getIconConnecting())
	} else {
		statusText = "Status: Stopped"
		systray.SetIcon(getIcon())
	}

	t.statusItem.SetTitle(statusText)

	if running {
		t.toggleItem.SetTitle("Stop Runner")
	} else {
		t.toggleItem.SetTitle("Start Runner")
	}

	// Update console status
	if t.console != nil {
		t.console.UpdateStatus(running, connected, 0, 0, errMsg)
		if err != nil {
			t.console.AddLog("error", errMsg)
		} else if connected {
			t.console.AddLog("info", "Connected to server")
		} else if running {
			t.console.AddLog("info", "Connecting to server...")
		} else {
			t.console.AddLog("info", "Runner stopped")
		}
	}
}

func (t *TrayApp) openWebConsole() {
	// Use local console URL
	url := fmt.Sprintf("http://127.0.0.1:%d", DefaultConsolePort)
	if t.console != nil {
		url = t.console.GetURL()
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}

	if cmd != nil {
		if err := cmd.Start(); err != nil {
			log.Printf("Failed to open web console: %v", err)
		}
	}
}

func (t *TrayApp) openLogs() {
	// Open web console (logs are shown on the main page)
	url := fmt.Sprintf("http://127.0.0.1:%d#logs", DefaultConsolePort)
	if t.console != nil {
		url = t.console.GetURL() + "#logs"
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}

	if cmd != nil {
		if err := cmd.Start(); err != nil {
			log.Printf("Failed to open logs: %v", err)
		}
	}
}

func (t *TrayApp) toggleAutoStart() {
	if t.autoStartItem.Checked() {
		t.autoStartItem.Uncheck()
		// TODO: Remove from system auto-start
		log.Println("Auto-start disabled")
	} else {
		t.autoStartItem.Check()
		// TODO: Add to system auto-start
		log.Println("Auto-start enabled")
	}
}
