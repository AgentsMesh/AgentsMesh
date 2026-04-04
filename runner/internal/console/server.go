// Package console provides a local web console for managing the runner.
package console

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

// Module logger for console
var log = logger.Console()

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
	Running    bool      `json:"running"`
	Connected  bool      `json:"connected"`
	ServerURL  string    `json:"server_url"`
	NodeID     string    `json:"node_id"`
	OrgSlug    string    `json:"org_slug"`
	Version    string    `json:"version"`
	Uptime     string    `json:"uptime"`
	StartTime  time.Time `json:"start_time"`
	ActivePods int       `json:"active_pods"`
	TotalPods  int       `json:"total_pods"`
	LastError  string    `json:"last_error,omitempty"`
	Platform   string    `json:"platform"`
	GoVersion  string    `json:"go_version"`
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

	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Bind port synchronously so that callers (Supervisor) see bind errors immediately.
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("console server failed to bind %s: %w", addr, err)
	}

	log.Info("Starting web console", "url", fmt.Sprintf("http://127.0.0.1:%d", s.port))

	safego.Go("console-http-listen", func() {
		if err := s.httpServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Error("Server error", "error", err)
		}
	})

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

// GetConfigDir returns the config directory path.
func GetConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".agentsmesh")
}
