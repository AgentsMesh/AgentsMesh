package i18n

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// LocaleMiddleware sets the locale from request headers
func LocaleMiddleware(i *I18n) gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := detectLocale(c, i)
		ctx := WithLocale(c.Request.Context(), locale)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// detectLocale detects the locale from the request
func detectLocale(c *gin.Context, i *I18n) string {
	// 1. Check X-Locale header
	if locale := c.GetHeader("X-Locale"); locale != "" {
		if isValidLocale(i, locale) {
			return locale
		}
	}

	// 2. Check Accept-Language header
	acceptLang := c.GetHeader("Accept-Language")
	if acceptLang != "" {
		locale := parseAcceptLanguage(acceptLang, i)
		if locale != "" {
			return locale
		}
	}

	// 3. Check query parameter
	if locale := c.Query("locale"); locale != "" {
		if isValidLocale(i, locale) {
			return locale
		}
	}

	// 4. Return default locale
	return i.GetLocale()
}

// parseAcceptLanguage parses the Accept-Language header and returns the best matching locale
func parseAcceptLanguage(acceptLang string, i *I18n) string {
	// Simple parser - split by comma and find first matching locale
	parts := strings.Split(acceptLang, ",")
	for _, part := range parts {
		// Extract language code (ignore quality values)
		lang := strings.TrimSpace(strings.Split(part, ";")[0])

		// Try exact match
		if isValidLocale(i, lang) {
			return lang
		}

		// Try base language (e.g., "zh-CN" -> "zh")
		if idx := strings.Index(lang, "-"); idx > 0 {
			baseLang := lang[:idx]
			if isValidLocale(i, baseLang) {
				return baseLang
			}
		}

		// Try matching with region (e.g., "zh" -> "zh-CN")
		availableLocales := i.GetAvailableLocales()
		for _, loc := range availableLocales {
			if strings.HasPrefix(loc.Code, lang+"-") || loc.Code == lang {
				return loc.Code
			}
		}
	}

	return ""
}

// isValidLocale checks if a locale is available
func isValidLocale(i *I18n, locale string) bool {
	for _, loc := range i.GetAvailableLocales() {
		if loc.Code == locale {
			return true
		}
	}
	return false
}

// SuccessResponse returns a localized success response
func SuccessResponse(c *gin.Context, key string, data interface{}, args ...interface{}) {
	ctx := c.Request.Context()
	message := TFromContext(ctx, key, args...)

	response := gin.H{
		"message": message,
	}
	if data != nil {
		response["data"] = data
	}

	c.JSON(200, response)
}
