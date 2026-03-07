package relay

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenClaims represents JWT claims for relay token
// Note: SessionID has been removed - channels are now identified by PodKey only
type TokenClaims struct {
	PodKey   string `json:"pod_key"`
	RunnerID int64  `json:"runner_id"`
	UserID   int64  `json:"user_id"` // 0 for runner tokens
	OrgID    int64  `json:"org_id"`
	jwt.RegisteredClaims
}

// TokenGenerator generates relay tokens
type TokenGenerator struct {
	secretKey []byte
	issuer    string
}

// NewTokenGenerator creates a new token generator.
// Panics if secret is empty to prevent signing tokens with a zero-length HMAC key.
func NewTokenGenerator(secret, issuer string) *TokenGenerator {
	if secret == "" {
		panic("relay token secret must not be empty")
	}
	return &TokenGenerator{
		secretKey: []byte(secret),
		issuer:    issuer,
	}
}

// GenerateToken generates a relay token
// Note: sessionID parameter has been removed - channels are identified by PodKey only
func (g *TokenGenerator) GenerateToken(podKey string, runnerID, userID, orgID int64, expiry time.Duration) (string, error) {
	if podKey == "" {
		return "", fmt.Errorf("podKey must not be empty")
	}
	if expiry <= 0 {
		return "", fmt.Errorf("expiry must be positive, got %v", expiry)
	}
	now := time.Now()
	expiresAt := now.Add(expiry)

	claims := &TokenClaims{
		PodKey:   podKey,
		RunnerID: runnerID,
		UserID:   userID,
		OrgID:    orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    g.issuer,
			Subject:   podKey,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(g.secretKey)
}
