// Package grpc implements the gRPC server for the OmniTest Controller.
package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/omnitest/omnitest/internal/grpc/omnitestv1"
)

// AgentHandler defines the interface that the controller must implement
// to handle agent-related gRPC calls.
type AgentHandler interface {
	HandleRegister(req *omnitestv1.RegisterRequest) (*omnitestv1.RegisterResponse, error)
	HandleHeartbeat(req *omnitestv1.HeartbeatRequest) error
	HandleMetricReport(report *omnitestv1.MetricReport) error
}

// Server wraps a gRPC server for the controller.
type Server struct {
	omnitestv1.UnimplementedAgentServiceServer
	grpcServer *grpc.Server
	port       int
	handler    AgentHandler

	// startTestHandlers maps agent_id → channel for delivering StartTest requests.
	// Set externally by Controller/Scheduler.
	OnStartTest func(req *omnitestv1.StartTestRequest) (*omnitestv1.StartTestResponse, error)
	OnStopTest  func(req *omnitestv1.StopTestRequest) (*omnitestv1.StopTestResponse, error)
}

// NewServer creates a new gRPC server.
func NewServer(port int, handler AgentHandler) *Server {
	grpcServer := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    10 * time.Second,
			Timeout: 5 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	s := &Server{
		grpcServer: grpcServer,
		port:       port,
		handler:    handler,
	}

	omnitestv1.RegisterAgentServiceServer(grpcServer, s)
	return s
}

// Start begins listening for gRPC connections.
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}
	log.Printf("[gRPC] Listening on :%d", s.port)
	return s.grpcServer.Serve(lis)
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

// Register handles agent registration.
func (s *Server) Register(_ context.Context, req *omnitestv1.RegisterRequest) (*omnitestv1.RegisterResponse, error) {
	return s.handler.HandleRegister(req)
}

// Heartbeat handles agent heartbeat.
func (s *Server) Heartbeat(_ context.Context, req *omnitestv1.HeartbeatRequest) (*omnitestv1.HeartbeatResponse, error) {
	if err := s.handler.HandleHeartbeat(req); err != nil {
		return nil, err
	}
	return &omnitestv1.HeartbeatResponse{Acknowledged: true}, nil
}

// StartTest handles test start command.
func (s *Server) StartTest(_ context.Context, req *omnitestv1.StartTestRequest) (*omnitestv1.StartTestResponse, error) {
	if s.OnStartTest != nil {
		return s.OnStartTest(req)
	}
	return &omnitestv1.StartTestResponse{Accepted: false, ErrorMessage: "no handler registered"}, nil
}

// StopTest handles test stop command.
func (s *Server) StopTest(_ context.Context, req *omnitestv1.StopTestRequest) (*omnitestv1.StopTestResponse, error) {
	if s.OnStopTest != nil {
		return s.OnStopTest(req)
	}
	return &omnitestv1.StopTestResponse{Stopped: false}, nil
}

// StreamMetrics handles client-streaming metrics from agents.
func (s *Server) StreamMetrics(stream omnitestv1.AgentService_StreamMetricsServer) error {
	for {
		report, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&omnitestv1.StreamMetricsResponse{Acknowledged: true})
		}
		if err != nil {
			return err
		}
		if err := s.handler.HandleMetricReport(report); err != nil {
			log.Printf("[gRPC] Error handling metric report: %v", err)
		}
	}
}
