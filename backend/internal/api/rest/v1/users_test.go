package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	orgService "github.com/anthropics/agentsmesh/backend/internal/service/organization"
	userService "github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupUserHandlerTest() (*UserHandler, *userService.MockService, *orgService.MockService, *gin.Engine) {
	mockUserSvc := userService.NewMockService()
	mockOrgSvc := orgService.NewMockService()
	handler := NewUserHandler(mockUserSvc, mockOrgSvc)

	router := gin.New()
	return handler, mockUserSvc, mockOrgSvc, router
}

func setUserContext(c *gin.Context, userID int64) {
	c.Set("user_id", userID)
}

func TestNewUserHandler(t *testing.T) {
	handler, _, _, _ := setupUserHandlerTest()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestGetCurrentUser(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	// Add test user
	testUser := &user.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		IsActive: true,
	}
	mockUserSvc.AddUser(testUser)

	router.GET("/users/me", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.GetCurrentUser(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	userData := resp["user"].(map[string]interface{})
	if userData["email"] != "test@example.com" {
		t.Errorf("expected email test@example.com, got %v", userData["email"])
	}
}

func TestGetCurrentUserNotFound(t *testing.T) {
	handler, _, _, router := setupUserHandlerTest()

	router.GET("/users/me", func(c *gin.Context) {
		setUserContext(c, 999) // Non-existent user
		handler.GetCurrentUser(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestUpdateCurrentUser(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	testUser := &user.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		IsActive: true,
	}
	mockUserSvc.AddUser(testUser)

	router.PUT("/users/me", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.UpdateCurrentUser(c)
	})

	body := `{"name": "New Name", "avatar_url": "https://example.com/avatar.png"}`
	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify update was called
	if len(mockUserSvc.UpdatedUsers) != 1 {
		t.Errorf("expected 1 update call, got %d", len(mockUserSvc.UpdatedUsers))
	}
}

func TestUpdateCurrentUserInvalidJSON(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	testUser := &user.User{ID: 1, Email: "test@example.com", Username: "testuser"}
	mockUserSvc.AddUser(testUser)

	router.PUT("/users/me", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.UpdateCurrentUser(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/users/me", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestChangePassword(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	testUser := &user.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		IsActive: true,
	}
	mockUserSvc.AddUser(testUser)

	router.POST("/users/me/password", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.ChangePassword(c)
	})

	body := `{"current_password": "oldpass", "new_password": "newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/users/me/password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify authentication was attempted
	if len(mockUserSvc.AuthAttempts) != 1 {
		t.Errorf("expected 1 auth attempt, got %d", len(mockUserSvc.AuthAttempts))
	}
}

func TestChangePasswordInvalidJSON(t *testing.T) {
	handler, _, _, router := setupUserHandlerTest()

	router.POST("/users/me/password", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.ChangePassword(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/users/me/password", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestChangePasswordUserNotFound(t *testing.T) {
	handler, _, _, router := setupUserHandlerTest()

	router.POST("/users/me/password", func(c *gin.Context) {
		setUserContext(c, 999)
		handler.ChangePassword(c)
	})

	body := `{"current_password": "oldpass", "new_password": "newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/users/me/password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestChangePasswordWrongCurrentPassword(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	testUser := &user.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		IsActive: true,
	}
	mockUserSvc.AddUser(testUser)
	mockUserSvc.AuthenticateErr = userService.ErrInvalidCredentials

	router.POST("/users/me/password", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.ChangePassword(c)
	})

	body := `{"current_password": "wrongpass", "new_password": "newpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/users/me/password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

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

func TestSearchUsers(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	testUser := &user.User{ID: 1, Email: "test@example.com", Username: "testuser"}
	mockUserSvc.AddUser(testUser)

	router.GET("/users/search", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.SearchUsers(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/search?q=test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify search was called
	if len(mockUserSvc.SearchQueries) != 1 {
		t.Errorf("expected 1 search query, got %d", len(mockUserSvc.SearchQueries))
	}
	if mockUserSvc.SearchQueries[0] != "test" {
		t.Errorf("expected query 'test', got '%s'", mockUserSvc.SearchQueries[0])
	}
}

func TestSearchUsersWithLimit(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	for i := 0; i < 20; i++ {
		mockUserSvc.AddUser(&user.User{
			Email:    "user@example.com",
			Username: "user",
		})
	}

	router.GET("/users/search", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.SearchUsers(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/search?q=test&limit=5", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestSearchUsersMissingQuery(t *testing.T) {
	handler, _, _, router := setupUserHandlerTest()

	router.GET("/users/search", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.SearchUsers(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/search", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSearchUsersQueryTooShort(t *testing.T) {
	handler, _, _, router := setupUserHandlerTest()

	router.GET("/users/search", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.SearchUsers(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/search?q=a", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSearchUsersError(t *testing.T) {
	handler, mockUserSvc, _, router := setupUserHandlerTest()

	mockUserSvc.SearchErr = userService.ErrUserNotFound

	router.GET("/users/search", func(c *gin.Context) {
		setUserContext(c, 1)
		handler.SearchUsers(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/search?q=test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}
