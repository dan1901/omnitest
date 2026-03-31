package grpc

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/omnitest/omnitest/internal/grpc/omnitestv1"
)

// NewClientConn creates a gRPC client connection to the controller.
func NewClientConn(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to controller at %s: %w", addr, err)
	}
	return conn, nil
}

// NewAgentClient creates an AgentServiceClient from a connection.
func NewAgentClient(conn *grpc.ClientConn) omnitestv1.AgentServiceClient {
	return omnitestv1.NewAgentServiceClient(conn)
}
