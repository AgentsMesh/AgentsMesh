package v1

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	fileservice "github.com/anthropics/agentsmesh/backend/internal/service/file"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// FileServiceInterface defines the interface for file service operations
type FileServiceInterface interface {
	RequestPresignedUpload(ctx context.Context, req *fileservice.PresignUploadRequest) (*fileservice.PresignUploadResponse, error)
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

// presignUploadRequest is the JSON body for presign upload
type presignUploadRequest struct {
	Filename    string `json:"filename" binding:"required"`
	ContentType string `json:"content_type" binding:"required"`
	Size        int64  `json:"size" binding:"required,gt=0"`
}

// PresignUpload returns presigned PUT and GET URLs for direct-to-S3 upload
// POST /api/v1/orgs/:slug/files/presign
func (h *FileHandler) PresignUpload(c *gin.Context) {
	if h.fileService == nil {
		apierr.InternalError(c, "Storage not configured")
		return
	}

	var req presignUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.BadRequest(c, apierr.VALIDATION_FAILED, "Invalid request: "+err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	resp, err := h.fileService.RequestPresignedUpload(c.Request.Context(), &fileservice.PresignUploadRequest{
		OrganizationID: tenant.OrganizationID,
		FileName:       req.Filename,
		ContentType:    req.ContentType,
		Size:           req.Size,
	})
	if err != nil {
		switch {
		case errors.Is(err, fileservice.ErrFileTooLarge):
			apierr.PayloadTooLarge(c, err.Error())
		case errors.Is(err, fileservice.ErrInvalidFileType):
			slog.Warn("Presign upload rejected: invalid type",
				"content_type", req.ContentType,
				"filename", req.Filename,
				"org_id", tenant.OrganizationID,
			)
			apierr.UnsupportedMediaType(c, err.Error())
		case errors.Is(err, fileservice.ErrStorageError):
			slog.Error("Presign upload failed: storage error",
				"error", err,
				"filename", req.Filename,
				"org_id", tenant.OrganizationID,
			)
			apierr.InternalError(c, "Failed to generate upload URL")
		default:
			slog.Error("Presign upload failed",
				"error", err,
				"filename", req.Filename,
				"org_id", tenant.OrganizationID,
			)
			apierr.InternalError(c, "Failed to generate upload URL")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"put_url": resp.PutURL,
		"get_url": resp.GetURL,
	})
}
