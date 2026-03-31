// Package controller implements the OmniTest Controller server.
package controller

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	omnigrpc "github.com/omnitest/omnitest/internal/grpc"
	"github.com/omnitest/omnitest/internal/grpc/omnitestv1"
	"github.com/omnitest/omnitest/internal/store"
	"github.com/omnitest/omnitest/internal/ws"
	"github.com/omnitest/omnitest/pkg/model"
)

// Config는 Controller 설정이다.
type Config struct {
	GRPCPort    int
	HTTPPort    int
	DatabaseURL string
}

// Controller는 분산 테스트의 중앙 제어 서버다.
type Controller struct {
	config       Config
	grpcServer   *omnigrpc.Server
	httpServer   *http.Server
	wsHub        *ws.Hub
	store        *store.Store
	AgentManager *AgentManager
	Scheduler    *Scheduler
	Aggregator   *Aggregator
	startTime    time.Time
}

// New는 Controller를 생성한다.
func New(ctx context.Context, cfg Config) (*Controller, error) {
	// PostgreSQL 연결
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create store: %w", err)
	}

	wsHub := ws.NewHub()
	am := NewAgentManager(st)
	sched := NewScheduler(am)
	agg := NewAggregator(wsHub, st)

	c := &Controller{
		config:       cfg,
		wsHub:        wsHub,
		store:        st,
		AgentManager: am,
		Scheduler:    sched,
		Aggregator:   agg,
		startTime:    time.Now(),
	}

	// gRPC 서버 생성
	c.grpcServer = omnigrpc.NewServer(cfg.GRPCPort, c)

	return c, nil
}

// Store returns the underlying data store.
func (c *Controller) Store() *store.Store {
	return c.store
}

// WSHub returns the WebSocket hub.
func (c *Controller) WSHub() *ws.Hub {
	return c.wsHub
}

// StartTime returns when the controller started.
func (c *Controller) StartTime() time.Time {
	return c.startTime
}

// Start는 gRPC, HTTP 서버를 모두 시작한다.
func (c *Controller) Start(ctx context.Context, httpHandler http.Handler) error {
	// gRPC 서버 시작
	go func() {
		if err := c.grpcServer.Start(); err != nil {
			log.Printf("[Controller] gRPC server error: %v", err)
		}
	}()

	// Agent 헬스체크 루프
	go c.AgentManager.HealthCheckLoop(ctx)

	// HTTP 서버 시작 (REST API + WebSocket)
	c.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", c.config.HTTPPort),
		Handler: httpHandler,
	}
	log.Printf("[Controller] HTTP server listening on :%d", c.config.HTTPPort)
	return c.httpServer.ListenAndServe()
}

// Shutdown은 graceful shutdown을 수행한다.
func (c *Controller) Shutdown(ctx context.Context) error {
	c.grpcServer.Stop()
	if c.httpServer != nil {
		_ = c.httpServer.Shutdown(ctx)
	}
	c.store.Close()
	return nil
}

// --- AgentHandler interface implementation ---

// HandleRegister implements grpc.AgentHandler.
func (c *Controller) HandleRegister(req *omnitestv1.RegisterRequest) (*omnitestv1.RegisterResponse, error) {
	agent := &model.AgentInfo{
		AgentID:      req.GetAgentId(),
		Hostname:     req.GetHostname(),
		MaxVUsers:    int(req.GetMaxVusers()),
		Labels:       req.GetLabels(),
		Status:       model.AgentStatusIdle,
		RegisteredAt: time.Now(),
	}

	c.AgentManager.Register(agent)
	log.Printf("[Controller] Agent registered: %s (%s)", agent.AgentID, agent.Hostname)

	c.wsHub.BroadcastEvent("agent_registered", agent)

	return &omnitestv1.RegisterResponse{
		Accepted:                 true,
		ControllerId:             "controller-1",
		HeartbeatIntervalSeconds: 10,
	}, nil
}

// HandleHeartbeat implements grpc.AgentHandler.
func (c *Controller) HandleHeartbeat(req *omnitestv1.HeartbeatRequest) error {
	status := model.AgentStatusIdle
	switch req.GetStatus() {
	case omnitestv1.AgentStatus_AGENT_STATUS_RUNNING:
		status = model.AgentStatusRunning
	case omnitestv1.AgentStatus_AGENT_STATUS_ERROR:
		status = model.AgentStatusError
	}

	c.AgentManager.Heartbeat(req.GetAgentId(), status, req.GetCpuUsage(), req.GetMemoryUsage(), int(req.GetActiveVusers()))
	return nil
}

// HandleMetricReport implements grpc.AgentHandler.
func (c *Controller) HandleMetricReport(report *omnitestv1.MetricReport) error {
	return c.Aggregator.OnMetricReport(report)
}
