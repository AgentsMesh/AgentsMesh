package i18n

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func LocaleMiddleware(i *I18n) gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := detectLocale(c, i)
		ctx := WithLocale(c.Request.Context(), locale)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func detectLocale(c *gin.Context, i *I18n) string {
	if locale := c.GetHeader("X-Locale"); locale != "" {
		if isValidLocale(i, locale) {
			return locale
		}
	}

	acceptLang := c.GetHeader("Accept-Language")
	if acceptLang != "" {
		locale := parseAcceptLanguage(acceptLang, i)
		if locale != "" {
			return locale
		}
	}

	if locale := c.Query("locale"); locale != "" {
		if isValidLocale(i, locale) {
			return locale
		}
	}

	return i.GetLocale()
}

func parseAcceptLanguage(acceptLang string, i *I18n) string {
	parts := strings.Split(acceptLang, ",")
	for _, part := range parts {
		lang := strings.TrimSpace(strings.Split(part, ";")[0])

		if isValidLocale(i, lang) {
			return lang
		}

		if idx := strings.Index(lang, "-"); idx > 0 {
			baseLang := lang[:idx]
			if isValidLocale(i, baseLang) {
				return baseLang
			}
		}

		availableLocales := i.GetAvailableLocales()
		for _, loc := range availableLocales {
			if strings.HasPrefix(loc.Code, lang+"-") || loc.Code == lang {
				return loc.Code
			}
		}
	}

	return ""
}

func isValidLocale(i *I18n, locale string) bool {
	for _, loc := range i.GetAvailableLocales() {
		if loc.Code == locale {
			return true
		}
	}
	return false
}

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
