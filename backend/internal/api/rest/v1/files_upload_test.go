package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	fileservice "github.com/anthropics/agentsmesh/backend/internal/service/file"
	"github.com/gin-gonic/gin"
)

func TestPresignUpload_Success(t *testing.T) {
	handler, _, router := setupFileHandlerTest()

	router.POST("/files/presign", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.PresignUpload(c)
	})

	body := `{"filename":"test.png","content_type":"image/png","size":1024}`
	req := httptest.NewRequest(http.MethodPost, "/files/presign", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["put_url"] == nil || resp["put_url"] == "" {
		t.Error("expected put_url in response")
	}
	if resp["get_url"] == nil || resp["get_url"] == "" {
		t.Error("expected get_url in response")
	}
}

func TestPresignUpload_InvalidBody(t *testing.T) {
	handler, _, router := setupFileHandlerTest()

	router.POST("/files/presign", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.PresignUpload(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/files/presign", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPresignUpload_FileTooLarge(t *testing.T) {
	handler, mockSvc, router := setupFileHandlerTest()
	mockSvc.SetPresignErr(fileservice.ErrFileTooLarge)

	router.POST("/files/presign", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.PresignUpload(c)
	})

	body := `{"filename":"large.png","content_type":"image/png","size":104857600}`
	req := httptest.NewRequest(http.MethodPost, "/files/presign", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPresignUpload_InvalidFileType(t *testing.T) {
	handler, mockSvc, router := setupFileHandlerTest()
	mockSvc.SetPresignErr(fileservice.ErrInvalidFileType)

	router.POST("/files/presign", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.PresignUpload(c)
	})

	body := `{"filename":"test.exe","content_type":"application/x-executable","size":1024}`
	req := httptest.NewRequest(http.MethodPost, "/files/presign", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected status 415, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPresignUpload_StorageError(t *testing.T) {
	handler, mockSvc, router := setupFileHandlerTest()
	mockSvc.SetPresignErr(fileservice.ErrStorageError)

	router.POST("/files/presign", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.PresignUpload(c)
	})

	body := `{"filename":"test.png","content_type":"image/png","size":1024}`
	req := httptest.NewRequest(http.MethodPost, "/files/presign", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d: %s", w.Code, w.Body.String())
	}
}
