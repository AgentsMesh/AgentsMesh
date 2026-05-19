package apikey

import (
	apikeyDomain "github.com/anthropics/agentsmesh/backend/internal/domain/apikey"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	repo        apikeyDomain.Repository
	redisClient *redis.Client
}

var _ Interface = (*Service)(nil)

func NewService(repo apikeyDomain.Repository, redisClient *redis.Client) *Service {
	return &Service{
		repo:        repo,
		redisClient: redisClient,
	}
}
