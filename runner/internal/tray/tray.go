//go:build desktop

// Package tray provides system tray functionality for the desktop mode.
package tray

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/getlantern/systray"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/console"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/runner"
	"github.com/anthropics/agentsmesh/runner/internal/updater"
)

// Module logger for tray
var log = logger.Tray()

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

	// Update management
	updateChecker  *updater.BackgroundChecker
	gracefulUpdate *updater.GracefulUpdater
	updateInfo     *updater.UpdateInfo

	// Menu items
	statusItem    *systray.MenuItem
	toggleItem    *systray.MenuItem
	webItem       *systray.MenuItem
	logsItem      *systray.MenuItem
	updateItem    *systray.MenuItem
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
		log.Error("Failed to start web console", "error", err)
	} else {
		log.Info("Web console available", "url", t.console.GetURL())
	}

	// Build menu
	t.statusItem = systray.AddMenuItem("Status: Stopped", "Current runner status")
	t.statusItem.Disable()

	systray.AddSeparator()

	t.toggleItem = systray.AddMenuItem("Start Runner", "Start/Stop the runner")
	t.webItem = systray.AddMenuItem("Open Web Console", "Open AgentsMesh web console")
	t.logsItem = systray.AddMenuItem("View Logs", "View runner logs")

	systray.AddSeparator()

	t.updateItem = systray.AddMenuItem("Check for Updates...", "Check for available updates")
	t.autoStartItem = systray.AddMenuItemCheckbox("Start at Login", "Auto-start runner at login", false)

	systray.AddSeparator()

	t.quitItem = systray.AddMenuItem("Quit", "Quit AgentsMesh Runner")

	// Initialize update checker
	t.initUpdateChecker()

	// Handle menu events
	go t.handleEvents()

	// Auto-start runner
	t.startRunner()
}

func (t *TrayApp) onExit() {
	log.Info("Tray exiting")

	// Stop update checker
	if t.updateChecker != nil {
		t.updateChecker.Stop()
	}

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

		case <-t.updateItem.ClickedCh:
			t.checkForUpdates()

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
		log.Error("Failed to create runner", "error", err)
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
			log.Error("Runner error", "error", err)
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
			log.Error("Failed to open web console", "error", err)
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
			log.Error("Failed to open logs", "error", err)
		}
	}
}

func (t *TrayApp) toggleAutoStart() {
	if t.autoStartItem.Checked() {
		t.autoStartItem.Uncheck()
		// TODO: Remove from system auto-start
		log.Info("Auto-start disabled")
	} else {
		t.autoStartItem.Check()
		// TODO: Add to system auto-start
		log.Info("Auto-start enabled")
	}
}

// initUpdateChecker initializes the background update checker.
func (t *TrayApp) initUpdateChecker() {
	if !t.cfg.AutoUpdate.Enabled {
		log.Info("Auto-update is disabled")
		return
	}

	// Create updater
	opts := []updater.Option{}
	if t.cfg.AutoUpdate.Channel == "beta" {
		opts = append(opts, updater.WithPrerelease(true))
	}
	u := updater.New(t.version, opts...)

	// Create graceful updater with pod counter
	podCounter := func() int {
		t.mu.RLock()
		r := t.runner
		t.mu.RUnlock()
		if r != nil {
			return r.GetActivePodCount()
		}
		return 0
	}

	t.gracefulUpdate = updater.NewGracefulUpdater(
		u, podCounter,
		updater.WithMaxWaitTime(t.cfg.AutoUpdate.MaxWaitTime),
		updater.WithStatusCallback(t.onUpdateStatus),
	)

	// Create background checker
	t.updateChecker = updater.NewBackgroundChecker(
		u, t.gracefulUpdate, t.cfg.AutoUpdate.CheckInterval,
		updater.WithOnUpdate(t.onUpdateAvailable),
		updater.WithAutoApply(t.cfg.AutoUpdate.AutoApply),
	)

	// Start background checks
	t.updateChecker.Start(context.Background())
	log.Info("Auto-update enabled", "checkInterval", t.cfg.AutoUpdate.CheckInterval)
}

// onUpdateAvailable is called when a new update is found.
func (t *TrayApp) onUpdateAvailable(info *updater.UpdateInfo) {
	t.mu.Lock()
	t.updateInfo = info
	t.mu.Unlock()

	t.updateItem.SetTitle(fmt.Sprintf("Update Available (%s)", info.LatestVersion))

	if t.console != nil {
		t.console.AddLog("info", fmt.Sprintf("Update available: %s -> %s", info.CurrentVersion, info.LatestVersion))
	}
}

// onUpdateStatus is called when the update status changes.
func (t *TrayApp) onUpdateStatus(state updater.State, info *updater.UpdateInfo, activePods int) {
	switch state {
	case updater.StateDraining:
		t.updateItem.SetTitle(fmt.Sprintf("⏳ Waiting to Update (%d pods active)", activePods))
	case updater.StateDownloading:
		t.updateItem.SetTitle("🔄 Downloading Update...")
	case updater.StateApplying:
		t.updateItem.SetTitle("🔄 Applying Update...")
	case updater.StateRestarting:
		t.updateItem.SetTitle("🔄 Restarting...")
	case updater.StateIdle:
		t.mu.RLock()
		hasUpdate := t.updateInfo != nil && t.updateInfo.HasUpdate
		t.mu.RUnlock()
		if hasUpdate {
			t.updateItem.SetTitle(fmt.Sprintf("Update Available (%s)", t.updateInfo.LatestVersion))
		} else {
			t.updateItem.SetTitle("Check for Updates...")
		}
	}
}

// checkForUpdates manually triggers an update check.
func (t *TrayApp) checkForUpdates() {
	t.updateItem.SetTitle("Checking for Updates...")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Use existing background checker if available
		var info *updater.UpdateInfo
		var err error

		if t.updateChecker != nil {
			info, err = t.updateChecker.CheckNow(ctx)
		} else {
			// Fallback: create a new updater if checker not initialized
			opts := []updater.Option{}
			if t.cfg.AutoUpdate.Channel == "beta" {
				opts = append(opts, updater.WithPrerelease(true))
			}
			u := updater.New(t.version, opts...)
			info, err = u.CheckForUpdate(ctx)
		}

		if err != nil {
			log.Error("Update check failed", "error", err)
			t.updateItem.SetTitle("Check for Updates...")
			if t.console != nil {
				t.console.AddLog("error", fmt.Sprintf("Update check failed: %v", err))
			}
			return
		}

		if !info.HasUpdate {
			t.updateItem.SetTitle("Up to Date ✓")
			if t.console != nil {
				t.console.AddLog("info", "You are running the latest version")
			}
			// Reset title after a few seconds
			go func() {
				time.Sleep(3 * time.Second)
				t.updateItem.SetTitle("Check for Updates...")
			}()
			return
		}

		t.onUpdateAvailable(info)
	}()
}
