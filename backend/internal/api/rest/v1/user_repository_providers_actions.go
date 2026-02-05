package v1

import (
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/infra/git"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/internal/service/user"
	"github.com/gin-gonic/gin"
)

// SetDefault sets a repository provider as default
// POST /api/v1/user/repository-providers/:id/default
func (h *UserRepositoryProviderHandler) SetDefault(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	err = h.userService.SetDefaultRepositoryProvider(c.Request.Context(), userID, providerID)
	if err != nil {
		if err == user.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default provider set"})
}

// TestConnection tests the connection to a repository provider
// POST /api/v1/user/repository-providers/:id/test
func (h *UserRepositoryProviderHandler) TestConnection(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	// Get provider
	provider, err := h.userService.GetRepositoryProvider(c.Request.Context(), userID, providerID)
	if err != nil {
		if err == user.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get provider"})
		return
	}

	// Get decrypted token
	token, err := h.userService.GetDecryptedProviderToken(c.Request.Context(), userID, providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt token"})
		return
	}

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No token configured for this provider"})
		return
	}

	// Create git provider and test connection
	gitProvider, err := git.NewProvider(provider.ProviderType, provider.BaseURL, token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create git provider: " + err.Error()})
		return
	}

	// Try to list projects to verify connection
	_, err = gitProvider.ListProjects(c.Request.Context(), 1, 1)
	if err != nil {
		if err == git.ErrUnauthorized {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authentication failed - token may be invalid or expired",
			})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"error":   "Connection failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Connection successful",
	})
}

// ListRepositories lists repositories accessible through a repository provider
// GET /api/v1/user/repository-providers/:id/repositories
func (h *UserRepositoryProviderHandler) ListRepositories(c *gin.Context) {
	userID := middleware.GetUserID(c)

	providerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	// Get provider
	provider, err := h.userService.GetRepositoryProvider(c.Request.Context(), userID, providerID)
	if err != nil {
		if err == user.ErrProviderNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get provider"})
		return
	}

	// Get decrypted token
	token, err := h.userService.GetDecryptedProviderToken(c.Request.Context(), userID, providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt token"})
		return
	}

	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No token configured for this provider"})
		return
	}

	// Create git provider
	gitProvider, err := git.NewProvider(provider.ProviderType, provider.BaseURL, token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create git provider: " + err.Error()})
		return
	}

	// Fetch repositories
	var projects []*git.Project
	if search != "" {
		projects, err = gitProvider.SearchProjects(c.Request.Context(), search, page, perPage)
	} else {
		projects, err = gitProvider.ListProjects(c.Request.Context(), page, perPage)
	}

	if err != nil {
		if err == git.ErrUnauthorized {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Access token is invalid or expired"})
			return
		}
		if err == git.ErrRateLimited {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limited by git provider"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch repositories: " + err.Error()})
		return
	}

	// Convert to response format
	repositories := make([]*RepositoryResponse, len(projects))
	for i, p := range projects {
		repositories[i] = &RepositoryResponse{
			ID:            p.ID,
			Name:          p.Name,
			FullPath:      p.FullPath,
			Description:   p.Description,
			DefaultBranch: p.DefaultBranch,
			Visibility:    p.Visibility,
			CloneURL:      p.CloneURL,
			SSHCloneURL:   p.SSHCloneURL,
			WebURL:        p.WebURL,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"repositories": repositories,
		"page":         page,
		"per_page":     perPage,
	})
}
