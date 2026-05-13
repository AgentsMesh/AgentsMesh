package bindingconnect

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	bindingservice "github.com/anthropics/agentsmesh/backend/internal/service/binding"
	bindingv1 "github.com/anthropics/agentsmesh/proto/gen/go/binding/v1"
)

// Read-only RPCs: ListBindings, GetPendingBindings, GetBoundPods,
// CheckBinding.

func (s *Server) ListBindings(
	ctx context.Context, req *connect.Request[bindingv1.ListBindingsRequest],
) (*connect.Response[bindingv1.ListBindingsResponse], error) {
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

	var statusFilter *string
	if s := req.Msg.GetStatus(); s != "" {
		statusFilter = &s
	}

	bindings, err := s.bindingSvc.GetBindingsForPod(
		ctx, req.Msg.GetInitiatorPod(), statusFilter,
	)
	if err != nil {
		return nil, mapServiceError(err)
	}

	items := make([]*bindingv1.PodBinding, 0, len(bindings))
	for _, b := range bindings {
		items = append(items, toProtoBinding(b))
	}
	return connect.NewResponse(&bindingv1.ListBindingsResponse{
		Items: items,
		Total: int64(len(items)),
	}), nil
}

func (s *Server) GetPendingBindings(
	ctx context.Context, req *connect.Request[bindingv1.GetPendingBindingsRequest],
) (*connect.Response[bindingv1.ListBindingsResponse], error) {
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

	pending, err := s.bindingSvc.GetPendingRequests(ctx, req.Msg.GetInitiatorPod())
	if err != nil {
		return nil, mapServiceError(err)
	}
	items := make([]*bindingv1.PodBinding, 0, len(pending))
	for _, b := range pending {
		items = append(items, toProtoBinding(b))
	}
	return connect.NewResponse(&bindingv1.ListBindingsResponse{
		Items: items,
		Total: int64(len(items)),
	}), nil
}

func (s *Server) GetBoundPods(
	ctx context.Context, req *connect.Request[bindingv1.GetBoundPodsRequest],
) (*connect.Response[bindingv1.GetBoundPodsResponse], error) {
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

	pods, err := s.bindingSvc.GetBoundPods(ctx, req.Msg.GetInitiatorPod())
	if err != nil {
		return nil, mapServiceError(err)
	}
	return connect.NewResponse(&bindingv1.GetBoundPodsResponse{Pods: pods}), nil
}

func (s *Server) CheckBinding(
	ctx context.Context, req *connect.Request[bindingv1.CheckBindingRequest],
) (*connect.Response[bindingv1.CheckBindingResponse], error) {
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

	isBound, err := s.bindingSvc.IsBound(
		ctx, req.Msg.GetInitiatorPod(), req.Msg.GetTargetPod(),
	)
	if err != nil {
		return nil, mapServiceError(err)
	}
	resp := &bindingv1.CheckBindingResponse{IsBound: isBound}
	if isBound {
		// Mirrors REST's two-direction lookup (bindings_query.go:132-138).
		binding, errA := s.bindingSvc.GetActiveBinding(
			ctx, req.Msg.GetInitiatorPod(), req.Msg.GetTargetPod(),
		)
		if errors.Is(errA, bindingservice.ErrBindingNotFound) {
			binding, _ = s.bindingSvc.GetActiveBinding(
				ctx, req.Msg.GetTargetPod(), req.Msg.GetInitiatorPod(),
			)
		}
		if binding != nil {
			resp.Binding = toProtoBinding(binding)
		}
	}
	return connect.NewResponse(resp), nil
}
