package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	fileservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/file"
	"github.com/gin-gonic/gin"
)

func TestUploadFile_Success(t *testing.T) {
	handler, _, router := setupFileHandlerTest()

	router.POST("/files/upload", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.UploadFile(c)
	})

	content := []byte("test image content")
	req, err := createMultipartRequest("file", "test.png", content, "image/png")
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["url"] == nil || resp["url"] == "" {
		t.Error("expected url in response")
	}

	fileData := resp["file"].(map[string]interface{})
	if fileData["original_name"] != "test.png" {
		t.Errorf("expected original_name 'test.png', got %v", fileData["original_name"])
	}
}

func TestUploadFile_NoFile(t *testing.T) {
	handler, _, router := setupFileHandlerTest()

	router.POST("/files/upload", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.UploadFile(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/files/upload", nil)
	req.Header.Set("Content-Type", "multipart/form-data")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadFile_FileTooLarge(t *testing.T) {
	handler, mockSvc, router := setupFileHandlerTest()
	mockSvc.SetUploadErr(fileservice.ErrFileTooLarge)

	router.POST("/files/upload", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.UploadFile(c)
	})

	content := []byte("test content")
	req, err := createMultipartRequest("file", "large.png", content, "image/png")
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadFile_InvalidFileType(t *testing.T) {
	handler, mockSvc, router := setupFileHandlerTest()
	mockSvc.SetUploadErr(fileservice.ErrInvalidFileType)

	router.POST("/files/upload", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.UploadFile(c)
	})

	content := []byte("test content")
	req, err := createMultipartRequest("file", "test.exe", content, "application/x-executable")
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected status 415, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadFile_StorageError(t *testing.T) {
	handler, mockSvc, router := setupFileHandlerTest()
	mockSvc.SetUploadErr(fileservice.ErrStorageError)

	router.POST("/files/upload", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.UploadFile(c)
	})

	content := []byte("test content")
	req, err := createMultipartRequest("file", "test.png", content, "image/png")
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUploadFile_MultipleFiles(t *testing.T) {
	handler, mockSvc, router := setupFileHandlerTest()

	router.POST("/files/upload", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.UploadFile(c)
	})

	// Upload first file
	content1 := []byte("test image 1")
	req1, _ := createMultipartRequest("file", "test1.png", content1, "image/png")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("first upload: expected status 200, got %d", w1.Code)
	}

	// Upload second file
	content2 := []byte("test image 2")
	req2, _ := createMultipartRequest("file", "test2.png", content2, "image/png")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("second upload: expected status 200, got %d", w2.Code)
	}

	// Verify both files were stored
	if mockSvc.FileCount() != 2 {
		t.Errorf("expected 2 files, got %d", mockSvc.FileCount())
	}
}
