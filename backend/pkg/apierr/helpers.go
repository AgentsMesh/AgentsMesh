package apierr

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Forbidden(c *gin.Context, code, message string) {
	Respond(c, http.StatusForbidden, code, message)
}

func ForbiddenAccess(c *gin.Context) {
	Forbidden(c, ACCESS_DENIED, "Access denied")
}

func ForbiddenAdmin(c *gin.Context) {
	Forbidden(c, ADMIN_REQUIRED, "Admin permission required")
}

func ForbiddenOwner(c *gin.Context) {
	Forbidden(c, OWNER_REQUIRED, "Owner permission required")
}

func ForbiddenDisabled(c *gin.Context) {
	Forbidden(c, ACCOUNT_DISABLED, "Account is disabled")
}

func BadRequest(c *gin.Context, code, message string) {
	Respond(c, http.StatusBadRequest, code, message)
}

func ValidationError(c *gin.Context, message string) {
	BadRequest(c, VALIDATION_FAILED, message)
}

func InvalidInput(c *gin.Context, message string) {
	BadRequest(c, INVALID_INPUT, message)
}

func Unauthorized(c *gin.Context, code, message string) {
	Respond(c, http.StatusUnauthorized, code, message)
}

func PaymentRequired(c *gin.Context, code, message string) {
	Respond(c, http.StatusPaymentRequired, code, message)
}

func NotFound(c *gin.Context, code, message string) {
	Respond(c, http.StatusNotFound, code, message)
}

func ResourceNotFound(c *gin.Context, message string) {
	NotFound(c, RESOURCE_NOT_FOUND, message)
}

func Conflict(c *gin.Context, code, message string) {
	Respond(c, http.StatusConflict, code, message)
}

func InternalError(c *gin.Context, message string) {
	Respond(c, http.StatusInternalServerError, INTERNAL_ERROR, message)
}

func ServiceUnavailable(c *gin.Context, code, message string) {
	Respond(c, http.StatusServiceUnavailable, code, message)
}

func NotImplemented(c *gin.Context, message string) {
	Respond(c, http.StatusNotImplemented, NOT_IMPLEMENTED, message)
}

func TooManyRequests(c *gin.Context, message string) {
	Respond(c, http.StatusTooManyRequests, RATE_LIMITED, message)
}

func CapacityExceeded(c *gin.Context, message string) {
	Respond(c, http.StatusTooManyRequests, CAPACITY_EXCEEDED, message)
}

func PayloadTooLarge(c *gin.Context, message string) {
	Respond(c, http.StatusRequestEntityTooLarge, PAYLOAD_TOO_LARGE, message)
}

func UnsupportedMediaType(c *gin.Context, message string) {
	Respond(c, http.StatusUnsupportedMediaType, UNSUPPORTED_MEDIA, message)
}

func PaymentRequiredWithExtra(c *gin.Context, code, message string, extra gin.H) {
	RespondWithExtra(c, http.StatusPaymentRequired, code, message, extra)
}

func AbortForbidden(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

func AbortUnauthorized(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

func AbortBadRequest(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

func AbortNotFound(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{
		Error: message,
		Code:  code,
	})
}
