package autopilotconnect

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apv1 "github.com/anthropics/agentsmesh/proto/gen/go/autopilot/v1"
)

func connectCodeOf(t *testing.T, err error) connect.Code {
	t.Helper()
	var ce *connect.Error
	require.True(t, errors.As(err, &ce), "expected *connect.Error, got %v", err)
	return ce.Code()
}

// --- Validation guards (org_slug missing) ---

func TestListAutopilots_NoOrgSlug(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil)
	_, err := srv.ListAutopilotControllers(context.Background(),
		connect.NewRequest(&apv1.ListAutopilotControllersRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestGetAutopilot_NoOrgSlug(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil)
	_, err := srv.GetAutopilotController(context.Background(),
		connect.NewRequest(&apv1.GetAutopilotControllerRequest{Key: "k"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestCreateAutopilot_NoOrgSlug(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil)
	_, err := srv.CreateAutopilotController(context.Background(),
		connect.NewRequest(&apv1.CreateAutopilotControllerRequest{PodKey: "p"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestPauseAutopilot_NoOrgSlug(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil)
	_, err := srv.PauseAutopilotController(context.Background(),
		connect.NewRequest(&apv1.ActionRequest{Key: "k"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestApproveAutopilot_NoOrgSlug(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil)
	_, err := srv.ApproveAutopilotController(context.Background(),
		connect.NewRequest(&apv1.ApproveRequest{Key: "k"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

func TestGetIterations_NoOrgSlug(t *testing.T) {
	srv := NewServer(nil, nil, nil, nil)
	_, err := srv.GetIterations(context.Background(),
		connect.NewRequest(&apv1.GetIterationsRequest{Key: "k"}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connectCodeOf(t, err))
}

// --- procedure constants identity (conventions §12) ---

func TestProcedureConstants(t *testing.T) {
	cases := map[string]string{
		"/proto.autopilot.v1.AutopilotControllerService/ListAutopilotControllers":    ListAutopilotControllersProcedure,
		"/proto.autopilot.v1.AutopilotControllerService/GetAutopilotController":      GetAutopilotControllerProcedure,
		"/proto.autopilot.v1.AutopilotControllerService/CreateAutopilotController":   CreateAutopilotControllerProcedure,
		"/proto.autopilot.v1.AutopilotControllerService/PauseAutopilotController":    PauseAutopilotControllerProcedure,
		"/proto.autopilot.v1.AutopilotControllerService/ResumeAutopilotController":   ResumeAutopilotControllerProcedure,
		"/proto.autopilot.v1.AutopilotControllerService/StopAutopilotController":     StopAutopilotControllerProcedure,
		"/proto.autopilot.v1.AutopilotControllerService/ApproveAutopilotController":  ApproveAutopilotControllerProcedure,
		"/proto.autopilot.v1.AutopilotControllerService/TakeoverAutopilotController": TakeoverAutopilotControllerProcedure,
		"/proto.autopilot.v1.AutopilotControllerService/HandbackAutopilotController": HandbackAutopilotControllerProcedure,
		"/proto.autopilot.v1.AutopilotControllerService/GetIterations":               GetIterationsProcedure,
	}
	for want, got := range cases {
		assert.Equal(t, want, got)
	}
}
