package repository

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/service/user"
	"gorm.io/gorm"
)

// Webhook-related errors
var (
	ErrNoAccessToken    = errors.New("no access token available for git provider")
	ErrWebhookNotFound  = errors.New("webhook not found")
	ErrWebhookExists    = errors.New("webhook already registered")
	ErrProviderMismatch = errors.New("user provider type does not match repository")
)

// WebhookService handles webhook registration and management for repositories
type WebhookService struct {
	db          *gorm.DB
	cfg         *config.Config
	userService *user.Service
	logger      *slog.Logger
}

// NewWebhookService creates a new webhook service
func NewWebhookService(db *gorm.DB, cfg *config.Config, userService *user.Service, logger *slog.Logger) *WebhookService {
	return &WebhookService{
		db:          db,
		cfg:         cfg,
		userService: userService,
		logger:      logger,
	}
}

// WebhookResult represents the result of a webhook registration attempt
type WebhookResult struct {
	RepoID              int64  `json:"repo_id"`
	Registered          bool   `json:"registered"`
	WebhookID           string `json:"webhook_id,omitempty"`
	NeedsManualSetup    bool   `json:"needs_manual_setup"`
	ManualWebhookURL    string `json:"manual_webhook_url,omitempty"`
	ManualWebhookSecret string `json:"manual_webhook_secret,omitempty"` // Only returned when needs_manual_setup is true
	Error               string `json:"error,omitempty"`
}

// generateWebhookSecret generates a cryptographically secure random secret
func generateWebhookSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to less secure method if crypto/rand fails
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
