package bindingconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	bindingv1 "github.com/anthropics/agentsmesh/proto/gen/go/binding/v1"
)

// Scope mutation RPCs — RequestScopes (initiator side) and ApproveScopes
// (target side). Both gate on ResolveOrgScope and require explicit
// initiator_pod + binding_id + scopes.

func (s *Server) RequestScopes(
	ctx context.Context, req *connect.Request[bindingv1.RequestScopesRequest],
) (*connect.Response[bindingv1.PodBinding], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if req.Msg.GetInitiatorPod() == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("initiator_pod is required"),
		)
	}
	if req.Msg.GetBindingId() == 0 {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("binding_id is required"),
		)
	}
	if len(req.Msg.GetScopes()) == 0 {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("scopes is required"),
		)
	}

	binding, err := s.bindingSvc.RequestScopes(
		ctx,
		req.Msg.GetBindingId(),
		req.Msg.GetInitiatorPod(),
		req.Msg.GetScopes(),
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoBinding(binding)), nil
}

func (s *Server) ApproveScopes(
	ctx context.Context, req *connect.Request[bindingv1.ApproveScopesRequest],
) (*connect.Response[bindingv1.PodBinding], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if req.Msg.GetInitiatorPod() == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("initiator_pod is required"),
		)
	}
	if req.Msg.GetBindingId() == 0 {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("binding_id is required"),
		)
	}
	if len(req.Msg.GetScopes()) == 0 {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("scopes is required"),
		)
	}

	binding, err := s.bindingSvc.ApproveScopes(
		ctx,
		req.Msg.GetBindingId(),
		req.Msg.GetInitiatorPod(),
		req.Msg.GetScopes(),
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoBinding(binding)), nil
}
