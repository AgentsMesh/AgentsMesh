package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// revokeRefreshToken removes a refresh token from Redis
func (s *Service) revokeRefreshToken(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	key := refreshTokenPrefix + tokenHash
	return s.redis.Del(ctx, key).Err()
}

// RevokeToken revokes an access token by adding it to the blacklist
func (s *Service) RevokeToken(ctx context.Context, token string) error {
	if s.redis == nil {
		return nil
	}

	claims, err := s.ValidateToken(token)
	if err != nil && !errors.Is(err, ErrTokenExpired) {
		return nil
	}

	var ttl time.Duration
	if claims != nil && claims.ExpiresAt != nil {
		ttl = time.Until(claims.ExpiresAt.Time)
		if ttl <= 0 {
			return nil
		}
	} else {
		ttl = s.config.JWTExpiration
	}

	tokenHash := hashToken(token)
	key := tokenBlacklistKey + tokenHash
	return s.redis.Set(ctx, key, "1", ttl).Err()
}

// RevokeAllUserTokens revokes all tokens for a user
func (s *Service) RevokeAllUserTokens(ctx context.Context, userID int64) error {
	if s.redis == nil {
		return nil
	}

	pattern := refreshTokenPrefix + "*"
	iter := s.redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		data, err := s.redis.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		var tokenData RefreshTokenData
		if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
			continue
		}
		if tokenData.UserID == userID {
			s.redis.Del(ctx, key)
		}
	}
	return iter.Err()
}
