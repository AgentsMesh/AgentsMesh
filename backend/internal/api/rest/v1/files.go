package v1

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/file"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	fileservice "github.com/anthropics/agentsmesh/backend/internal/service/file"
	"github.com/gin-gonic/gin"
)

// FileServiceInterface defines the interface for file service operations
type FileServiceInterface interface {
	Upload(ctx context.Context, req *fileservice.UploadRequest) (*fileservice.UploadResponse, error)
	GetByID(ctx context.Context, id int64, orgID int64) (*file.File, error)
	GetURL(ctx context.Context, id int64, orgID int64, expiry time.Duration) (string, error)
	Delete(ctx context.Context, id int64, orgID int64) error
}

// FileHandler handles file-related requests
type FileHandler struct {
	fileService FileServiceInterface
}

// NewFileHandler creates a new file handler
func NewFileHandler(fileService FileServiceInterface) *FileHandler {
	return &FileHandler{
		fileService: fileService,
	}
}

// UploadFile handles file upload
// POST /api/v1/orgs/:slug/files/upload
func (h *FileHandler) UploadFile(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	// Get file from multipart form
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	// Get content type
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Upload file
	resp, err := h.fileService.Upload(c.Request.Context(), &fileservice.UploadRequest{
		OrganizationID: tenant.OrganizationID,
		UploaderID:     tenant.UserID,
		FileName:       fileHeader.Filename,
		ContentType:    contentType,
		Size:           fileHeader.Size,
		Reader:         file,
	})
	if err != nil {
		switch err {
		case fileservice.ErrFileTooLarge:
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
		case fileservice.ErrInvalidFileType:
			c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteFile handles file deletion
// DELETE /api/v1/orgs/:slug/files/:id
func (h *FileHandler) DeleteFile(c *gin.Context) {
	tenant := middleware.GetTenant(c)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID"})
		return
	}

	if err := h.fileService.Delete(c.Request.Context(), id, tenant.OrganizationID); err != nil {
		if err == fileservice.ErrFileNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}
