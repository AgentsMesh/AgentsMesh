package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	userService "github.com/AgentsMesh/AgentsMesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

// ===========================================
// Search Tests
// ===========================================

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

