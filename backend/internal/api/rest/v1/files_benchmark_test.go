package v1

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func BenchmarkPresignUpload(b *testing.B) {
	handler, _, router := setupFileHandlerTest()

	router.POST("/files/presign", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.PresignUpload(c)
	})

	body := `{"filename":"test.png","content_type":"image/png","size":1024}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/files/presign", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
