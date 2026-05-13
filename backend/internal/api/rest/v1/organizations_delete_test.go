package v1_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/anthropics/agentsmesh/backend/internal/api/rest/v1"
	orgDomain "github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/organization"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
)

// deleteOrgRepoStub is a configurable Repository fake for DeleteOrganization
// scenarios. Unlike orgRepoStub (which panics by design — used to assert that
// reserved-slug short-circuits *before* any repo call), this one lets each
// test program GetBySlug / GetMember / DeleteWithCleanup outcomes to cover the
// 200 / 403 / 404 / 500 branches of the handler.
type deleteOrgRepoStub struct {
	orgRepoStub
	org              *orgDomain.Organization
	getBySlugErr     error
	member           *orgDomain.Member
	getMemberErr     error
	deleteErr        error
	deleteWithCleanupCalls int
}

func (s *deleteOrgRepoStub) GetBySlug(context.Context, string) (*orgDomain.Organization, error) {
	return s.org, s.getBySlugErr
}
func (s *deleteOrgRepoStub) GetMember(context.Context, int64, int64) (*orgDomain.Member, error) {
	return s.member, s.getMemberErr
}
func (s *deleteOrgRepoStub) DeleteWithCleanup(context.Context, int64) error {
	s.deleteWithCleanupCalls++
	return s.deleteErr
}

func setupDeleteTestRouter(t *testing.T, repo *deleteOrgRepoStub) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.RecoveryWithWriter(io.Discard))

	orgSvc := organization.NewService(repo)
	userSvc := user.NewService(nil)
	handler := v1.NewOrganizationHandler(orgSvc, userSvc)

	g := r.Group("/api/v1/orgs")
	g.Use(func(c *gin.Context) { c.Set("user_id", int64(42)); c.Next() })
	g.DELETE("/:slug", handler.DeleteOrganization)
	return r
}

func decodeBody(t *testing.T, body io.Reader) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.NewDecoder(body).Decode(&m))
	return m
}

func TestDeleteOrganization_Success_200(t *testing.T) {
	repo := &deleteOrgRepoStub{
		org:    &orgDomain.Organization{ID: 213, Slug: "acme", Name: "Acme"},
		member: &orgDomain.Member{OrganizationID: 213, UserID: 42, Role: orgDomain.RoleOwner},
	}
	r := setupDeleteTestRouter(t, repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/orgs/acme", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, repo.deleteWithCleanupCalls)
	assert.Equal(t, "Organization deleted", decodeBody(t, w.Body)["message"])
}

func TestDeleteOrganization_NotOwner_403(t *testing.T) {
	repo := &deleteOrgRepoStub{
		org:    &orgDomain.Organization{ID: 213, Slug: "acme"},
		member: &orgDomain.Member{OrganizationID: 213, UserID: 42, Role: orgDomain.RoleMember},
	}
	r := setupDeleteTestRouter(t, repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/orgs/acme", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "OWNER_REQUIRED", decodeBody(t, w.Body)["code"])
	assert.Zero(t, repo.deleteWithCleanupCalls, "non-owner must not reach repo")
}

func TestDeleteOrganization_NotFound_404(t *testing.T) {
	repo := &deleteOrgRepoStub{getBySlugErr: orgDomain.ErrNotFound}
	r := setupDeleteTestRouter(t, repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/orgs/no-such-org", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "RESOURCE_NOT_FOUND", decodeBody(t, w.Body)["code"])
	assert.Zero(t, repo.deleteWithCleanupCalls)
}

// Regression guard for the production bug (DeleteWithCleanup hitting SQLSTATE
// 42703 on pod_bindings.channel_id): a repo-level failure must surface as a
// 500 INTERNAL_ERROR, *not* be silently dropped. If the handler ever changes
// to ignore Delete errors or returns 2xx on failure, this test catches it.
func TestDeleteOrganization_ServiceFailure_500(t *testing.T) {
	repo := &deleteOrgRepoStub{
		org:       &orgDomain.Organization{ID: 213, Slug: "acme"},
		member:    &orgDomain.Member{OrganizationID: 213, UserID: 42, Role: orgDomain.RoleOwner},
		deleteErr: errors.New("simulated db failure"),
	}
	r := setupDeleteTestRouter(t, repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/orgs/acme", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "INTERNAL_ERROR", decodeBody(t, w.Body)["code"])
	assert.Equal(t, 1, repo.deleteWithCleanupCalls)
}
