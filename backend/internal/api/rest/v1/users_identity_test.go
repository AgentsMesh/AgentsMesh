package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/organization"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	orgService "github.com/AgentsMesh/AgentsMesh/backend/internal/service/organization"
	userService "github.com/AgentsMesh/AgentsMesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Organization and Identity Tests
// ===========================================

func TestListUserOrganizations(t *testing.T) {
	handler, mockUserSvc, mockOrgSvc, router := setupUserHandlerTest()

	testUser := &user.User{ID: 1, Email: "test@example.com", Username: "testuser"}
	mockUserSvc.AddUser(testUser)

	testOrg := &organization.Organization{
		ID:   1,
		Name: "Test Org",
		Slug: "test-org",
	}
	mockOrgSvc.AddOrg(testOrg)
	mockOrgSvc.SetMember(1, 1, organization.RoleOwner)

	router.GET("/users/me/organizations", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.ListUserOrganizations(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me/organizations", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListUserOrganizationsError(t *testing.T) {
	handler, _, mockOrgSvc, router := setupUserHandlerTest()

	mockOrgSvc.ListByUserErr = orgService.ErrOrganizationNotFound

	router.GET("/users/me/organizations", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.ListUserOrganizations(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me/organizations", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestListIdentities(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	testUser := &user.User{ID: 1, Email: "test@example.com", Username: "testuser"}
	mockUserSvc.AddUser(testUser)
	mockUserSvc.AddIdentity(1, &user.Identity{
		UserID:   1,
		Provider: "github",
	})

	router.GET("/users/me/identities", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.ListIdentities(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me/identities", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListIdentitiesError(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	mockUserSvc.ListIdentitiesErr = userService.ErrUserNotFound

	router.GET("/users/me/identities", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.ListIdentities(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me/identities", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestDeleteIdentity(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	passwordHash := "hashed"
	testUser := &user.User{
		ID:           1,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: &passwordHash,
	}
	mockUserSvc.AddUser(testUser)
	mockUserSvc.AddIdentity(1, &user.Identity{
		UserID:   1,
		Provider: "github",
	})

	router.DELETE("/users/me/identities/:provider", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.DeleteIdentity(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/users/me/identities/github", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteIdentityUserNotFound(t *testing.T) {
	handler, _, _, router := setupUserHandlerTest()

	router.DELETE("/users/me/identities/:provider", func(c *gin.Context) {
		setUserContext(c, 999)
		handler.DeleteIdentity(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/users/me/identities/github", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestDeleteIdentityCannotRemoveLastLogin(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	testUser := &user.User{
		ID:           1,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: nil, // No password
	}
	mockUserSvc.AddUser(testUser)
	mockUserSvc.AddIdentity(1, &user.Identity{
		UserID:   1,
		Provider: "github",
	})

	router.DELETE("/users/me/identities/:provider", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.DeleteIdentity(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/users/me/identities/github", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}
