package apikeyconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	apikeyservice "github.com/anthropics/agentsmesh/backend/internal/service/apikey"
	apikeyv1 "github.com/anthropics/agentsmesh/proto/gen/go/apikey/v1"
)

func (s *Server) CreateApiKey(
	ctx context.Context, req *connect.Request[apikeyv1.CreateApiKeyRequest],
) (*connect.Response[apikeyv1.CreateApiKeyResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOrgAdmin(ctx); err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	createReq := &apikeyservice.CreateAPIKeyRequest{
		OrganizationID: tenant.OrganizationID,
		CreatedBy:      tenant.UserID,
		Name:           req.Msg.GetName(),
		Description:    optionalString(req.Msg.Description),
		Scopes:         req.Msg.GetScopes(),
		ExpiresIn:      optionalIntFromInt64(req.Msg.ExpiresIn),
	}
	result, err := s.apiKeySvc.CreateAPIKey(ctx, createReq)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&apikeyv1.CreateApiKeyResponse{
		ApiKey: toProtoApiKey(result.APIKey),
		RawKey: result.RawKey,
	}), nil
}

func (s *Server) UpdateApiKey(
	ctx context.Context, req *connect.Request[apikeyv1.UpdateApiKeyRequest],
) (*connect.Response[apikeyv1.ApiKey], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOrgAdmin(ctx); err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	updateReq := &apikeyservice.UpdateAPIKeyRequest{
		Name:        optionalString(req.Msg.Name),
		Description: optionalString(req.Msg.Description),
		IsEnabled:   req.Msg.IsEnabled,
	}
	// scopes: empty repeated proto field decodes as nil → matches REST
	// PATCH semantics (apikey_update.go ignores nil; non-nil replaces).
	if scopes := req.Msg.GetScopes(); len(scopes) > 0 {
		updateReq.Scopes = scopes
	}

	key, err := s.apiKeySvc.UpdateAPIKey(ctx, req.Msg.GetId(), tenant.OrganizationID, updateReq)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoApiKey(key)), nil
}

func (s *Server) RevokeApiKey(
	ctx context.Context, req *connect.Request[apikeyv1.RevokeApiKeyRequest],
) (*connect.Response[apikeyv1.RevokeApiKeyResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOrgAdmin(ctx); err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	if err := s.apiKeySvc.RevokeAPIKey(ctx, req.Msg.GetId(), tenant.OrganizationID); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&apikeyv1.RevokeApiKeyResponse{}), nil
}

func (s *Server) DeleteApiKey(
	ctx context.Context, req *connect.Request[apikeyv1.DeleteApiKeyRequest],
) (*connect.Response[apikeyv1.DeleteApiKeyResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOrgAdmin(ctx); err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	if err := s.apiKeySvc.DeleteAPIKey(ctx, req.Msg.GetId(), tenant.OrganizationID); err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&apikeyv1.DeleteApiKeyResponse{}), nil
}
