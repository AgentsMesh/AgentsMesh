// Package console provides a local web console for managing the runner.
package console

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/config"
)

//go:embed static/*
var staticFiles embed.FS

// Server represents the web console server.
type Server struct {
	cfg        *config.Config
	port       int
	httpServer *http.Server

	// Status tracking
	status   *Status
	statusMu sync.RWMutex

	// Log buffer
	logBuffer *LogBuffer
}

// Status represents the current runner status.
type Status struct {
	Running      bool      `json:"running"`
	Connected    bool      `json:"connected"`
	ServerURL    string    `json:"server_url"`
	NodeID       string    `json:"node_id"`
	OrgSlug      string    `json:"org_slug"`
	Version      string    `json:"version"`
	Uptime       string    `json:"uptime"`
	StartTime    time.Time `json:"start_time"`
	ActivePods   int       `json:"active_pods"`
	TotalPods    int       `json:"total_pods"`
	LastError    string    `json:"last_error,omitempty"`
	Platform     string    `json:"platform"`
	GoVersion    string    `json:"go_version"`
}

// LogEntry represents a single log entry.
type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

// LogBuffer is a circular buffer for log entries.
type LogBuffer struct {
	entries []LogEntry
	maxSize int
	mu      sync.RWMutex
}

// NewLogBuffer creates a new log buffer.
func NewLogBuffer(maxSize int) *LogBuffer {
	return &LogBuffer{
		entries: make([]LogEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add adds a log entry to the buffer.
func (b *LogBuffer) Add(level, message string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry := LogEntry{
		Time:    time.Now(),
		Level:   level,
		Message: message,
	}

	if len(b.entries) >= b.maxSize {
		// Remove oldest entry
		b.entries = b.entries[1:]
	}
	b.entries = append(b.entries, entry)
}

// GetAll returns all log entries.
func (b *LogBuffer) GetAll() []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]LogEntry, len(b.entries))
	copy(result, b.entries)
	return result
}

// GetRecent returns the most recent n entries.
func (b *LogBuffer) GetRecent(n int) []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if n >= len(b.entries) {
		result := make([]LogEntry, len(b.entries))
		copy(result, b.entries)
		return result
	}

	start := len(b.entries) - n
	result := make([]LogEntry, n)
	copy(result, b.entries[start:])
	return result
}

// New creates a new web console server.
func New(cfg *config.Config, port int, version string) *Server {
	s := &Server{
		cfg:       cfg,
		port:      port,
		logBuffer: NewLogBuffer(1000),
		status: &Status{
			Running:   false,
			Connected: false,
			ServerURL: cfg.ServerURL,
			NodeID:    cfg.NodeID,
			OrgSlug:   cfg.OrgSlug,
			Version:   version,
			StartTime: time.Now(),
			Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			GoVersion: runtime.Version(),
		},
	}

	return s
}

// Start starts the web console server.
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/actions/restart", s.handleRestart)
	mux.HandleFunc("/api/actions/stop", s.handleStop)

	// Static files (embedded)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return fmt.Errorf("failed to get static files: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", s.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("[console] Starting web console on http://127.0.0.1:%d", s.port)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[console] Server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the web console server.
func (s *Server) Stop() error {
	if s.httpServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// GetURL returns the console URL.
func (s *Server) GetURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", s.port)
}

// UpdateStatus updates the runner status.
func (s *Server) UpdateStatus(running, connected bool, activePods, totalPods int, lastError string) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()

	s.status.Running = running
	s.status.Connected = connected
	s.status.ActivePods = activePods
	s.status.TotalPods = totalPods
	s.status.LastError = lastError

	if running {
		s.status.Uptime = time.Since(s.status.StartTime).Round(time.Second).String()
	}
}

// AddLog adds a log entry.
func (s *Server) AddLog(level, message string) {
	s.logBuffer.Add(level, message)
}

// API Handlers

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.statusMu.RLock()
	status := *s.status
	s.statusMu.RUnlock()

	// Update uptime
	status.Uptime = time.Since(status.StartTime).Round(time.Second).String()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get recent logs
	logs := s.logBuffer.GetRecent(100)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs": logs,
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return sanitized config (no secrets)
	cfg := map[string]interface{}{
		"server_url":          s.cfg.ServerURL,
		"node_id":             s.cfg.NodeID,
		"org_slug":            s.cfg.OrgSlug,
		"max_concurrent_pods": s.cfg.MaxConcurrentPods,
		"workspace_root":      s.cfg.WorkspaceRoot,
		"default_agent":       s.cfg.DefaultAgent,
		"log_level":           s.cfg.LogLevel,
		"health_check_port":   s.cfg.HealthCheckPort,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.AddLog("info", "Restart requested via web console")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Restart signal sent",
	})

	// TODO: Implement actual restart logic
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.AddLog("info", "Stop requested via web console")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Stop signal sent",
	})

	// TODO: Implement actual stop logic
}

// GetConfigDir returns the config directory path.
func GetConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".agentsmesh")
}
