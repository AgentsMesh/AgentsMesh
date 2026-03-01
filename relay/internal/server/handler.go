package server

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/AgentsMesh/AgentsMesh/relay/internal/auth"
	"github.com/AgentsMesh/AgentsMesh/relay/internal/channel"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 64, // 64KB
	WriteBufferSize: 1024 * 64, // 64KB
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development, should be restricted in production
		return true
	},
}

// Handler handles WebSocket connections
type Handler struct {
	channelManager *channel.ChannelManager
	tokenValidator *auth.TokenValidator
	logger         *slog.Logger
}

// NewHandler creates a new WebSocket handler
func NewHandler(channelManager *channel.ChannelManager, tokenValidator *auth.TokenValidator) *Handler {
	return &Handler{
		channelManager: channelManager,
		tokenValidator: tokenValidator,
		logger:         slog.With("component", "ws_handler"),
	}
}

// HandleRunnerWS handles runner WebSocket connections (Publisher)
// Path: /runner/terminal?token=xxx
// The token contains pod_key and runner_id for authentication
// Channel is identified by pod_key (not session_id)
func (h *Handler) HandleRunnerWS(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")

	if tokenStr == "" {
		h.logger.Warn("Runner connection missing token")
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := h.tokenValidator.ValidateToken(tokenStr)
	if err != nil {
		h.logger.Warn("Invalid runner token", "error", err)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Extract pod_key from token claims (channel identifier)
	podKey := claims.PodKey

	if podKey == "" {
		h.logger.Warn("Runner token missing pod_key")
		http.Error(w, "invalid token claims", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade runner connection", "error", err)
		return
	}

	h.logger.Info("Publisher (runner) connecting",
		"pod_key", podKey,
		"runner_id", claims.RunnerID)

	if err := h.channelManager.HandlePublisherConnect(podKey, conn); err != nil {
		h.logger.Error("Failed to handle publisher connect", "error", err, "pod_key", podKey)
		conn.Close()
		return
	}
}

// HandleBrowserWS handles browser WebSocket connections (Subscriber)
// Path: /browser/terminal?token=xxx
// Channel is identified by pod_key from the token (not session_id)
func (h *Handler) HandleBrowserWS(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")

	if tokenStr == "" {
		h.logger.Warn("Browser connection missing token")
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := h.tokenValidator.ValidateToken(tokenStr)
	if err != nil {
		h.logger.Warn("Invalid token", "error", err)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Use pod_key from token as channel identifier
	podKey := claims.PodKey

	if podKey == "" {
		h.logger.Warn("Browser token missing pod_key")
		http.Error(w, "invalid token claims", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade browser connection", "error", err)
		return
	}

	// Generate subscriber ID for this browser connection
	subscriberID := uuid.New().String()

	h.logger.Info("Subscriber (browser) connecting",
		"pod_key", podKey,
		"subscriber_id", subscriberID,
		"user_id", claims.UserID)

	if err := h.channelManager.HandleSubscriberConnect(podKey, subscriberID, conn); err != nil {
		h.logger.Error("Failed to handle subscriber connect", "error", err, "pod_key", podKey)

		// Send error message before closing
		if _, ok := err.(*channel.MaxSubscribersError); ok {
			_ = conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "max subscribers reached"))
		}
		_ = conn.Close()
		return
	}
}

// HandleHealth handles health check requests
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// HandleStats handles stats requests
func (h *Handler) HandleStats(w http.ResponseWriter, r *http.Request) {
	stats := h.channelManager.Stats()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Simple JSON encoding without external dependency
	_, _ = w.Write([]byte(`{"active_channels":` + itoa(stats.ActiveChannels) +
		`,"total_subscribers":` + itoa(stats.TotalSubscribers) +
		`,"pending_publishers":` + itoa(stats.PendingPublishers) +
		`,"pending_subscribers":` + itoa(stats.PendingSubscribers) + `}`))
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
