package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/file"
	fileservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/file"
	"github.com/gin-gonic/gin"
)

func TestDeleteFile_Success(t *testing.T) {
	handler, mockSvc, router := setupFileHandlerTest()

	// Add a file
	mockSvc.AddFile(&file.File{
		ID:             1,
		OrganizationID: 1,
		UploaderID:     100,
		OriginalName:   "test.png",
		StorageKey:     "orgs/1/files/test.png",
		MimeType:       "image/png",
		Size:           1024,
		CreatedAt:      time.Now(),
	})

	router.DELETE("/files/:id", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.DeleteFile(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/files/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["message"] != "File deleted successfully" {
		t.Errorf("expected message 'File deleted successfully', got %v", resp["message"])
	}
}

func TestDeleteFile_InvalidID(t *testing.T) {
	handler, _, router := setupFileHandlerTest()

	router.DELETE("/files/:id", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.DeleteFile(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/files/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFile_NotFound(t *testing.T) {
	handler, _, router := setupFileHandlerTest()

	router.DELETE("/files/:id", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.DeleteFile(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/files/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFile_WrongOrganization(t *testing.T) {
	handler, mockSvc, router := setupFileHandlerTest()

	// Add a file for org 2
	mockSvc.AddFile(&file.File{
		ID:             1,
		OrganizationID: 2, // Different org
		UploaderID:     100,
		OriginalName:   "test.png",
		StorageKey:     "orgs/2/files/test.png",
		MimeType:       "image/png",
		Size:           1024,
		CreatedAt:      time.Now(),
	})

	router.DELETE("/files/:id", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100) // User in org 1
		handler.DeleteFile(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/files/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteFile_StorageError(t *testing.T) {
	handler, mockSvc, router := setupFileHandlerTest()

	// Add a file
	mockSvc.AddFile(&file.File{
		ID:             1,
		OrganizationID: 1,
		UploaderID:     100,
		OriginalName:   "test.png",
		StorageKey:     "orgs/1/files/test.png",
		MimeType:       "image/png",
		Size:           1024,
		CreatedAt:      time.Now(),
	})

	// Set storage error
	mockSvc.SetDeleteErr(fileservice.ErrStorageError)

	router.DELETE("/files/:id", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.DeleteFile(c)
	})

	req := httptest.NewRequest(http.MethodDelete, "/files/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}
}
