// RepoMcp sub-service handlers — installed MCP server management per
// repository. Mirrors backend/internal/api/rest/v1/extension_mcp.go.
package extensionconnect

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	extdom "github.com/anthropics/agentsmesh/backend/internal/domain/extension"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	extensionv1 "github.com/anthropics/agentsmesh/proto/gen/go/extension/v1"
)

const RepoMcpServiceName = "proto.extension.v1.RepoMcpService"

const (
	ListRepoMcpServersProcedure     = "/" + RepoMcpServiceName + "/ListRepoMcpServers"
	InstallMcpFromMarketProcedure   = "/" + RepoMcpServiceName + "/InstallMcpFromMarket"
	InstallCustomMcpServerProcedure = "/" + RepoMcpServiceName + "/InstallCustomMcpServer"
	UpdateMcpServerProcedure        = "/" + RepoMcpServiceName + "/UpdateMcpServer"
	UninstallMcpServerProcedure     = "/" + RepoMcpServiceName + "/UninstallMcpServer"
)

type RepoMcpServer struct{ *Server }

func NewRepoMcpServer(srv *Server) *RepoMcpServer { return &RepoMcpServer{Server: srv} }

func (s *RepoMcpServer) ListRepoMcpServers(
	ctx context.Context, req *connect.Request[extensionv1.ListRepoMcpServersRequest],
) (*connect.Response[extensionv1.ListRepoMcpServersResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)

	scope := req.Msg.GetScope()
	if scope == "" {
		scope = "all"
	}
	servers, err := s.extensionSvc.ListRepoMcpServers(
		ctx, tenant.OrganizationID, req.Msg.GetRepositoryId(), tenant.UserID, scope,
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	items := make([]*extensionv1.InstalledMcpServer, 0, len(servers))
	for _, srv := range servers {
		items = append(items, toProtoInstalledMcpServer(srv))
	}
	return connect.NewResponse(&extensionv1.ListRepoMcpServersResponse{
		Items: items,
		Total: int64(len(items)),
	}), nil
}

func (s *RepoMcpServer) InstallMcpFromMarket(
	ctx context.Context, req *connect.Request[extensionv1.InstallMcpFromMarketRequest],
) (*connect.Response[extensionv1.InstalledMcpServer], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if req.Msg.GetScope() == "org" {
		if err := requireOrgAdmin(ctx); err != nil {
			return nil, err
		}
	}
	tenant := middleware.GetTenant(ctx)

	envVars, err := decodeEnvVars(req.Msg.GetEnvVars())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	server, err := s.extensionSvc.InstallMcpFromMarket(
		ctx, tenant.OrganizationID, req.Msg.GetRepositoryId(), tenant.UserID,
		req.Msg.GetMarketItemId(), envVars, req.Msg.GetScope(),
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoInstalledMcpServer(server)), nil
}

func (s *RepoMcpServer) InstallCustomMcpServer(
	ctx context.Context, req *connect.Request[extensionv1.InstallCustomMcpServerRequest],
) (*connect.Response[extensionv1.InstalledMcpServer], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}

	switch req.Msg.GetTransportType() {
	case "stdio", "http", "sse":
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("transport_type must be 'stdio', 'http', or 'sse'"))
	}

	args, err := validateJSONArrayBound(req.Msg.GetArgs(), 50, "args")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	headers, err := validateJSONObjectBound(req.Msg.GetHttpHeaders(), 20, "http_headers")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if req.Msg.GetScope() == "org" {
		if err := requireOrgAdmin(ctx); err != nil {
			return nil, err
		}
	}
	tenant := middleware.GetTenant(ctx)

	envVars, err := decodeEnvVars(req.Msg.GetEnvVars())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	server := &extdom.InstalledMcpServer{
		Name:          req.Msg.GetName(),
		Slug:          req.Msg.GetSlug(),
		Scope:         req.Msg.GetScope(),
		TransportType: req.Msg.GetTransportType(),
		Command:       req.Msg.GetCommand(),
		Args:          args,
		HttpURL:       req.Msg.GetHttpUrl(),
		HttpHeaders:   headers,
	}

	result, err := s.extensionSvc.InstallCustomMcpServer(
		ctx, tenant.OrganizationID, req.Msg.GetRepositoryId(), tenant.UserID, server, envVars,
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoInstalledMcpServer(result)), nil
}

