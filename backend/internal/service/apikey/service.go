package apikey

import (
	apikeyDomain "github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
	"github.com/redis/go-redis/v9"
)

// Service implements the API key business logic
type Service struct {
	repo        apikeyDomain.Repository
	redisClient *redis.Client
}

// Compile-time interface check
var _ Interface = (*Service)(nil)

// NewService creates a new API key service
func NewService(repo apikeyDomain.Repository, redisClient *redis.Client) *Service {
	return &Service{
		repo:        repo,
		redisClient: redisClient,
	}
}
