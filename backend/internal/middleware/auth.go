package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the JWT claims
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Claims is an alias for JWTClaims for convenience
type Claims = JWTClaims

// AuthMiddleware validates JWT tokens
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Get token from Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Check Bearer prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// If no token in header, check query parameter (for WebSocket connections)
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			apierr.AbortUnauthorized(c, apierr.AUTH_REQUIRED, "Authorization is required")
			return
		}

		// Parse and validate token
		claims := &JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			apierr.AbortUnauthorized(c, apierr.INVALID_TOKEN, "Invalid or expired token")
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)
		c.Set("claims", claims) // Also set full claims for WebSocket handlers

		c.Next()
	}
}

// GenerateToken generates a new JWT token for a user
func GenerateToken(userID int64, email, username, jwtSecret string, expirationHours int) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Email:    email,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expirationHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// OptionalAuthMiddleware validates JWT tokens but doesn't require them
func OptionalAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
		claims := &JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})

		if err == nil && token.Valid {
			c.Set("user_id", claims.UserID)
			c.Set("email", claims.Email)
			c.Set("username", claims.Username)
		}

		c.Next()
	}
}
