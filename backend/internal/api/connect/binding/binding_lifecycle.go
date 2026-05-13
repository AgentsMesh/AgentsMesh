package bindingconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	bindingv1 "github.com/anthropics/agentsmesh/proto/gen/go/binding/v1"
)

// Mutating RPCs that change binding lifecycle state: RequestBinding,
// AcceptBinding, RejectBinding, Unbind. RequestScopes / ApproveScopes
// live in binding_scopes.go to keep file size under the 200-line ceiling.

func (s *Server) RequestBinding(
	ctx context.Context, req *connect.Request[bindingv1.RequestBindingRequest],
) (*connect.Response[bindingv1.PodBinding], error) {
	ctx, org, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if req.Msg.GetInitiatorPod() == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("initiator_pod is required"),
		)
	}
	if req.Msg.GetTargetPod() == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("target_pod is required"),
		)
	}
	if len(req.Msg.GetScopes()) == 0 {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("scopes is required"),
		)
	}

	binding, err := s.bindingSvc.RequestBinding(
		ctx,
		org.GetID(),
		req.Msg.GetInitiatorPod(),
		req.Msg.GetTargetPod(),
		req.Msg.GetScopes(),
		req.Msg.GetPolicy(),
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoBinding(binding)), nil
}

func (s *Server) AcceptBinding(
	ctx context.Context, req *connect.Request[bindingv1.AcceptBindingRequest],
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

	binding, err := s.bindingSvc.AcceptBinding(
		ctx, req.Msg.GetBindingId(), req.Msg.GetInitiatorPod(),
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoBinding(binding)), nil
}

func (s *Server) RejectBinding(
	ctx context.Context, req *connect.Request[bindingv1.RejectBindingRequest],
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

	binding, err := s.bindingSvc.RejectBinding(
		ctx, req.Msg.GetBindingId(), req.Msg.GetInitiatorPod(), req.Msg.GetReason(),
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(toProtoBinding(binding)), nil
}

func (s *Server) Unbind(
	ctx context.Context, req *connect.Request[bindingv1.UnbindRequest],
) (*connect.Response[bindingv1.UnbindResponse], error) {
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
	if req.Msg.GetTargetPod() == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("target_pod is required"),
		)
	}

	removed, err := s.bindingSvc.Unbind(
		ctx, req.Msg.GetInitiatorPod(), req.Msg.GetTargetPod(),
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	// REST returned 404 when no active binding existed; Connect surfaces
	// that as `removed = false` so the caller can distinguish "successfully
	// removed" from "nothing to remove" without exception-string parsing.
	return connect.NewResponse(&bindingv1.UnbindResponse{Removed: removed}), nil
}
