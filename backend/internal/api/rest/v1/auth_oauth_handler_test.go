package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/gin-gonic/gin"
)

// HTTP-level integration test for the OAuth redirect endpoint. Drives
// the same code path the desktop client hits when the user clicks
// "GitHub" — ensures redirect validation happens before any service
// dispatch, so a malformed or hostile `redirect=` parameter is
// rejected at the gateway and never produces an OAuth state record.
//
// Why we don't go further: completing OAuth needs a Redis-backed
// state generator + mocked GitHub credentials. That belongs in a
// dedicated integration suite (testkit + docker-compose Redis), not
// in this unit-style file. The valid-redirect cases assert "we
// passed the redirect gate" by observing the next-stage error
// ("OAuth provider not configured", produced by getOAuthConfig with
// an empty ClientID).
func TestOAuthRedirect_RedirectValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{PrimaryDomain: "agentsmesh.ai"}
	h := &AuthHandler{config: cfg}

	router := gin.New()
	router.GET("/api/v1/auth/oauth/:provider", h.OAuthRedirect("github"))

	cases := []struct {
		name           string
		redirect       string
		wantStatus     int
		wantBodyHas    string // substring match, empty = skip
		wantBodyMisses string // substring that must NOT appear
	}{
		{
			name:        "rejects different host",
			redirect:    "https://attacker.com/auth/callback",
			wantStatus:  http.StatusBadRequest,
			wantBodyHas: "Invalid redirect URL",
		},
		{
			name:        "rejects unknown scheme",
			redirect:    "javascript:alert(1)",
			wantStatus:  http.StatusBadRequest,
			wantBodyHas: "Invalid redirect URL",
		},
		{
			name:        "rejects deep link with wrong host",
			redirect:    "agentsmesh://attacker/oauth/callback",
			wantStatus:  http.StatusBadRequest,
			wantBodyHas: "Invalid redirect URL",
		},
		{
			name:        "rejects deep link with wrong path",
			redirect:    "agentsmesh://oauth/evil",
			wantStatus:  http.StatusBadRequest,
			wantBodyHas: "Invalid redirect URL",
		},
		{
			// `agentsmesh://oauth/callback` must pass redirect validation;
			// the next-stage error comes from missing OAuth config (empty
			// ClientID). If redirect validation regresses, we'd see the
			// same "Invalid redirect URL" message as the rejection cases,
			// which is exactly what this assertion guards against.
			name:           "accepts desktop deep link",
			redirect:       "agentsmesh://oauth/callback",
			wantStatus:     http.StatusBadRequest,
			wantBodyHas:    "OAuth provider not configured",
			wantBodyMisses: "Invalid redirect URL",
		},
		{
			name:           "accepts same-host https",
			redirect:       "https://agentsmesh.ai/auth/callback",
			wantStatus:     http.StatusBadRequest,
			wantBodyHas:    "OAuth provider not configured",
			wantBodyMisses: "Invalid redirect URL",
		},
		{
			name:           "accepts relative path",
			redirect:       "/auth/callback",
			wantStatus:     http.StatusBadRequest,
			wantBodyHas:    "OAuth provider not configured",
			wantBodyMisses: "Invalid redirect URL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/auth/oauth/github?redirect="+tc.redirect,
				nil,
			)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d, body=%s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantBodyHas != "" && !strings.Contains(rec.Body.String(), tc.wantBodyHas) {
				t.Errorf("body missing %q, got: %s", tc.wantBodyHas, rec.Body.String())
			}
			if tc.wantBodyMisses != "" && strings.Contains(rec.Body.String(), tc.wantBodyMisses) {
				t.Errorf("body unexpectedly contains %q: %s", tc.wantBodyMisses, rec.Body.String())
			}
		})
	}
}
