// Package autopilotconnect hosts Connect-RPC handlers for the autopilot
// controller service. Mirrors backend/internal/api/rest/v1/autopilot_controller*.go.
//
// Split rationale (200-line rule):
//   - autopilot.go              — server scaffolding + Mount + queries
//   - autopilot_actions.go      — 6 control actions + Create
//   - autopilot_convert.go      — domain ↔ proto translation
package autopilotconnect

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/anthropics/agentsmesh/backend/internal/api/connect/interceptors"
	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/middleware"
	agentpodsvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	apv1 "github.com/anthropics/agentsmesh/proto/gen/go/autopilot/v1"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const ServiceName = "proto.autopilot.v1.AutopilotControllerService"

const (
	ListAutopilotControllersProcedure    = "/" + ServiceName + "/ListAutopilotControllers"
	GetAutopilotControllerProcedure      = "/" + ServiceName + "/GetAutopilotController"
	CreateAutopilotControllerProcedure   = "/" + ServiceName + "/CreateAutopilotController"
	PauseAutopilotControllerProcedure    = "/" + ServiceName + "/PauseAutopilotController"
	ResumeAutopilotControllerProcedure   = "/" + ServiceName + "/ResumeAutopilotController"
	StopAutopilotControllerProcedure     = "/" + ServiceName + "/StopAutopilotController"
	ApproveAutopilotControllerProcedure  = "/" + ServiceName + "/ApproveAutopilotController"
	TakeoverAutopilotControllerProcedure = "/" + ServiceName + "/TakeoverAutopilotController"
	HandbackAutopilotControllerProcedure = "/" + ServiceName + "/HandbackAutopilotController"
	GetIterationsProcedure               = "/" + ServiceName + "/GetIterations"
)

// AutopilotServiceInterface mirrors REST AutopilotControllerServiceInterface
// (v1/autopilot_controller.go:23).
type AutopilotServiceInterface interface {
	GetAutopilotController(ctx context.Context, orgID int64, key string) (*agentpod.AutopilotController, error)
	ListAutopilotControllers(ctx context.Context, orgID int64) ([]*agentpod.AutopilotController, error)
	CreateAndStart(ctx context.Context, req *agentpodsvc.CreateAndStartRequest) (*agentpod.AutopilotController, error)
	GetIterations(ctx context.Context, autopilotPodID int64) ([]*agentpod.AutopilotIteration, error)
}

// PodLookup is the slice of pod service we need.
type PodLookup interface {
	GetPod(ctx context.Context, podKey string) (*agentpod.Pod, error)
}

// CommandSender mirrors REST AutopilotControllerCommandSender
// (v1/autopilot_controller.go:18).
type CommandSender interface {
	SendAutopilotControl(runnerID int64, cmd *runnerv1.AutopilotControlCommand) error
}

type Server struct {
	svc           AutopilotServiceInterface
	orgSvc        middleware.OrganizationService
	podSvc        PodLookup
	commandSender CommandSender
}

func NewServer(
	svc AutopilotServiceInterface,
	orgSvc middleware.OrganizationService,
	podSvc PodLookup,
	cmdSender CommandSender,
) *Server {
	return &Server{
		svc:           svc,
		orgSvc:        orgSvc,
		podSvc:        podSvc,
		commandSender: cmdSender,
	}
}

