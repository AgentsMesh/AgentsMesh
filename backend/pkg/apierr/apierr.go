package apierr

import (
	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func Respond(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

func RespondWithExtra(c *gin.Context, status int, code, message string, extra gin.H) {
	resp := gin.H{
		"error": message,
		"code":  code,
	}
	for k, v := range extra {
		resp[k] = v
	}
	c.JSON(status, resp)
}
