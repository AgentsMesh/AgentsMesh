package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
		setUserContext(c, 999)
		handler.GetCurrentUser(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
