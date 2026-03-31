// Package agent implements the OmniTest Agent mode.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/omnitest/omnitest/internal/config"
	omnigrpc "github.com/omnitest/omnitest/internal/grpc"
	"github.com/omnitest/omnitest/internal/grpc/omnitestv1"
	"github.com/omnitest/omnitest/internal/runner"
	"github.com/omnitest/omnitest/pkg/model"
)

// Config holds agent configuration.
type Config struct {
	ControllerAddr string
	ControllerHTTP string // HTTP base URL for command polling (e.g. http://controller:8080)
	Name           string
	MaxVUsers      int
	Labels         map[string]string
	LogLevel       string
}

// Agent는 Controller에 연결하여 명령을 수신하는 Agent 모드다.
type Agent struct {
	mu             sync.RWMutex
	agentID        string
	name           string
	maxVUsers      int
	labels         map[string]string
	controllerAddr string
	client         omnitestv1.AgentServiceClient
	status         string

	// 실행 중인 테스트의 cancel func
	activeCancel context.CancelFunc
	activeRunID  string

	heartbeatInterval time.Duration
	httpBaseURL       string // Controller HTTP URL for command polling
}

// New는 Agent를 생성한다.
func New(cfg Config) *Agent {
	hostname, _ := os.Hostname()
	name := cfg.Name
	if name == "" {
		name = hostname
	}

	agentID := fmt.Sprintf("agent-%s-%d", name, time.Now().UnixMilli()%10000)

	maxVUsers := cfg.MaxVUsers
	if maxVUsers <= 0 {
		maxVUsers = 1000
	}

	httpBase := cfg.ControllerHTTP
	if httpBase == "" {
		// gRPC 주소에서 HTTP URL 유추 (host:grpcPort → http://host:8080)
		httpBase = fmt.Sprintf("http://%s", cfg.ControllerAddr)
		// gRPC port를 8080으로 대체 (간단한 휴리스틱)
		if idx := len(httpBase) - 1; idx > 0 {
			for i := len(httpBase) - 1; i >= 0; i-- {
				if httpBase[i] == ':' {
					httpBase = httpBase[:i] + ":8080"
					break
				}
			}
		}
	}

	return &Agent{
		agentID:           agentID,
		name:              name,
		maxVUsers:         maxVUsers,
		labels:            cfg.Labels,
		controllerAddr:    cfg.ControllerAddr,
		httpBaseURL:       httpBase,
		status:            "idle",
		heartbeatInterval: 10 * time.Second,
	}
}

// Run은 Agent의 메인 루프를 실행한다.
func (a *Agent) Run(ctx context.Context) error {
	log.Printf("→ Connecting to controller at %s...", a.controllerAddr)

	conn, err := omnigrpc.NewClientConn(a.controllerAddr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	a.client = omnigrpc.NewAgentClient(conn)

	// Register
	resp, err := a.client.Register(ctx, &omnitestv1.RegisterRequest{
		AgentId:   a.agentID,
		Hostname:  a.name,
		MaxVusers: int32(a.maxVUsers),
		Labels:    a.labels,
	})
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}
	if !resp.GetAccepted() {
		return fmt.Errorf("registration rejected by controller")
	}

	if resp.GetHeartbeatIntervalSeconds() > 0 {
		a.heartbeatInterval = time.Duration(resp.GetHeartbeatIntervalSeconds()) * time.Second
	}

	log.Printf("→ Agent registered: %s (name: %s)", a.agentID, a.name)
	log.Printf("→ Polling for test commands from %s", a.httpBaseURL)

	// Heartbeat goroutine
	go a.heartbeatLoop(ctx)

	// 명령 폴링 루프
	a.commandPollLoop(ctx)

	log.Printf("→ Graceful shutdown: finishing active workers...")
	a.mu.RLock()
	if a.activeCancel != nil {
		a.activeCancel()
	}
	a.mu.RUnlock()

	log.Printf("→ Agent disconnected.")
	return nil
}

// commandPollLoop는 Controller HTTP API를 폴링하여 pending command를 가져온다.
func (a *Agent) commandPollLoop(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	httpClient := &http.Client{Timeout: 5 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.pollAndExecuteCommand(ctx, httpClient)
		}
	}
}

