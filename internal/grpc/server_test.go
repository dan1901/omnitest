package grpc

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/omnitest/omnitest/internal/grpc/omnitestv1"
)

// mockAgentHandler implements AgentHandler for testing.
type mockAgentHandler struct {
	registerCalled  bool
	heartbeatCalled bool
	metricCalled    bool
}

func (m *mockAgentHandler) HandleRegister(req *omnitestv1.RegisterRequest) (*omnitestv1.RegisterResponse, error) {
	m.registerCalled = true
	return &omnitestv1.RegisterResponse{
		Accepted:                 true,
		ControllerId:             "test-controller",
		HeartbeatIntervalSeconds: 10,
	}, nil
}

func (m *mockAgentHandler) HandleHeartbeat(req *omnitestv1.HeartbeatRequest) error {
	m.heartbeatCalled = true
	return nil
}

func (m *mockAgentHandler) HandleMetricReport(report *omnitestv1.MetricReport) error {
	m.metricCalled = true
	return nil
}

func TestNewServer(t *testing.T) {
	handler := &mockAgentHandler{}
	server := NewServer(19999, handler)
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
	if server.port != 19999 {
		t.Errorf("port = %d, want 19999", server.port)
	}
	if server.handler == nil {
		t.Error("handler should not be nil")
	}
}

func TestServer_HandleRegister_Direct(t *testing.T) {
	handler := &mockAgentHandler{}
	server := NewServer(0, handler)

	req := &omnitestv1.RegisterRequest{
		AgentId:   "test-agent",
		Hostname:  "test-host",
		MaxVusers: 1000,
	}

	resp, err := server.Register(nil, req)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if !resp.Accepted {
		t.Error("Register() should be accepted")
	}
	if resp.ControllerId != "test-controller" {
		t.Errorf("ControllerId = %q, want %q", resp.ControllerId, "test-controller")
	}
	if !handler.registerCalled {
		t.Error("HandleRegister was not called")
	}
}

func TestServer_HandleHeartbeat_Direct(t *testing.T) {
	handler := &mockAgentHandler{}
	server := NewServer(0, handler)

	req := &omnitestv1.HeartbeatRequest{
		AgentId:     "test-agent",
		Status:      omnitestv1.AgentStatus_AGENT_STATUS_IDLE,
		CpuUsage:    10.5,
		MemoryUsage: 30.0,
	}

	resp, err := server.Heartbeat(nil, req)
	if err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}
	if !resp.Acknowledged {
		t.Error("Heartbeat() should be acknowledged")
	}
	if !handler.heartbeatCalled {
		t.Error("HandleHeartbeat was not called")
	}
}

func TestServer_StartTest_NoHandler(t *testing.T) {
	handler := &mockAgentHandler{}
	server := NewServer(0, handler)

	resp, err := server.StartTest(nil, &omnitestv1.StartTestRequest{
		TestRunId:      "run-1",
		ScenarioYaml:   "test: yaml",
		AssignedVusers: 100,
	})
	if err != nil {
		t.Fatalf("StartTest() error = %v", err)
	}
	if resp.Accepted {
		t.Error("StartTest() should not be accepted when no handler is set")
	}
	if resp.ErrorMessage == "" {
		t.Error("StartTest() should have error message when no handler")
	}
}

func TestServer_StartTest_WithHandler(t *testing.T) {
	handler := &mockAgentHandler{}
	server := NewServer(0, handler)
	server.OnStartTest = func(req *omnitestv1.StartTestRequest) (*omnitestv1.StartTestResponse, error) {
		return &omnitestv1.StartTestResponse{Accepted: true}, nil
	}

	resp, err := server.StartTest(nil, &omnitestv1.StartTestRequest{TestRunId: "run-1"})
	if err != nil {
		t.Fatalf("StartTest() error = %v", err)
	}
	if !resp.Accepted {
		t.Error("StartTest() should be accepted")
	}
}

func TestServer_StopTest_NoHandler(t *testing.T) {
	handler := &mockAgentHandler{}
	server := NewServer(0, handler)

	resp, err := server.StopTest(nil, &omnitestv1.StopTestRequest{TestRunId: "run-1"})
	if err != nil {
		t.Fatalf("StopTest() error = %v", err)
	}
	if resp.Stopped {
		t.Error("StopTest() should return false when no handler")
	}
}

func TestServer_StopTest_WithHandler(t *testing.T) {
	handler := &mockAgentHandler{}
	server := NewServer(0, handler)
	server.OnStopTest = func(req *omnitestv1.StopTestRequest) (*omnitestv1.StopTestResponse, error) {
		return &omnitestv1.StopTestResponse{Stopped: true}, nil
	}

	resp, err := server.StopTest(nil, &omnitestv1.StopTestRequest{TestRunId: "run-1"})
	if err != nil {
		t.Fatalf("StopTest() error = %v", err)
	}
	if !resp.Stopped {
		t.Error("StopTest() should return true")
	}
}

