package envbundleconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/domain/envbundle"
	envbundleservice "github.com/anthropics/agentsmesh/backend/internal/service/envbundle"
	ebv1 "github.com/anthropics/agentsmesh/proto/gen/go/env_bundle/v1"
)

// ListEnvBundles — REST analogue: GET /api/v1/users/env-bundles.
// Filter semantics mirror REST: kind="" disables the filter, agent_slug
// optional → narrows to that slug.
func (s *Server) ListEnvBundles(
	ctx context.Context, req *connect.Request[ebv1.ListEnvBundlesRequest],
) (*connect.Response[ebv1.ListEnvBundlesResponse], error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return nil, err
	}

	filter := envbundle.OwnerFilter{
		OwnerScope: envbundle.OwnerScopeUser,
		OwnerID:    userID,
	}
	if req.Msg.Kind != nil {
		filter.Kind = *req.Msg.Kind
	}
	if req.Msg.AgentSlug != nil {
		v := *req.Msg.AgentSlug
		filter.AgentSlug = &v
	}

	bundles, err := s.svc.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	responses := s.svc.ResponsesWithValues(ctx, bundles)
	items := make([]*ebv1.EnvBundle, 0, len(responses))
	for _, resp := range responses {
		items = append(items, ToProtoEnvBundle(resp))
	}
	total := int64(len(items))
	return connect.NewResponse(&ebv1.ListEnvBundlesResponse{
		Items:  items,
		Total:  total,
		Limit:  int32(total),
		Offset: 0,
	}), nil
}

func (s *Server) GetEnvBundle(
	ctx context.Context, req *connect.Request[ebv1.GetEnvBundleRequest],
) (*connect.Response[ebv1.EnvBundle], error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	bundle, err := s.svc.Get(ctx, envbundle.OwnerScopeUser, userID, req.Msg.GetId())
	if err != nil {
		return nil, mapBundleError(err)
	}
	resp := s.svc.ResponseWithValuesDegrading(ctx, bundle)
	return connect.NewResponse(ToProtoEnvBundle(resp)), nil
}

func (s *Server) CreateEnvBundle(
	ctx context.Context, req *connect.Request[ebv1.CreateEnvBundleRequest],
) (*connect.Response[ebv1.EnvBundle], error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := &envbundleservice.CreateParams{
		OwnerScope:  envbundle.OwnerScopeUser,
		OwnerID:     userID,
		Name:        req.Msg.GetName(),
		Kind:        req.Msg.GetKind(),
		KindPrimary: req.Msg.GetKindPrimary(),
		Data:        req.Msg.Data,
	}
	if req.Msg.AgentSlug != nil {
		v := *req.Msg.AgentSlug
		params.AgentSlug = &v
	}
	if req.Msg.Description != nil {
		v := *req.Msg.Description
		params.Description = &v
	}

	bundle, err := s.svc.Create(ctx, params)
	if err != nil {
		return nil, mapBundleError(err)
	}
	resp, err := s.svc.ResponseWithValues(bundle)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(ToProtoEnvBundle(resp)), nil
}

func (s *Server) UpdateEnvBundle(
	ctx context.Context, req *connect.Request[ebv1.UpdateEnvBundleRequest],
) (*connect.Response[ebv1.EnvBundle], error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return nil, err
	}

	params := &envbundleservice.UpdateParams{
		Name:        req.Msg.Name,
		Description: req.Msg.Description,
		Kind:        req.Msg.Kind,
		KindPrimary: req.Msg.KindPrimary,
		IsActive:    req.Msg.IsActive,
	}
	// Three-state Data semantics: has_data flag distinguishes "leave alone"
	// (false) from "clear" (true + empty) vs "replace" (true + populated).
	if req.Msg.GetHasData() {
		data := req.Msg.Data
		if data == nil {
			data = map[string]string{}
		}
		params.Data = &data
	}

	bundle, err := s.svc.Update(ctx, envbundle.OwnerScopeUser, userID, req.Msg.GetId(), params)
	if err != nil {
		return nil, mapBundleError(err)
	}
	resp := s.svc.ResponseWithValuesDegrading(ctx, bundle)
	return connect.NewResponse(ToProtoEnvBundle(resp)), nil
}

func (s *Server) DeleteEnvBundle(
	ctx context.Context, req *connect.Request[ebv1.DeleteEnvBundleRequest],
) (*connect.Response[ebv1.DeleteEnvBundleResponse], error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.svc.Delete(ctx, envbundle.OwnerScopeUser, userID, req.Msg.GetId()); err != nil {
		return nil, mapBundleError(err)
	}
	return connect.NewResponse(&ebv1.DeleteEnvBundleResponse{}), nil
}

func (s *Server) SetPrimaryEnvBundle(
	ctx context.Context, req *connect.Request[ebv1.SetPrimaryEnvBundleRequest],
) (*connect.Response[ebv1.SetPrimaryEnvBundleResponse], error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.svc.SetPrimary(ctx, envbundle.OwnerScopeUser, userID, req.Msg.GetId()); err != nil {
		return nil, mapBundleError(err)
	}
	return connect.NewResponse(&ebv1.SetPrimaryEnvBundleResponse{}), nil
}

func mapBundleError(err error) error {
	switch {
	case errors.Is(err, envbundleservice.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, envbundleservice.ErrNameExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, envbundleservice.ErrInvalidKind),
		errors.Is(err, envbundleservice.ErrInvalidScope):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
