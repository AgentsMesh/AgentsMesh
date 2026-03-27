package apierr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err, "failed to parse response JSON")
	return resp
}

// --- Respond ---

func TestRespond(t *testing.T) {
	c, w := setupTestContext()

	Respond(c, http.StatusBadRequest, VALIDATION_FAILED, "field is required")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "field is required", resp["error"])
	assert.Equal(t, "VALIDATION_FAILED", resp["code"])
	assert.Len(t, resp, 2, "response should only contain error and code fields")
}

func TestRespond_StructFields(t *testing.T) {
	c, w := setupTestContext()

	Respond(c, http.StatusForbidden, ACCESS_DENIED, "not allowed")

	var resp ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "not allowed", resp.Error)
	assert.Equal(t, "ACCESS_DENIED", resp.Code)
}

// --- RespondWithExtra ---

func TestRespondWithExtra(t *testing.T) {
	c, w := setupTestContext()

	RespondWithExtra(c, http.StatusPaymentRequired, CONCURRENT_POD_QUOTA_EXCEEDED, "quota exceeded", gin.H{
		"current_count": 5,
		"max_count":     5,
	})

	assert.Equal(t, http.StatusPaymentRequired, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "quota exceeded", resp["error"])
	assert.Equal(t, "CONCURRENT_POD_QUOTA_EXCEEDED", resp["code"])
	assert.Equal(t, float64(5), resp["current_count"])
	assert.Equal(t, float64(5), resp["max_count"])
	assert.Len(t, resp, 4)
}

func TestRespondWithExtra_EmptyExtra(t *testing.T) {
	c, w := setupTestContext()

	RespondWithExtra(c, http.StatusBadRequest, INVALID_INPUT, "bad input", gin.H{})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "bad input", resp["error"])
	assert.Equal(t, "INVALID_INPUT", resp["code"])
	assert.Len(t, resp, 2)
}

// --- Helper functions ---

func TestForbidden(t *testing.T) {
	c, w := setupTestContext()
	Forbidden(c, ACCESS_DENIED, "custom forbidden")
	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "ACCESS_DENIED", resp["code"])
	assert.Equal(t, "custom forbidden", resp["error"])
}

func TestForbiddenAccess(t *testing.T) {
	c, w := setupTestContext()
	ForbiddenAccess(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "ACCESS_DENIED", resp["code"])
	assert.Equal(t, "Access denied", resp["error"])
}

func TestForbiddenAdmin(t *testing.T) {
	c, w := setupTestContext()
	ForbiddenAdmin(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "ADMIN_REQUIRED", resp["code"])
	assert.Equal(t, "Admin permission required", resp["error"])
}

func TestForbiddenOwner(t *testing.T) {
	c, w := setupTestContext()
	ForbiddenOwner(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "OWNER_REQUIRED", resp["code"])
	assert.Equal(t, "Owner permission required", resp["error"])
}

func TestForbiddenDisabled(t *testing.T) {
	c, w := setupTestContext()
	ForbiddenDisabled(c)
	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "ACCOUNT_DISABLED", resp["code"])
	assert.Equal(t, "Account is disabled", resp["error"])
}

func TestBadRequest(t *testing.T) {
	c, w := setupTestContext()
	BadRequest(c, MISSING_REQUIRED, "name is required")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "MISSING_REQUIRED", resp["code"])
	assert.Equal(t, "name is required", resp["error"])
}

func TestValidationError(t *testing.T) {
	c, w := setupTestContext()
	ValidationError(c, "email format invalid")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "VALIDATION_FAILED", resp["code"])
	assert.Equal(t, "email format invalid", resp["error"])
}

func TestInvalidInput(t *testing.T) {
	c, w := setupTestContext()
	InvalidInput(c, "unexpected field")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "INVALID_INPUT", resp["code"])
	assert.Equal(t, "unexpected field", resp["error"])
}

func TestUnauthorized(t *testing.T) {
	c, w := setupTestContext()
	Unauthorized(c, AUTH_REQUIRED, "authentication required")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "AUTH_REQUIRED", resp["code"])
	assert.Equal(t, "authentication required", resp["error"])
}

func TestPaymentRequired(t *testing.T) {
	c, w := setupTestContext()
	PaymentRequired(c, SUBSCRIPTION_FROZEN, "subscription expired")
	assert.Equal(t, http.StatusPaymentRequired, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "SUBSCRIPTION_FROZEN", resp["code"])
	assert.Equal(t, "subscription expired", resp["error"])
}

func TestNotFound(t *testing.T) {
	c, w := setupTestContext()
	NotFound(c, SOURCE_POD_NOT_FOUND, "pod not found")
	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "SOURCE_POD_NOT_FOUND", resp["code"])
	assert.Equal(t, "pod not found", resp["error"])
}

