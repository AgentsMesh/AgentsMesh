package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// hashToken creates a SHA-256 hash of the token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// ValidateToken validates a JWT token
func (s *Service) ValidateToken(tokenString string) (*Claims, error) {
	return s.ValidateTokenWithContext(context.Background(), tokenString)
}

// ValidateTokenWithContext validates a JWT token with context and blacklist check
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

	// Check if token is blacklisted (revoked)
	if s.redis != nil {
		if revoked, _ := s.isTokenBlacklisted(ctx, tokenString); revoked {
			return nil, ErrTokenRevoked
		}
	}

	return claims, nil
}

// isTokenBlacklisted checks if a token has been revoked
func (s *Service) isTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	tokenHash := hashToken(token)
	key := tokenBlacklistKey + tokenHash
	exists, err := s.redis.Exists(ctx, key).Result()
	return exists > 0, err
}

// RefreshTokens generates new tokens using refresh token
func (s *Service) RefreshTokens(ctx context.Context, accessToken, refreshToken string) (*TokenPair, error) {
	// Parse expired access token to get user info
	token, _ := jwt.ParseWithClaims(accessToken, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Get user
	u, err := s.userService.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// Generate new token pair
	return s.GenerateTokenPair(u, claims.OrganizationID, claims.Role)
}

// RefreshToken refreshes access token using refresh token stored in Redis
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*LoginResult, error) {
	if s.redis == nil {
		return nil, ErrInvalidRefreshToken
	}

	tokenData, err := s.validateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	u, err := s.userService.GetByID(ctx, tokenData.UserID)
	if err != nil {
		return nil, err
	}

	if !u.IsActive {
		return nil, ErrUserDisabled
	}

	// Invalidate old refresh token (token rotation for security)
	if err := s.revokeRefreshToken(ctx, refreshToken); err != nil {
		// Log but don't fail
	}

	tokens, err := s.GenerateTokenPairWithContext(ctx, u, tokenData.OrganizationID, tokenData.Role)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User:         u,
		Token:        tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
	}, nil
}

// validateRefreshToken validates a refresh token against Redis storage
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
