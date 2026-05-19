package repository

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
)

var (
	ErrNoAccessToken    = errors.New("no access token available for git provider")
	ErrWebhookNotFound  = errors.New("webhook not found")
	ErrWebhookExists    = errors.New("webhook already registered")
	ErrProviderMismatch = errors.New("user provider type does not match repository")
)

type WebhookService struct {
	repo        gitprovider.RepositoryRepo
	cfg         *config.Config
	userService *user.Service
	logger      *slog.Logger
}

func NewWebhookService(repo gitprovider.RepositoryRepo, cfg *config.Config, userService *user.Service, logger *slog.Logger) *WebhookService {
	return &WebhookService{
		repo:        repo,
		cfg:         cfg,
		userService: userService,
		logger:      logger,
	}
}

type WebhookResult struct {
	RepoID              int64  `json:"repo_id"`
	Registered          bool   `json:"registered"`
	WebhookID           string `json:"webhook_id,omitempty"`
	NeedsManualSetup    bool   `json:"needs_manual_setup"`
	ManualWebhookURL    string `json:"manual_webhook_url,omitempty"`
	ManualWebhookSecret string `json:"manual_webhook_secret,omitempty"` // Only returned when needs_manual_setup is true
	Error               string `json:"error,omitempty"`
}

func generateWebhookSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
