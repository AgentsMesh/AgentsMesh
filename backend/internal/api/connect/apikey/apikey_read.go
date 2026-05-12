package apikeyconnect

import (
	"context"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	apikeyservice "github.com/anthropics/agentsmesh/backend/internal/service/apikey"
	apikeyv1 "github.com/anthropics/agentsmesh/proto/gen/go/apikey/v1"
)

func (s *Server) ListApiKeys(
	ctx context.Context, req *connect.Request[apikeyv1.ListApiKeysRequest],
) (*connect.Response[apikeyv1.ListApiKeysResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOrgAdmin(ctx); err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	limit := defaultLimit(req.Msg.Limit)
	offset := defaultOffset(req.Msg.Offset)

	keys, total, err := s.apiKeySvc.ListAPIKeys(ctx, &apikeyservice.ListAPIKeysFilter{
		OrganizationID: tenant.OrganizationID,
		Limit:          int(limit),
		Offset:         int(offset),
	})
	if err != nil {
		return nil, mapServiceError(err)
	}

	items := make([]*apikeyv1.ApiKey, 0, len(keys))
	for i := range keys {
		items = append(items, toProtoApiKey(&keys[i]))
	}
	return connect.NewResponse(&apikeyv1.ListApiKeysResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}), nil
}

func (s *Server) GetApiKey(
	ctx context.Context, req *connect.Request[apikeyv1.GetApiKeyRequest],
) (*connect.Response[apikeyv1.ApiKey], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if err := requireOrgAdmin(ctx); err != nil {
		return nil, err
	}

	tenant := middleware.GetTenant(ctx)
	key, err := s.apiKeySvc.GetAPIKey(ctx, req.Msg.GetId(), tenant.OrganizationID)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoApiKey(key)), nil
}
