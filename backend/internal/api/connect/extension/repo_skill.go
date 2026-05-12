// RepoSkill sub-service handlers — installed Skill management per repository.
// Mirrors backend/internal/api/rest/v1/extension_skills.go (minus
// install-from-upload, which stays REST because Connect lacks multipart).
package extensionconnect

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	extensionv1 "github.com/anthropics/agentsmesh/proto/gen/go/extension/v1"
)

const RepoSkillServiceName = "proto.extension.v1.RepoSkillService"

const (
	ListRepoSkillsProcedure         = "/" + RepoSkillServiceName + "/ListRepoSkills"
	InstallSkillFromMarketProcedure = "/" + RepoSkillServiceName + "/InstallSkillFromMarket"
	InstallSkillFromGitHubProcedure = "/" + RepoSkillServiceName + "/InstallSkillFromGitHub"
	UpdateSkillProcedure            = "/" + RepoSkillServiceName + "/UpdateSkill"
	UninstallSkillProcedure         = "/" + RepoSkillServiceName + "/UninstallSkill"
)

type RepoSkillServer struct{ *Server }

func NewRepoSkillServer(srv *Server) *RepoSkillServer { return &RepoSkillServer{Server: srv} }

func (s *RepoSkillServer) ListRepoSkills(
	ctx context.Context, req *connect.Request[extensionv1.ListRepoSkillsRequest],
) (*connect.Response[extensionv1.ListRepoSkillsResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)

	scope := req.Msg.GetScope()
	if scope == "" {
		scope = "all"
	}
	skills, err := s.extensionSvc.ListRepoSkills(
		ctx, tenant.OrganizationID, req.Msg.GetRepositoryId(), tenant.UserID, scope,
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	items := make([]*extensionv1.InstalledSkill, 0, len(skills))
	for _, sk := range skills {
		items = append(items, toProtoInstalledSkill(sk))
	}
	return connect.NewResponse(&extensionv1.ListRepoSkillsResponse{
		Items: items,
		Total: int64(len(items)),
	}), nil
}

func (s *RepoSkillServer) InstallSkillFromMarket(
	ctx context.Context, req *connect.Request[extensionv1.InstallSkillFromMarketRequest],
) (*connect.Response[extensionv1.InstalledSkill], error) {
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

	skill, err := s.extensionSvc.InstallSkillFromMarket(
		ctx, tenant.OrganizationID, req.Msg.GetRepositoryId(),
		tenant.UserID, req.Msg.GetMarketItemId(), req.Msg.GetScope(),
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoInstalledSkill(skill)), nil
}

func (s *RepoSkillServer) InstallSkillFromGitHub(
	ctx context.Context, req *connect.Request[extensionv1.InstallSkillFromGitHubRequest],
) (*connect.Response[extensionv1.InstalledSkill], error) {
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

	skill, err := s.extensionSvc.InstallSkillFromGitHub(
		ctx, tenant.OrganizationID, req.Msg.GetRepositoryId(), tenant.UserID,
		req.Msg.GetUrl(), req.Msg.GetBranch(), req.Msg.GetPath(), req.Msg.GetScope(),
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoInstalledSkill(skill)), nil
}

func (s *RepoSkillServer) UpdateSkill(
	ctx context.Context, req *connect.Request[extensionv1.UpdateSkillRequest],
) (*connect.Response[extensionv1.InstalledSkill], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)

	// Tri-state propagation: nil = no change, *bool = update.
	var enabled *bool
	if req.Msg.IsEnabled != nil {
		v := req.Msg.GetIsEnabled()
		enabled = &v
	}
	var pinned *int
	if req.Msg.PinnedVersion != nil {
		v := int(req.Msg.GetPinnedVersion())
		pinned = &v
	}

	skill, err := s.extensionSvc.UpdateSkill(
		ctx, tenant.OrganizationID, req.Msg.GetRepositoryId(),
		req.Msg.GetInstallId(), tenant.UserID, tenant.UserRole, enabled, pinned,
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoInstalledSkill(skill)), nil
}

func (s *RepoSkillServer) UninstallSkill(
	ctx context.Context, req *connect.Request[extensionv1.UninstallSkillRequest],
) (*connect.Response[extensionv1.UninstallSkillResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	tenant := middleware.GetTenant(ctx)

	if err := s.extensionSvc.UninstallSkill(
		ctx, tenant.OrganizationID, req.Msg.GetRepositoryId(),
		req.Msg.GetInstallId(), tenant.UserID, tenant.UserRole,
	); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&extensionv1.UninstallSkillResponse{}), nil
}

func MountRepoSkill(mux *http.ServeMux, srv *RepoSkillServer, opts ...connect.HandlerOption) {
	mux.Handle(ListRepoSkillsProcedure, connect.NewUnaryHandler(
		ListRepoSkillsProcedure, srv.ListRepoSkills, opts...,
	))
	mux.Handle(InstallSkillFromMarketProcedure, connect.NewUnaryHandler(
		InstallSkillFromMarketProcedure, srv.InstallSkillFromMarket, opts...,
	))
	mux.Handle(InstallSkillFromGitHubProcedure, connect.NewUnaryHandler(
		InstallSkillFromGitHubProcedure, srv.InstallSkillFromGitHub, opts...,
	))
	mux.Handle(UpdateSkillProcedure, connect.NewUnaryHandler(
		UpdateSkillProcedure, srv.UpdateSkill, opts...,
	))
	mux.Handle(UninstallSkillProcedure, connect.NewUnaryHandler(
		UninstallSkillProcedure, srv.UninstallSkill, opts...,
	))
}
