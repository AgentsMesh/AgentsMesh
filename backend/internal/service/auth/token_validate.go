package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	return s.ValidateTokenWithContext(context.Background(), tokenString)
}

func (s *Service) ValidateTokenWithContext(ctx context.Context, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if s.redis != nil {
		if revoked, _ := s.isTokenBlacklisted(ctx, tokenString); revoked {
			return nil, ErrTokenRevoked
		}
	}

	return claims, nil
}

func (s *Service) isTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	tokenHash := hashToken(token)
	key := tokenBlacklistKey + tokenHash
	exists, err := s.redis.Exists(ctx, key).Result()
	return exists > 0, err
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*LoginResult, error) {
	if s.redis == nil {
		slog.ErrorContext(ctx, "refresh token attempted without redis")
		return nil, ErrInvalidRefreshToken
	}

	tokenData, err := s.validateRefreshToken(ctx, refreshToken)
	if err != nil {
		slog.WarnContext(ctx, "refresh token validation failed", "error", err)
		return nil, err
	}

	u, err := s.userService.GetByID(ctx, tokenData.UserID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get user for token refresh", "user_id", tokenData.UserID, "error", err)
		return nil, err
	}

	if !u.IsActive {
		slog.WarnContext(ctx, "token refresh denied for disabled user", "user_id", u.ID)
		return nil, ErrUserDisabled
	}

	if err := s.revokeRefreshToken(ctx, refreshToken); err != nil {
		slog.WarnContext(ctx, "failed to revoke old refresh token during rotation", "user_id", u.ID, "error", err)
	}

	tokens, err := s.GenerateTokenPairWithContext(ctx, u, tokenData.OrganizationID, tokenData.Role)
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate new token pair during refresh", "user_id", u.ID, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "token refreshed", "user_id", u.ID, "org_id", tokenData.OrganizationID)
	return &LoginResult{
		User:         u,
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
	}, nil
}

func (s *Service) validateRefreshToken(ctx context.Context, refreshToken string) (*RefreshTokenData, error) {
	tokenHash := hashToken(refreshToken)
	key := refreshTokenPrefix + tokenHash

	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	var tokenData RefreshTokenData
	if err := json.Unmarshal([]byte(data), &tokenData); err != nil {
		return nil, fmt.Errorf("failed to parse refresh token data: %w", err)
	}

	if time.Now().After(tokenData.ExpiresAt) {
		s.redis.Del(ctx, key)
		return nil, ErrRefreshExpired
	}

	return &tokenData, nil
}
