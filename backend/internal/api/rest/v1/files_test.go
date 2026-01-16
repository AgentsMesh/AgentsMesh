package v1

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/file"
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

// Benchmark tests
func BenchmarkUploadFile(b *testing.B) {
	handler, _, router := setupFileHandlerTest()

	router.POST("/files/upload", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.UploadFile(c)
	})

	content := []byte("benchmark test content")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := createMultipartRequest("file", "test.png", content, "image/png")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkDeleteFile(b *testing.B) {
	handler, mockSvc, router := setupFileHandlerTest()

	router.DELETE("/files/:id", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.DeleteFile(c)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Add a file for each iteration
		mockSvc.AddFile(&file.File{
			ID:             int64(i + 1),
			OrganizationID: 1,
			UploaderID:     100,
			OriginalName:   "test.png",
			StorageKey:     "orgs/1/files/test.png",
			MimeType:       "image/png",
			Size:           1024,
			CreatedAt:      time.Now(),
		})
		b.StartTimer()

		req := httptest.NewRequest(http.MethodDelete, "/files/"+string(rune(i+1)), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