func Mount(mux *http.ServeMux, srv *Server, opts ...connect.HandlerOption) {
	mux.Handle(ListAutopilotControllersProcedure, connect.NewUnaryHandler(
		ListAutopilotControllersProcedure, srv.ListAutopilotControllers, opts...,
	))
	mux.Handle(GetAutopilotControllerProcedure, connect.NewUnaryHandler(
		GetAutopilotControllerProcedure, srv.GetAutopilotController, opts...,
	))
	mux.Handle(CreateAutopilotControllerProcedure, connect.NewUnaryHandler(
		CreateAutopilotControllerProcedure, srv.CreateAutopilotController, opts...,
	))
	mux.Handle(PauseAutopilotControllerProcedure, connect.NewUnaryHandler(
		PauseAutopilotControllerProcedure, srv.PauseAutopilotController, opts...,
	))
	mux.Handle(ResumeAutopilotControllerProcedure, connect.NewUnaryHandler(
		ResumeAutopilotControllerProcedure, srv.ResumeAutopilotController, opts...,
	))
	mux.Handle(StopAutopilotControllerProcedure, connect.NewUnaryHandler(
		StopAutopilotControllerProcedure, srv.StopAutopilotController, opts...,
	))
	mux.Handle(ApproveAutopilotControllerProcedure, connect.NewUnaryHandler(
		ApproveAutopilotControllerProcedure, srv.ApproveAutopilotController, opts...,
	))
	mux.Handle(TakeoverAutopilotControllerProcedure, connect.NewUnaryHandler(
		TakeoverAutopilotControllerProcedure, srv.TakeoverAutopilotController, opts...,
	))
	mux.Handle(HandbackAutopilotControllerProcedure, connect.NewUnaryHandler(
		HandbackAutopilotControllerProcedure, srv.HandbackAutopilotController, opts...,
	))
	mux.Handle(GetIterationsProcedure, connect.NewUnaryHandler(
		GetIterationsProcedure, srv.GetIterations, opts...,
	))
}

// ListAutopilotControllers — REST analogue: GET /autopilot-controllers.
func (s *Server) ListAutopilotControllers(
	ctx context.Context, req *connect.Request[apv1.ListAutopilotControllersRequest],
) (*connect.Response[apv1.ListAutopilotControllersResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if s.svc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("autopilot service not configured"))
	}
	tenant := middleware.GetTenant(ctx)
	pods, err := s.svc.ListAutopilotControllers(ctx, tenant.OrganizationID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*apv1.AutopilotController, 0, len(pods))
	for _, p := range pods {
		items = append(items, toProtoController(p))
	}
	return connect.NewResponse(&apv1.ListAutopilotControllersResponse{
		Items:  items,
		Total:  int64(len(items)),
		Limit:  int32(len(items)),
		Offset: 0,
	}), nil
}

// GetAutopilotController — REST analogue: GET /autopilot-controllers/:key.
func (s *Server) GetAutopilotController(
	ctx context.Context, req *connect.Request[apv1.GetAutopilotControllerRequest],
) (*connect.Response[apv1.AutopilotController], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if s.svc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("autopilot service not configured"))
	}
	if req.Msg.GetKey() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("key is required"))
	}
	tenant := middleware.GetTenant(ctx)
	ap, err := s.svc.GetAutopilotController(ctx, tenant.OrganizationID, req.Msg.GetKey())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("autopilot pod not found"))
	}
	return connect.NewResponse(toProtoController(ap)), nil
}

// GetIterations — REST analogue: GET /autopilot-controllers/:key/iterations.
func (s *Server) GetIterations(
	ctx context.Context, req *connect.Request[apv1.GetIterationsRequest],
) (*connect.Response[apv1.GetIterationsResponse], error) {
	ctx, _, err := interceptors.ResolveOrgScope(ctx, req.Msg, s.orgSvc)
	if err != nil {
		return nil, err
	}
	if s.svc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("autopilot service not configured"))
	}
	if req.Msg.GetKey() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("key is required"))
	}
	tenant := middleware.GetTenant(ctx)
	ap, err := s.svc.GetAutopilotController(ctx, tenant.OrganizationID, req.Msg.GetKey())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("autopilot pod not found"))
	}
	iters, err := s.svc.GetIterations(ctx, ap.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*apv1.AutopilotIteration, 0, len(iters))
	for _, it := range iters {
		items = append(items, toProtoIteration(it))
	}
	return connect.NewResponse(&apv1.GetIterationsResponse{
		Items:  items,
		Total:  int64(len(items)),
		Limit:  int32(len(items)),
		Offset: 0,
	}), nil
}
