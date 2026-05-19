package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const LatestRunnerVersion = "0.29.0"

// Source: release-staging/checksums.txt produced by .github/workflows/release.yml.
// MUST be updated with LatestRunnerVersion — stale digests cause install_binary ChecksumMismatch.
// NEVER commit empty values: missing entry means desktop client downloads without integrity check.
var LatestRunnerSha256 = map[string]string{
	// TODO: populate from release pipeline. Empty map = verification not yet available,
	// desktop client warns + proceeds rather than blocking onboarding.
}

type RunnerReleaseResponse struct {
	Version string            `json:"version"`
	Sha256  map[string]string `json:"sha256"`
}

func GetLatestRunnerRelease(c *gin.Context) {
	c.JSON(http.StatusOK, RunnerReleaseResponse{
		Version: LatestRunnerVersion,
		Sha256:  LatestRunnerSha256,
	})
}

func RegisterRunnerReleaseRoutes(r *gin.RouterGroup) {
	r.GET("/runners/latest-release", GetLatestRunnerRelease)
}