func TestServer_Stop(t *testing.T) {
	handler := &mockAgentHandler{}
	server := NewServer(0, handler)
	server.Stop()
}

// --- gRPC over-the-wire tests (proto.Message 인터페이스 구현 완료) ---

func TestGRPC_OverTheWire_RegisterAndHeartbeat(t *testing.T) {
	handler := &mockAgentHandler{}
	port := 29090

	server := NewServer(port, handler)
	go func() {
		if err := server.Start(); err != nil {
			t.Logf("Server error (may be expected): %v", err)
		}
	}()
	time.Sleep(200 * time.Millisecond)
	defer server.Stop()

	conn, err := grpc.NewClient(
		"localhost:29090",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := omnitestv1.NewAgentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Register
	regResp, err := client.Register(ctx, &omnitestv1.RegisterRequest{
		AgentId:   "wire-agent-1",
		Hostname:  "wire-host",
		MaxVusers: 2000,
	})
	if err != nil {
		t.Fatalf("Register() over wire error = %v", err)
	}
	if !regResp.GetAccepted() {
		t.Error("Register() should be accepted")
	}
	if regResp.GetHeartbeatIntervalSeconds() != 10 {
		t.Errorf("HeartbeatInterval = %d, want 10", regResp.GetHeartbeatIntervalSeconds())
	}
	if !handler.registerCalled {
		t.Error("HandleRegister was not called")
	}

	// Heartbeat
	hbResp, err := client.Heartbeat(ctx, &omnitestv1.HeartbeatRequest{
		AgentId:      "wire-agent-1",
		Status:       omnitestv1.AgentStatus_AGENT_STATUS_RUNNING,
		CpuUsage:     25.5,
		MemoryUsage:  40.0,
		ActiveVusers: 500,
	})
	if err != nil {
		t.Fatalf("Heartbeat() over wire error = %v", err)
	}
	if !hbResp.GetAcknowledged() {
		t.Error("Heartbeat() should be acknowledged")
	}
	if !handler.heartbeatCalled {
		t.Error("HandleHeartbeat was not called")
	}
}

func TestGRPC_OverTheWire_StreamMetrics(t *testing.T) {
	handler := &mockAgentHandler{}
	port := 29091

	server := NewServer(port, handler)
	go func() {
		server.Start()
	}()
	time.Sleep(200 * time.Millisecond)
	defer server.Stop()

	conn, err := grpc.NewClient(
		"localhost:29091",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := omnitestv1.NewAgentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.StreamMetrics(ctx)
	if err != nil {
		t.Fatalf("StreamMetrics() error = %v", err)
	}

	// Send multiple metric reports
	for i := 0; i < 3; i++ {
		err = stream.Send(&omnitestv1.MetricReport{
			AgentId:       "wire-agent-1",
			TestRunId:     "run-1",
			TotalRequests: int64(100 * (i + 1)),
			Rps:           float64(50 * (i + 1)),
			AvgLatencyMs:  10.5,
			P99LatencyMs:  25.0,
			ActiveVusers:  500,
		})
		if err != nil {
			t.Fatalf("Send() #%d error = %v", i, err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv() error = %v", err)
	}
	if !resp.GetAcknowledged() {
		t.Error("StreamMetrics response should be acknowledged")
	}
	if !handler.metricCalled {
		t.Error("HandleMetricReport was not called")
	}
}

func TestGRPC_OverTheWire_StartTest(t *testing.T) {
	handler := &mockAgentHandler{}
	port := 29092

	server := NewServer(port, handler)
	server.OnStartTest = func(req *omnitestv1.StartTestRequest) (*omnitestv1.StartTestResponse, error) {
		if req.GetTestRunId() != "run-wire" {
			t.Errorf("TestRunId = %q, want %q", req.GetTestRunId(), "run-wire")
		}
		return &omnitestv1.StartTestResponse{Accepted: true}, nil
	}

	go func() {
		server.Start()
	}()
	time.Sleep(200 * time.Millisecond)
	defer server.Stop()

	conn, err := grpc.NewClient(
		"localhost:29092",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := omnitestv1.NewAgentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.StartTest(ctx, &omnitestv1.StartTestRequest{
		TestRunId:      "run-wire",
		ScenarioYaml:   "version: '1'\ntargets:\n  - name: test\n    base_url: http://localhost",
		AssignedVusers: 200,
	})
	if err != nil {
		t.Fatalf("StartTest() over wire error = %v", err)
	}
	if !resp.GetAccepted() {
		t.Error("StartTest() should be accepted")
	}
}
