package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func AuditMiddleware(config *AuditConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, path := range config.SkipPaths {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		for _, method := range config.SkipMethods {
			if c.Request.Method == method {
				c.Next()
				return
			}
		}

		startTime := time.Now()

		var requestBody []byte
		if config.CaptureBody && c.Request.Body != nil {
			requestBody, _ = io.ReadAll(io.LimitReader(c.Request.Body, config.MaxBodySize))
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		rw := &responseWriter{ResponseWriter: c.Writer, statusCode: http.StatusOK}
		c.Writer = rw

		c.Next()

		duration := time.Since(startTime).Milliseconds()

		log := buildAuditLog(c, config, requestBody, rw.statusCode, duration)
		if log != nil {
			go func() {
				if err := config.DB.Create(log).Error; err != nil {
				}
			}()
		}
	}
}

type responseWriter struct {
	gin.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func buildAuditLog(c *gin.Context, config *AuditConfig, body []byte, statusCode int, duration int64) *AuditLog {
	action, resourceType, resourceID := parseAction(c.Request.Method, c.Request.URL.Path)
	if action == "" {
		return nil
	}

	tc := GetTenant(c.Request.Context())

	log := &AuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ActorType:    "user",
		StatusCode:   statusCode,
		Duration:     duration,
		CreatedAt:    time.Now(),
	}

	if tc != nil && tc.OrganizationID > 0 {
		log.OrganizationID = &tc.OrganizationID
		log.ActorID = &tc.UserID
	}

	ip := c.ClientIP()
	if ip != "" {
		log.IPAddress = &ip
	}

	ua := c.Request.UserAgent()
	if ua != "" {
		log.UserAgent = &ua
	}

	details := buildDetails(c, config, body)
	if len(details) > 0 {
		detailsJSON, _ := json.Marshal(details)
		log.Details = detailsJSON
	}

	return log
}

func parseAction(method, path string) (action string, resourceType string, resourceID *int64) {
	path = strings.TrimPrefix(path, "/api/v1")
	path = strings.TrimPrefix(path, "/api")

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "", "", nil
	}

	resourceType = parts[0]

	switch method {
	case "POST":
		action = resourceType + ".created"
	case "PUT", "PATCH":
		action = resourceType + ".updated"
	case "DELETE":
		action = resourceType + ".deleted"
	default:
		return "", "", nil
	}

	if len(parts) > 1 {
		var id int64
		if _, err := json.Number(parts[1]).Int64(); err == nil {
			id, _ = json.Number(parts[1]).Int64()
			resourceID = &id
		}
	}

	switch {
	case strings.Contains(path, "/terminate"):
		action = "pods.terminated"
	case strings.Contains(path, "/archive"):
		action = "channels.archived"
	case strings.Contains(path, "/unarchive"):
		action = "channels.unarchived"
	case strings.Contains(path, "/join"):
		action = "channels.joined"
	case strings.Contains(path, "/leave"):
		action = "channels.left"
	case strings.Contains(path, "/register"):
		action = "users.registered"
		resourceType = "users"
	case strings.Contains(path, "/login"):
		action = "users.logged_in"
		resourceType = "users"
	case strings.Contains(path, "/oauth"):
		action = "users.oauth_login"
		resourceType = "users"
	}

	return action, resourceType, resourceID
}

func buildDetails(c *gin.Context, config *AuditConfig, body []byte) map[string]interface{} {
	details := make(map[string]interface{})

	if len(c.Request.URL.Query()) > 0 {
		query := make(map[string]string)
		for key, values := range c.Request.URL.Query() {
			if len(values) > 0 {
				query[key] = values[0]
			}
		}
		details["query"] = query
	}

	if len(body) > 0 && config.CaptureBody {
		var bodyData map[string]interface{}
		if err := json.Unmarshal(body, &bodyData); err == nil {
			sanitizedBody := sanitizeBody(bodyData, config.SensitiveFields)
			details["body"] = sanitizedBody
		}
	}

	return details
}

func sanitizeBody(body map[string]interface{}, sensitiveFields []string) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range body {
		isSensitive := false
		keyLower := strings.ToLower(key)
		for _, field := range sensitiveFields {
			if strings.Contains(keyLower, field) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			result[key] = "[REDACTED]"
		} else if nested, ok := value.(map[string]interface{}); ok {
			result[key] = sanitizeBody(nested, sensitiveFields)
		} else {
			result[key] = value
		}
	}
	return result
}
