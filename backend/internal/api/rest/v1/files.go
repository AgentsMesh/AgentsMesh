package v1

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/file"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/middleware"
	fileservice "github.com/AgentsMesh/AgentsMesh/backend/internal/service/file"
	"github.com/AgentsMesh/AgentsMesh/backend/pkg/apierr"
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
	slog.Info(">>> UploadFile handler entered")

	if h.fileService == nil {
		slog.Error("fileService is nil!")
		apierr.InternalError(c, "Storage not configured")
		return
	}

	tenant := middleware.GetTenant(c)
	slog.Info("File upload request received",
		"org_id", tenant.OrganizationID,
		"user_id", tenant.UserID,
	)

	// Get file from multipart form
	fileHeader, err := c.FormFile("file")
	if err != nil {
		slog.Warn("File upload failed: no file provided", "error", err)
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "No file provided")
		return
	}
	slog.Info("File received",
		"filename", fileHeader.Filename,
		"size", fileHeader.Size,
		"content_type", fileHeader.Header.Get("Content-Type"),
	)

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Failed to open file")
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
		switch {
		case errors.Is(err, fileservice.ErrFileTooLarge):
			apierr.PayloadTooLarge(c, err.Error())
		case errors.Is(err, fileservice.ErrInvalidFileType):
			slog.Warn("File upload rejected: invalid type",
				"content_type", contentType,
				"filename", fileHeader.Filename,
				"org_id", tenant.OrganizationID,
			)
			apierr.UnsupportedMediaType(c, err.Error())
		case errors.Is(err, fileservice.ErrStorageError):
			slog.Error("File upload failed: storage error",
				"error", err,
				"filename", fileHeader.Filename,
				"size", fileHeader.Size,
				"content_type", contentType,
				"org_id", tenant.OrganizationID,
			)
			apierr.InternalError(c, "Failed to upload file")
		default:
			slog.Error("File upload failed",
				"error", err,
				"filename", fileHeader.Filename,
				"size", fileHeader.Size,
				"content_type", contentType,
				"org_id", tenant.OrganizationID,
			)
			apierr.InternalError(c, "Failed to upload file")
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
		apierr.InvalidInput(c, "Invalid file ID")
		return
	}

	if err := h.fileService.Delete(c.Request.Context(), id, tenant.OrganizationID); err != nil {
		if err == fileservice.ErrFileNotFound {
			apierr.ResourceNotFound(c, "File not found")
			return
		}
		apierr.InternalError(c, "Failed to delete file")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}