func TestResourceNotFound(t *testing.T) {
	c, w := setupTestContext()
	ResourceNotFound(c, "user not found")
	assert.Equal(t, http.StatusNotFound, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "RESOURCE_NOT_FOUND", resp["code"])
	assert.Equal(t, "user not found", resp["error"])
}

func TestConflict(t *testing.T) {
	c, w := setupTestContext()
	Conflict(c, ALREADY_EXISTS, "resource already exists")
	assert.Equal(t, http.StatusConflict, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "ALREADY_EXISTS", resp["code"])
	assert.Equal(t, "resource already exists", resp["error"])
}

func TestInternalError(t *testing.T) {
	c, w := setupTestContext()
	InternalError(c, "unexpected failure")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "INTERNAL_ERROR", resp["code"])
	assert.Equal(t, "unexpected failure", resp["error"])
}

func TestServiceUnavailable(t *testing.T) {
	c, w := setupTestContext()
	ServiceUnavailable(c, NO_AVAILABLE_RUNNER, "no runner available")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	resp := parseResponse(t, w)
	assert.Equal(t, "NO_AVAILABLE_RUNNER", resp["code"])
	assert.Equal(t, "no runner available", resp["error"])
}

// --- Abort functions ---

func TestAbortForbidden(t *testing.T) {
	c, w := setupTestContext()
	AbortForbidden(c, ACCESS_DENIED, "forbidden")
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.True(t, c.IsAborted())
	resp := parseResponse(t, w)
	assert.Equal(t, "ACCESS_DENIED", resp["code"])
	assert.Equal(t, "forbidden", resp["error"])
}

func TestAbortUnauthorized(t *testing.T) {
	c, w := setupTestContext()
	AbortUnauthorized(c, INVALID_TOKEN, "invalid token")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, c.IsAborted())
	resp := parseResponse(t, w)
	assert.Equal(t, "INVALID_TOKEN", resp["code"])
	assert.Equal(t, "invalid token", resp["error"])
}

func TestAbortBadRequest(t *testing.T) {
	c, w := setupTestContext()
	AbortBadRequest(c, VALIDATION_FAILED, "bad request")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.True(t, c.IsAborted())
	resp := parseResponse(t, w)
	assert.Equal(t, "VALIDATION_FAILED", resp["code"])
	assert.Equal(t, "bad request", resp["error"])
}

func TestAbortNotFound(t *testing.T) {
	c, w := setupTestContext()
	AbortNotFound(c, RESOURCE_NOT_FOUND, "not found")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.True(t, c.IsAborted())
	resp := parseResponse(t, w)
	assert.Equal(t, "RESOURCE_NOT_FOUND", resp["code"])
	assert.Equal(t, "not found", resp["error"])
}

// --- Error code constants ---

func TestErrorCodeConstants(t *testing.T) {
	// Verify key constants match their string values (guards against typos)
	codes := map[string]string{
		"AUTH_REQUIRED":                AUTH_REQUIRED,
		"INVALID_TOKEN":               INVALID_TOKEN,
		"TOKEN_EXPIRED":               TOKEN_EXPIRED,
		"ACCESS_DENIED":               ACCESS_DENIED,
		"ADMIN_REQUIRED":              ADMIN_REQUIRED,
		"OWNER_REQUIRED":              OWNER_REQUIRED,
		"ACCOUNT_DISABLED":            ACCOUNT_DISABLED,
		"CONCURRENT_POD_QUOTA_EXCEEDED": CONCURRENT_POD_QUOTA_EXCEEDED,
		"SUBSCRIPTION_FROZEN":         SUBSCRIPTION_FROZEN,
		"VALIDATION_FAILED":           VALIDATION_FAILED,
		"INVALID_INPUT":               INVALID_INPUT,
		"MISSING_RUNNER_ID":           MISSING_RUNNER_ID,
		"MISSING_AGENT_SLUG":         MISSING_AGENT_SLUG,
		"SOURCE_POD_NOT_FOUND":        SOURCE_POD_NOT_FOUND,
		"SOURCE_POD_ALREADY_RESUMED":  SOURCE_POD_ALREADY_RESUMED,
		"RESOURCE_NOT_FOUND":          RESOURCE_NOT_FOUND,
		"ALREADY_EXISTS":              ALREADY_EXISTS,
		"INTERNAL_ERROR":              INTERNAL_ERROR,
		"SERVICE_UNAVAILABLE":         SERVICE_UNAVAILABLE,
	}

	for expected, actual := range codes {
		assert.Equal(t, expected, actual, "constant value mismatch for %s", expected)
	}
}
