package v1

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	"github.com/anthropics/agentsmesh/backend/pkg/apierr"
	"github.com/gin-gonic/gin"
)

// --- Repo MCP Servers ---

// ListRepoMcpServers lists installed MCP servers for a repository.
// GET /api/v1/organizations/:slug/repositories/:id/mcp-servers?scope=org|user|all
func (h *ExtensionHandler) ListRepoMcpServers(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	tenant := middleware.GetTenant(c)
	scope := c.DefaultQuery("scope", "all")

	servers, err := h.extensionSvc.ListRepoMcpServers(c.Request.Context(), tenant.OrganizationID, repoID, tenant.UserID, scope)
	if err != nil {
		handleServiceError(c, err, "Failed to list repository MCP servers")
		return
	}

	c.JSON(http.StatusOK, gin.H{"mcp_servers": servers})
}

// InstallMcpFromMarketRequest represents a market MCP server installation request.
type InstallMcpFromMarketRequest struct {
	MarketItemID int64             `json:"market_item_id" binding:"required"`
	EnvVars      map[string]string `json:"env_vars"`
	Scope        string            `json:"scope" binding:"required"`
}

// InstallMcpFromMarket installs an MCP server from a marketplace template.
// POST /api/v1/organizations/:slug/repositories/:id/mcp-servers/install-from-market
func (h *ExtensionHandler) InstallMcpFromMarket(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	var req InstallMcpFromMarketRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	if req.Scope == "org" {
		if !requireOrgAdmin(c) {
			return
		}
	}

	tenant := middleware.GetTenant(c)

	server, err := h.extensionSvc.InstallMcpFromMarket(c.Request.Context(), tenant.OrganizationID, repoID, tenant.UserID, req.MarketItemID, req.EnvVars, req.Scope)
	if err != nil {
		handleServiceError(c, err, "Failed to install MCP server from market")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mcp_server": server})
}

// InstallCustomMcpServerRequest represents a custom MCP server installation request.
type InstallCustomMcpServerRequest struct {
	Name          string            `json:"name" binding:"required"`
	Slug          string            `json:"slug" binding:"required"`
	TransportType string            `json:"transport_type" binding:"required"`
	Command       string            `json:"command"`
	Args          json.RawMessage   `json:"args"`
	HttpURL       string            `json:"http_url"`
	HttpHeaders   json.RawMessage   `json:"http_headers"`
	EnvVars       map[string]string `json:"env_vars"`
	Scope         string            `json:"scope" binding:"required"`
}

// InstallCustomMcpServer installs a custom MCP server.
// POST /api/v1/organizations/:slug/repositories/:id/mcp-servers/install-custom
func (h *ExtensionHandler) InstallCustomMcpServer(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	var req InstallCustomMcpServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	switch req.TransportType {
	case "stdio", "http", "sse":
	default:
		apierr.ValidationError(c, "transport_type must be 'stdio', 'http', or 'sse'")
		return
	}

	if len(req.Args) > 0 {
		var args []interface{}
		if err := json.Unmarshal(req.Args, &args); err != nil {
			apierr.ValidationError(c, "args must be a valid JSON array")
			return
		}
		if len(args) > 50 {
			apierr.ValidationError(c, "args exceeds maximum of 50 entries")
			return
		}
	}

	if len(req.HttpHeaders) > 0 {
		var headers map[string]interface{}
		if err := json.Unmarshal(req.HttpHeaders, &headers); err != nil {
			apierr.ValidationError(c, "http_headers must be a valid JSON object")
			return
		}
		if len(headers) > 20 {
			apierr.ValidationError(c, "http_headers exceeds maximum of 20 entries")
			return
		}
	}

	if req.Scope == "org" {
		if !requireOrgAdmin(c) {
			return
		}
	}

	tenant := middleware.GetTenant(c)

	server := &extension.InstalledMcpServer{
		Name:          req.Name,
		Slug:          req.Slug,
		Scope:         req.Scope,
		TransportType: req.TransportType,
		Command:       req.Command,
		Args:          req.Args,
		HttpURL:       req.HttpURL,
		HttpHeaders:   req.HttpHeaders,
	}

	result, err := h.extensionSvc.InstallCustomMcpServer(c.Request.Context(), tenant.OrganizationID, repoID, tenant.UserID, server, req.EnvVars)
	if err != nil {
		handleServiceError(c, err, "Failed to install custom MCP server")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mcp_server": result})
}

// UpdateMcpServerRequest represents an MCP server update request.
type UpdateMcpServerRequest struct {
	IsEnabled *bool             `json:"is_enabled"`
	EnvVars   map[string]string `json:"env_vars"`
}

// UpdateMcpServer updates an installed MCP server.
// PUT /api/v1/organizations/:slug/repositories/:id/mcp-servers/:installId
func (h *ExtensionHandler) UpdateMcpServer(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	installID, err := strconv.ParseInt(c.Param("installId"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid install ID")
		return
	}

	var req UpdateMcpServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierr.ValidationError(c, err.Error())
		return
	}

	tenant := middleware.GetTenant(c)

	server, err := h.extensionSvc.UpdateMcpServer(c.Request.Context(), tenant.OrganizationID, repoID, installID, tenant.UserID, tenant.UserRole, req.IsEnabled, req.EnvVars)
	if err != nil {
		handleServiceError(c, err, "Failed to update MCP server")
		return
	}

	c.JSON(http.StatusOK, gin.H{"mcp_server": server})
}

// UninstallMcpServer removes an installed MCP server.
// DELETE /api/v1/organizations/:slug/repositories/:id/mcp-servers/:installId
func (h *ExtensionHandler) UninstallMcpServer(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid repository ID")
		return
	}

	installID, err := strconv.ParseInt(c.Param("installId"), 10, 64)
	if err != nil {
		apierr.InvalidInput(c, "Invalid install ID")
		return
	}

	tenant := middleware.GetTenant(c)

	if err := h.extensionSvc.UninstallMcpServer(c.Request.Context(), tenant.OrganizationID, repoID, installID, tenant.UserID, tenant.UserRole); err != nil {
		handleServiceError(c, err, "Failed to uninstall MCP server")
		return
	}

	c.Status(http.StatusNoContent)
}
