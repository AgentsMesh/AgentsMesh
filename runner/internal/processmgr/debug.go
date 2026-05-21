package processmgr

import (
	"encoding/json"
	"net/http"
	"time"
)

// processView is the JSON shape served by /debug/processes. We expose enough
// to answer "which Pod owns this PID?" without leaking command lines or
// environment variables.
type processView struct {
	PID       int    `json:"pid"`
	Owner     string `json:"owner"`
	Mode      string `json:"mode"`
	StartedAt string `json:"started_at"`
	UptimeMs  int64  `json:"uptime_ms"`
	Alive     bool   `json:"alive"`
}

// HTTPHandler returns an http.Handler that serves the manager's current
// registry plus lifetime metrics. The runner mounts this at /debug/processes.
func HTTPHandler(m Manager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handles := m.List()
		views := make([]processView, 0, len(handles))
		now := time.Now()
		for _, p := range handles {
			views = append(views, processView{
				PID:       p.PID(),
				Owner:     p.Owner(),
				Mode:      p.Mode().String(),
				StartedAt: p.StartedAt().UTC().Format(time.RFC3339Nano),
				UptimeMs:  now.Sub(p.StartedAt()).Milliseconds(),
				Alive:     p.Alive(),
			})
		}

		body := struct {
			Processes []processView `json:"processes"`
			Metrics   Metrics       `json:"metrics"`
		}{
			Processes: views,
			Metrics:   currentMetrics(len(views)),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	})
}