func (a *Agent) pollAndExecuteCommand(ctx context.Context, httpClient *http.Client) {
	url := fmt.Sprintf("%s/api/v1/internal/agents/%s/command", a.httpBaseURL, a.agentID)

	resp, err := httpClient.Get(url)
	if err != nil {
		// 연결 실패는 무시 (Controller 재시작 중일 수 있음)
		return
	}
	defer resp.Body.Close()

	var envelope struct {
		Data struct {
			Command *struct {
				Type          string `json:"type"`
				TestRunID     string `json:"test_run_id"`
				ScenarioYAML  string `json:"scenario_yaml"`
				AssignedVUsers int32  `json:"assigned_vusers"`
			} `json:"command"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return
	}

	cmd := envelope.Data.Command
	if cmd == nil {
		return
	}

	switch cmd.Type {
	case "start":
		log.Printf("→ Received StartTest command: run=%s, vusers=%d", cmd.TestRunID, cmd.AssignedVUsers)
		go func() {
			req := &omnitestv1.StartTestRequest{
				TestRunId:      cmd.TestRunID,
				ScenarioYaml:   cmd.ScenarioYAML,
				AssignedVusers: cmd.AssignedVUsers,
			}
			if err := a.ExecuteTest(ctx, req); err != nil {
				log.Printf("[Agent] Test execution error: %v", err)
			}
		}()
	case "stop":
		log.Printf("→ Received StopTest command: run=%s", cmd.TestRunID)
		a.StopTest(cmd.TestRunID)
	}
}

func (a *Agent) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(a.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.RLock()
			status := omnitestv1.AgentStatus_AGENT_STATUS_IDLE
			if a.status == "running" {
				status = omnitestv1.AgentStatus_AGENT_STATUS_RUNNING
			}
			a.mu.RUnlock()

			_, err := a.client.Heartbeat(ctx, &omnitestv1.HeartbeatRequest{
				AgentId:      a.agentID,
				Status:       status,
				CpuUsage:     0, // TODO: 실제 CPU 사용률
				MemoryUsage:  0, // TODO: 실제 메모리 사용률
				ActiveVusers: 0,
			})
			if err != nil {
				log.Printf("[Agent] Heartbeat error: %v", err)
			}
		}
	}
}

// ExecuteTest는 Controller로부터 받은 테스트 명령을 실행한다.
func (a *Agent) ExecuteTest(ctx context.Context, req *omnitestv1.StartTestRequest) error {
	log.Printf("→ [%s] Received test: %s (%d VUsers assigned)",
		time.Now().Format("15:04:05"), req.GetTestRunId(), req.GetAssignedVusers())

	// scenario_yaml을 config로 파싱
	cfg, err := config.LoadFromString(req.GetScenarioYaml())
	if err != nil {
		return fmt.Errorf("failed to parse scenario: %w", err)
	}

	// VUser 수 오버라이드
	if len(cfg.Scenarios) > 0 {
		cfg.Scenarios[0].VUsers = int(req.GetAssignedVusers())
	}

	a.mu.Lock()
	a.status = "running"
	a.activeRunID = req.GetTestRunId()
	testCtx, cancel := context.WithCancel(ctx)
	a.activeCancel = cancel
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.status = "idle"
		a.activeCancel = nil
		a.activeRunID = ""
		a.mu.Unlock()
		cancel()
	}()

	// 메트릭 스트리밍 설정
	stream, err := a.client.StreamMetrics(ctx)
	if err != nil {
		return fmt.Errorf("failed to open metrics stream: %w", err)
	}

	// runner.Run 실행 (Cycle 1 코드 재사용)
	log.Printf("→ [%s] Starting workers...", time.Now().Format("15:04:05"))
	_, err = runner.Run(testCtx, cfg, runner.Options{
		Quiet: true,
		OnSnapshot: func(snap model.MetricSnapshot) {
			report := &omnitestv1.MetricReport{
				AgentId:       a.agentID,
				TestRunId:     req.GetTestRunId(),
				Timestamp:     timestamppb.Now(),
				TotalRequests: snap.TotalReqs,
				TotalErrors:   snap.TotalErrors,
				Rps:           snap.RPS,
				AvgLatencyMs:  float64(snap.AvgLatency.Milliseconds()),
				P50LatencyMs:  float64(snap.P50Latency.Milliseconds()),
				P95LatencyMs:  float64(snap.P95Latency.Milliseconds()),
				P99LatencyMs:  float64(snap.P99Latency.Milliseconds()),
				ActiveVusers:  int32(snap.ActiveVUsers),
			}
			if err := stream.Send(report); err != nil {
				log.Printf("[Agent] Failed to send metric: %v", err)
			}
		},
	})

	// 스트림 닫기
	stream.CloseAndRecv()

	log.Printf("→ [%s] Test completed. Sent final metrics to controller.",
		time.Now().Format("15:04:05"))
	log.Printf("→ Waiting for test commands...")

	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}
	return nil
}

// StopTest는 현재 실행 중인 테스트를 중지한다.
func (a *Agent) StopTest(testRunID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.activeRunID == testRunID && a.activeCancel != nil {
		a.activeCancel()
		return true
	}
	return false
}
