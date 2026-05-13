package v1

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func registerExtensionRoutes(rg *gin.RouterGroup, svc *Services) {
	if svc.Extension == nil {
		slog.Warn("Extension services not configured, extension routes not registered")
		return
	}

	handler := NewExtensionHandler(svc.Extension)

	skillRegistries := rg.Group("/skill-registries")
	{
		skillRegistries.GET("", handler.ListSkillRegistries)
		skillRegistries.POST("", handler.CreateSkillRegistry)
		skillRegistries.POST("/:id/sync", handler.SyncSkillRegistry)
		skillRegistries.DELETE("/:id", handler.DeleteSkillRegistry)
		skillRegistries.PUT("/:id/toggle", handler.TogglePlatformRegistry)
	}

	rg.GET("/skill-registry-overrides", handler.ListSkillRegistryOverrides)

	market := rg.Group("/market")
	{
		market.GET("/skills", handler.ListMarketSkills)
		market.GET("/mcp-servers", handler.ListMarketMcpServers)
	}

	repoSkills := rg.Group("/repositories/:id/skills")
	{
		repoSkills.GET("", handler.ListRepoSkills)
		repoSkills.POST("/install-from-market", handler.InstallSkillFromMarket)
		repoSkills.POST("/install-from-github", handler.InstallSkillFromGitHub)
		repoSkills.POST("/install-from-upload", handler.InstallSkillFromUpload)
		repoSkills.PUT("/:installId", handler.UpdateSkill)
		repoSkills.DELETE("/:installId", handler.UninstallSkill)
	}

	repoMcp := rg.Group("/repositories/:id/mcp-servers")
	{
		repoMcp.GET("", handler.ListRepoMcpServers)
		repoMcp.POST("/install-from-market", handler.InstallMcpFromMarket)
		repoMcp.POST("/install-custom", handler.InstallCustomMcpServer)
		repoMcp.PUT("/:installId", handler.UpdateMcpServer)
		repoMcp.DELETE("/:installId", handler.UninstallMcpServer)
	}

	slog.Info("Extension routes registered")
}
