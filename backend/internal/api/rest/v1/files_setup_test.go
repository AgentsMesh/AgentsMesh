package v1

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	fileservice "github.com/anthropics/agentsmesh/backend/internal/service/file"
	"github.com/gin-gonic/gin"
)

func setupFileHandlerTest() (*FileHandler, *fileservice.MockService, *gin.Engine) {
	mockSvc := fileservice.NewMockService()
	handler := NewFileHandler(mockSvc)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	return handler, mockSvc, router
}

func setFileTenantContext(c *gin.Context, orgID int64, userID int64) {
	tc := &middleware.TenantContext{
		OrganizationID:   orgID,
		OrganizationSlug: "test-org",
		UserID:           userID,
		UserRole:         "owner",
	}
	c.Set("tenant", tc)
}

func TestNewFileHandler(t *testing.T) {
	handler, _, _ := setupFileHandlerTest()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}
