
package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// setupTestServer creates a gRPC server for testing and returns cleanup function.
func setupTestServer(t *testing.T, adapter *GRPCRunnerAdapter) (string, func()) {
	grpcServer := grpc.NewServer()
	adapter.Register(grpcServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	cleanup := func() {
		grpcServer.Stop()
		listener.Close()
	}

	return listener.Addr().String(), cleanup
}

// connectRunner creates a client connection and stream with timeout.
func connectRunner(t *testing.T, addr, nodeID, orgSlug string) (runnerv1.RunnerService_ConnectClient, *grpc.ClientConn, context.CancelFunc) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := runnerv1.NewRunnerServiceClient(conn)

	md := metadata.New(map[string]string{
		MetadataKeyClientCertDN: "CN=" + nodeID,
		MetadataKeyOrgSlug:      orgSlug,
	})

	// Use timeout context to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := client.Connect(ctx)
	require.NoError(t, err)

	return stream, conn, cancel
}

// completeHandshake performs the initialization handshake.
func completeHandshake(t *testing.T, stream runnerv1.RunnerService_ConnectClient, agents []string) {
	// Send Initialize
	err := stream.Send(&runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Initialize{
			Initialize: &runnerv1.InitializeRequest{ProtocolVersion: 2},
		},
	})
	require.NoError(t, err)

	// Receive InitializeResult (with timeout via context)
	msg, err := stream.Recv()
	require.NoError(t, err)
	require.NotNil(t, msg.GetInitializeResult())

	// Send Initialized
	err = stream.Send(&runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Initialized{
			Initialized: &runnerv1.InitializedConfirm{AvailableAgents: agents},
		},
	})
	require.NoError(t, err)
}
