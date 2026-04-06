package console

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleStatus handles GET /api/status
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

// handleLogs handles GET /api/logs
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

// handleConfig handles GET /api/config
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

// handleRestart handles POST /api/actions/restart
func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.AddLog("info", "Restart requested via web console")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "error",
		"message": "Restart not implemented",
	})
}

// handleStop handles POST /api/actions/stop
func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.AddLog("info", "Stop requested via web console")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "error",
		"message": "Stop not implemented",
	})
}
