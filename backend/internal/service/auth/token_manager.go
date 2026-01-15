package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/agentmesh/backend/internal/domain/user"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// GenerateTokenPair generates access and refresh tokens
func (s *Service) GenerateTokenPair(u *user.User, orgID int64, role string) (*TokenPair, error) {
	return s.GenerateTokenPairWithContext(context.Background(), u, orgID, role)
}

// GenerateTokenPairWithContext generates access and refresh tokens with context
func (s *Service) GenerateTokenPairWithContext(ctx context.Context, u *user.User, orgID int64, role string) (*TokenPair, error) {
	now := time.Now()
	expiresAt := now.Add(s.config.JWTExpiration)
	refreshExpiresAt := now.Add(s.config.RefreshExpiration)

	claims := &Claims{
		UserID:         u.ID,
		Email:          u.Email,
		Username:       u.Username,
		OrganizationID: orgID,
		Role:           role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.config.Issuer,
			Subject:   u.Email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshBytes := make([]byte, 32)
	if _, err := rand.Read(refreshBytes); err != nil {
		return nil, err
	}
	refreshToken := base64.URLEncoding.EncodeToString(refreshBytes)

	// Store refresh token in Redis if available
	if s.redis != nil {
		tokenData := &RefreshTokenData{
			UserID:         u.ID,
			OrganizationID: orgID,
			Role:           role,
			CreatedAt:      now,
			ExpiresAt:      refreshExpiresAt,
		}
		if err := s.storeRefreshToken(ctx, refreshToken, tokenData); err != nil {
			return nil, fmt.Errorf("failed to store refresh token: %w", err)
		}
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, nil
}

// storeRefreshToken stores refresh token data in Redis
func (s *Service) storeRefreshToken(ctx context.Context, refreshToken string, data *RefreshTokenData) error {
	tokenHash := hashToken(refreshToken)
	key := refreshTokenPrefix + tokenHash

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	ttl := time.Until(data.ExpiresAt)
	return s.redis.Set(ctx, key, jsonData, ttl).Err()
}

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

// GenerateState generates a random state for OAuth
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GenerateTokens generates tokens for a user (used after email verification)
func (s *Service) GenerateTokens(ctx context.Context, u *user.User) (*LoginResult, error) {
	tokens, err := s.GenerateTokenPairWithContext(ctx, u, 0, "")
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