func (s *RepoMcpServer) UpdateMcpServer(
	ctx context.Context, req *connect.Request[extensionv1.UpdateMcpServerRequest],
) (*connect.Response[extensionv1.InstalledMcpServer], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)

	var enabled *bool
	if req.Msg.IsEnabled != nil {
		v := req.Msg.GetIsEnabled()
		enabled = &v
	}
	envVars, err := decodeEnvVars(req.Msg.GetEnvVars())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	server, err := s.extensionSvc.UpdateMcpServer(
		ctx, tenant.OrganizationID, req.Msg.GetRepositoryId(),
		req.Msg.GetInstallId(), tenant.UserID, tenant.UserRole, enabled, envVars,
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoInstalledMcpServer(server)), nil
}

func (s *RepoMcpServer) UninstallMcpServer(
	ctx context.Context, req *connect.Request[extensionv1.UninstallMcpServerRequest],
) (*connect.Response[extensionv1.UninstallMcpServerResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)

	if err := s.extensionSvc.UninstallMcpServer(
		ctx, tenant.OrganizationID, req.Msg.GetRepositoryId(),
		req.Msg.GetInstallId(), tenant.UserID, tenant.UserRole,
	); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&extensionv1.UninstallMcpServerResponse{}), nil
}

func MountRepoMcp(mux *http.ServeMux, srv *RepoMcpServer, opts ...connect.HandlerOption) {
	mux.Handle(ListRepoMcpServersProcedure, connect.NewUnaryHandler(
		ListRepoMcpServersProcedure, srv.ListRepoMcpServers, opts...,
	))
	mux.Handle(InstallMcpFromMarketProcedure, connect.NewUnaryHandler(
		InstallMcpFromMarketProcedure, srv.InstallMcpFromMarket, opts...,
	))
	mux.Handle(InstallCustomMcpServerProcedure, connect.NewUnaryHandler(
		InstallCustomMcpServerProcedure, srv.InstallCustomMcpServer, opts...,
	))
	mux.Handle(UpdateMcpServerProcedure, connect.NewUnaryHandler(
		UpdateMcpServerProcedure, srv.UpdateMcpServer, opts...,
	))
	mux.Handle(UninstallMcpServerProcedure, connect.NewUnaryHandler(
		UninstallMcpServerProcedure, srv.UninstallMcpServer, opts...,
	))
}

// decodeEnvVars parses a JSON object string into the map shape the service
// layer expects. Empty / unset returns nil so the service can skip env_var
// encryption.
func decodeEnvVars(s string) (map[string]string, error) {
	if s == "" {
		return nil, nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, errors.New("env_vars must be a valid JSON object of string values")
	}
	return m, nil
}

// validateJSONArrayBound mirrors REST's extension_mcp.go:111-120 bound check
// for `args`. Returns the raw bytes (or nil for empty input) so callers can
// hand them straight to the GORM jsonb column.
func validateJSONArrayBound(s string, max int, field string) (json.RawMessage, error) {
	if s == "" {
		return nil, nil
	}
	var arr []interface{}
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		return nil, errors.New(field + " must be a valid JSON array")
	}
	if len(arr) > max {
		return nil, errors.New(field + " exceeds maximum entries")
	}
	return json.RawMessage(s), nil
}

func validateJSONObjectBound(s string, max int, field string) (json.RawMessage, error) {
	if s == "" {
		return nil, nil
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return nil, errors.New(field + " must be a valid JSON object")
	}
	if len(obj) > max {
		return nil, errors.New(field + " exceeds maximum entries")
	}
	return json.RawMessage(s), nil
}
