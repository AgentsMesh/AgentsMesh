package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/file"
	"github.com/gin-gonic/gin"
)

// Benchmark tests
func BenchmarkUploadFile(b *testing.B) {
	handler, _, router := setupFileHandlerTest()

	router.POST("/files/upload", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.UploadFile(c)
	})

	content := []byte("benchmark test content")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := createMultipartRequest("file", "test.png", content, "image/png")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkDeleteFile(b *testing.B) {
	handler, mockSvc, router := setupFileHandlerTest()

	router.DELETE("/files/:id", func(c *gin.Context) {
		setFileTenantContext(c, 1, 100)
		handler.DeleteFile(c)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Add a file for each iteration
		mockSvc.AddFile(&file.File{
			ID:             int64(i + 1),
			OrganizationID: 1,
			UploaderID:     100,
			OriginalName:   "test.png",
			StorageKey:     "orgs/1/files/test.png",
			MimeType:       "image/png",
			Size:           1024,
			CreatedAt:      time.Now(),
		})
		b.StartTimer()

		req := httptest.NewRequest(http.MethodDelete, "/files/"+string(rune(i+1)), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
