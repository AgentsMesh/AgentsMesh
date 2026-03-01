package v1

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	fileservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/file"
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

func createMultipartRequest(fieldName, fileName string, content []byte, contentType string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		return nil, err
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/files/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

func TestNewFileHandler(t *testing.T) {
	handler, _, _ := setupFileHandlerTest()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}
