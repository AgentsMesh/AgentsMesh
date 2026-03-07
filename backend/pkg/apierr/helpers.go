package apierr

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// --- Standard responses (for handlers) ---

// Forbidden sends a 403 Forbidden response.
func Forbidden(c *gin.Context, code, message string) {
	Respond(c, http.StatusForbidden, code, message)
}

// ForbiddenAccess sends a 403 with ACCESS_DENIED code.
func ForbiddenAccess(c *gin.Context) {
	Forbidden(c, ACCESS_DENIED, "Access denied")
}

// ForbiddenAdmin sends a 403 with ADMIN_REQUIRED code.
func ForbiddenAdmin(c *gin.Context) {
	Forbidden(c, ADMIN_REQUIRED, "Admin permission required")
}

// ForbiddenOwner sends a 403 with OWNER_REQUIRED code.
func ForbiddenOwner(c *gin.Context) {
	Forbidden(c, OWNER_REQUIRED, "Owner permission required")
}

// ForbiddenDisabled sends a 403 with ACCOUNT_DISABLED code.
func ForbiddenDisabled(c *gin.Context) {
	Forbidden(c, ACCOUNT_DISABLED, "Account is disabled")
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(c *gin.Context, code, message string) {
	Respond(c, http.StatusBadRequest, code, message)
}

// ValidationError sends a 400 with VALIDATION_FAILED code.
func ValidationError(c *gin.Context, message string) {
	BadRequest(c, VALIDATION_FAILED, message)
}

// InvalidInput sends a 400 with INVALID_INPUT code.
func InvalidInput(c *gin.Context, message string) {
	BadRequest(c, INVALID_INPUT, message)
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(c *gin.Context, code, message string) {
	Respond(c, http.StatusUnauthorized, code, message)
}

// PaymentRequired sends a 402 Payment Required response.
func PaymentRequired(c *gin.Context, code, message string) {
	Respond(c, http.StatusPaymentRequired, code, message)
}

// NotFound sends a 404 Not Found response.
func NotFound(c *gin.Context, code, message string) {
	Respond(c, http.StatusNotFound, code, message)
}

// ResourceNotFound sends a 404 with RESOURCE_NOT_FOUND code.
func ResourceNotFound(c *gin.Context, message string) {
	NotFound(c, RESOURCE_NOT_FOUND, message)
}

// Conflict sends a 409 Conflict response.
func Conflict(c *gin.Context, code, message string) {
	Respond(c, http.StatusConflict, code, message)
}

// InternalError sends a 500 Internal Server Error response.
func InternalError(c *gin.Context, message string) {
	Respond(c, http.StatusInternalServerError, INTERNAL_ERROR, message)
}

// ServiceUnavailable sends a 503 Service Unavailable response.
func ServiceUnavailable(c *gin.Context, code, message string) {
	Respond(c, http.StatusServiceUnavailable, code, message)
}

// NotImplemented sends a 501 Not Implemented response.
func NotImplemented(c *gin.Context, message string) {
	Respond(c, http.StatusNotImplemented, NOT_IMPLEMENTED, message)
}

// TooManyRequests sends a 429 Too Many Requests response.
func TooManyRequests(c *gin.Context, message string) {
	Respond(c, http.StatusTooManyRequests, RATE_LIMITED, message)
}

// CapacityExceeded sends a 429 Too Many Requests response with CAPACITY_EXCEEDED code.
func CapacityExceeded(c *gin.Context, message string) {
	Respond(c, http.StatusTooManyRequests, CAPACITY_EXCEEDED, message)
}

// PayloadTooLarge sends a 413 Request Entity Too Large response.
func PayloadTooLarge(c *gin.Context, message string) {
	Respond(c, http.StatusRequestEntityTooLarge, PAYLOAD_TOO_LARGE, message)
}

// UnsupportedMediaType sends a 415 Unsupported Media Type response.
func UnsupportedMediaType(c *gin.Context, message string) {
	Respond(c, http.StatusUnsupportedMediaType, UNSUPPORTED_MEDIA, message)
}

// PaymentRequiredWithExtra sends a 402 with extra fields.
func PaymentRequiredWithExtra(c *gin.Context, code, message string, extra gin.H) {
	RespondWithExtra(c, http.StatusPaymentRequired, code, message, extra)
}

// --- Abort responses (for middleware) ---

// AbortForbidden aborts with a 403 Forbidden response.
func AbortForbidden(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// AbortUnauthorized aborts with a 401 Unauthorized response.
func AbortUnauthorized(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// AbortBadRequest aborts with a 400 Bad Request response.
func AbortBadRequest(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// AbortNotFound aborts with a 404 Not Found response.
func AbortNotFound(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{
		Error: message,
		Code:  code,
	})
}
