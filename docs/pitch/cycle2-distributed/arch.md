# Architecture Guide: Cycle 2 - 분산 아키텍처

> Builder가 즉시 구현에 착수할 수 있는 기술 명세
> Cycle 1 MVP Core Engine 위에 Controller-Agent 분산 아키텍처를 구축한다.

---

## 1. 프로젝트 구조

### 1.1 Cycle 1 기존 구조 (유지)

```
omnitest/
├── cmd/omnitest/main.go              # CLI 엔트리포인트 (run, validate, version + 신규: controller, agent)
├── internal/
│   ├── config/config.go              # YAML 파싱 + 검증 + 환경변수 치환
│   ├── runner/runner.go              # 테스트 오케스트레이션 (Standalone + Agent 모드 공용)
│   ├── worker/worker.go              # goroutine 기반 VUser 워커 풀
│   ├── http/client.go                # net/http 래퍼
│   ├── metrics/collector.go          # HDR Histogram 기반 메트릭 수집기
│   ├── output/printer.go             # 터미널 실시간 출력
│   ├── threshold/threshold.go        # Threshold 평가
│   └── report/
│       ├── json.go                   # JSON 리포트
│       └── html.go                   # HTML 리포트
├── pkg/model/model.go                # 공개 데이터 모델
└── testdata/                         # 예제 시나리오 파일
```

### 1.2 Cycle 2 추가 패키지

```
omnitest/
├── cmd/omnitest/main.go              # + controller, agent 서브커맨드 추가
├── internal/
│   ├── controller/                   # [신규] Controller 서버
│   │   ├── server.go                 # Controller 메인 서버 (gRPC + HTTP + WS 통합)
│   │   ├── agent_manager.go          # Agent 등록/헬스체크/상태 관리
│   │   ├── scheduler.go              # 테스트 실행 시 Agent별 VUser 분배
│   │   └── aggregator.go             # Agent별 메트릭 병합 (HDR Histogram Merge)
│   ├── agent/                        # [신규] Agent 모드
│   │   ├── agent.go                  # Agent 메인 (Controller 연결, 명령 수신)
│   │   └── reconnect.go             # Exponential backoff 재연결 로직
│   ├── grpc/                         # [신규] gRPC 서버/클라이언트
│   │   ├── server.go                 # gRPC 서버 구현 (AgentService)
│   │   ├── client.go                 # gRPC 클라이언트 (Agent → Controller)
│   │   └── omnitestv1/               # protoc 생성 코드 (agent.pb.go, agent_grpc.pb.go)
│   ├── api/                          # [신규] REST API 핸들러
│   │   ├── server.go                 # HTTP 서버 + 라우팅 (net/http ServeMux)
│   │   ├── middleware.go             # 공통 미들웨어 (request_id, logging, CORS)
│   │   ├── response.go              # Envelope 응답 헬퍼 (JSON, Error, List)
│   │   ├── test_handler.go           # /api/v1/tests/* 핸들러
│   │   ├── result_handler.go         # /api/v1/results/* 핸들러
│   │   ├── agent_handler.go          # /api/v1/agents/* 핸들러
│   │   └── system_handler.go         # /api/v1/health, /api/v1/version
│   ├── store/                        # [신규] PostgreSQL 스토어
│   │   ├── store.go                  # DB 연결 관리 (pgx pool)
│   │   ├── test_store.go             # tests 테이블 CRUD
│   │   ├── run_store.go              # test_runs 테이블 CRUD
│   │   ├── agent_store.go            # agents 테이블 CRUD
│   │   └── metric_store.go           # metrics 테이블 INSERT/SELECT
│   ├── ws/                           # [신규] WebSocket 허브
│   │   ├── hub.go                    # 연결 관리 + 브로드캐스트
│   │   └── client.go                 # 개별 WebSocket 클라이언트 goroutine
│   ├── runner/runner.go              # [수정] Agent 모드에서 재사용
│   ├── worker/worker.go              # [유지] 변경 없음
│   ├── http/client.go                # [유지] 변경 없음
│   └── metrics/collector.go          # [유지] 변경 없음
├── proto/omnitest/v1/
│   └── agent.proto                   # [이미 존재] gRPC 서비스 정의
├── migrations/
│   └── 001_init.sql                  # [이미 존재] 초기 DB 스키마
├── docker/
│   ├── Dockerfile                    # [이미 존재] 멀티스테이지 빌드
│   └── postgres/init.sql             # [이미 존재] DB 초기화
├── web/                              # [신규] React 웹 대시보드
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   ├── index.html
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── api/                      # API 클라이언트
│       │   ├── client.ts             # fetch 래퍼 + 에러 핸들링
│       │   └── hooks.ts              # TanStack Query 커스텀 훅
│       ├── ws/                       # WebSocket 연결
│       │   └── useWebSocket.ts       # WebSocket 커스텀 훅
│       ├── store/                    # Zustand 스토어
│       │   └── useStore.ts
│       ├── pages/
│       │   ├── TestListPage.tsx      # 테스트 목록/관리
│       │   ├── TestRunPage.tsx       # 실시간 메트릭 차트
│       │   └── AgentsPage.tsx        # 에이전트 상태 모니터링
│       ├── components/
│       │   ├── Layout.tsx            # 공통 레이아웃 + 네비게이션
│       │   ├── MetricsChart.tsx      # RPS/Latency/Error 실시간 차트
│       │   ├── LatencyDistribution.tsx # P50/P95/P99 바 차트
│       │   ├── AgentTable.tsx        # 에이전트 테이블
│       │   ├── TestTable.tsx         # 테스트 목록 테이블
│       │   └── StatusBadge.tsx       # 상태 배지 컴포넌트
│       └── lib/
│           └── utils.ts              # 포맷팅, 색상 유틸
├── docker-compose.yml                # [이미 존재] 풀스택 배포
├── buf.yaml                          # [이미 존재] proto 린팅
├── go.mod
└── Makefile                          # [수정] proto, dashboard, docker 타겟 추가
```

### 1.3 패키지 의존성 그래프

```
cmd/omnitest
  ├── internal/controller (controller 서브커맨드)
  │     ├── internal/grpc (gRPC 서버)
  │     ├── internal/api (REST API)
  │     ├── internal/ws (WebSocket)
  │     ├── internal/store (PostgreSQL)
  │     └── internal/controller/aggregator
  ├── internal/agent (agent 서브커맨드)
  │     ├── internal/grpc (gRPC 클라이언트)
  │     ├── internal/runner ← Cycle 1 재사용
  │     │     ├── internal/worker
  │     │     ├── internal/http
  │     │     └── internal/metrics
  │     └── internal/config
  └── internal/runner (run 서브커맨드, Cycle 1 그대로)
```

---

## 2. gRPC 서비스 정의

### 2.1 proto 파일 (이미 존재: `proto/omnitest/v1/agent.proto`)

현재 proto 파일이 validator에 의해 이미 세팅되어 있다. 구조 요약:

```protobuf
service AgentService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  rpc StartTest(StartTestRequest) returns (StartTestResponse);
  rpc StopTest(StopTestRequest) returns (StopTestResponse);
  rpc StreamMetrics(stream MetricReport) returns (StreamMetricsResponse);
}
```

### 2.2 메시지 타입 상세

| 메시지 | 방향 | 용도 |
|--------|------|------|
| `RegisterRequest` | Agent → Controller | agent_id, hostname, max_vusers, labels |
| `RegisterResponse` | Controller → Agent | accepted, controller_id, heartbeat_interval_seconds |
| `HeartbeatRequest` | Agent → Controller | agent_id, status(enum), cpu_usage, memory_usage, active_vusers |
| `HeartbeatResponse` | Controller → Agent | acknowledged |
| `StartTestRequest` | Controller → Agent | test_run_id, scenario_yaml, assigned_vusers |
| `StartTestResponse` | Agent → Controller | accepted, error_message |
| `StopTestRequest` | Controller → Agent | test_run_id |
| `StopTestResponse` | Agent → Controller | stopped |
| `MetricReport` | Agent → Controller (stream) | agent_id, test_run_id, timestamp, rps, latencies, active_vusers |
| `StreamMetricsResponse` | Controller → Agent | acknowledged |

### 2.3 양방향 스트리밍 패턴

`StreamMetrics`는 **클라이언트 스트리밍** (Agent가 1초 간격으로 MetricReport를 push)이다.

```
Agent                                    Controller
  │                                          │
  │──── MetricReport (t=1s) ──────────────>│
  │──── MetricReport (t=2s) ──────────────>│
  │──── MetricReport (t=3s) ──────────────>│
  │           ...                            │
  │──── MetricReport (t=final) ───────────>│
  │<─── StreamMetricsResponse ─────────────│
```

**백프레셔 전략**:
- Agent 측: 로컬 ring buffer (최근 60초분). Controller 수신 불가 시 오래된 데이터 드롭
- gRPC 기본 flow control 활용 (윈도우 사이즈 기본값 64KB)

### 2.4 proto 코드 생성

```bash
# buf.yaml 기반 코드 생성
buf generate

# 생성 위치: internal/grpc/omnitestv1/
# - agent.pb.go        (메시지 타입)
# - agent_grpc.pb.go   (서비스 인터페이스)
```

**buf.gen.yaml** (추가 필요):
```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: internal/grpc/omnitestv1
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: internal/grpc/omnitestv1
    opt: paths=source_relative
```

---

## 3. 핵심 데이터 모델

### 3.1 Go 구조체 (Cycle 2 추가분)

> `pkg/model/model.go`에 추가. Cycle 1 기존 구조체는 그대로 유지한다.

```go
// ─── Cycle 2: Controller 모델 ───

// AgentInfo는 등록된 Agent의 상태 정보다.
type AgentInfo struct {
    AgentID      string            `json:"agent_id"`
    Hostname     string            `json:"hostname"`
    MaxVUsers    int               `json:"max_vusers"`
    Labels       map[string]string `json:"labels"`
    Status       AgentStatus       `json:"status"`        // idle, running, error, offline
    ActiveVUsers int               `json:"active_vusers"`
    CPUUsage     float64           `json:"cpu_usage"`
    MemoryUsage  float64           `json:"memory_usage"`
    LastHeartbeat time.Time        `json:"last_heartbeat"`
    RegisteredAt  time.Time        `json:"registered_at"`
}

// AgentStatus는 에이전트 상태를 나타낸다.
type AgentStatus string

const (
    AgentStatusIdle    AgentStatus = "idle"
    AgentStatusRunning AgentStatus = "running"
    AgentStatusError   AgentStatus = "error"
    AgentStatusOffline AgentStatus = "offline"
)

// TestDefinition은 DB에 저장되는 테스트 정의다.
type TestDefinition struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    ScenarioYAML string    `json:"scenario_yaml"`
    CreatedBy    string    `json:"created_by,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

// TestRun은 개별 테스트 실행 레코드다.
type TestRun struct {
    ID               string          `json:"id"`
    TestID           string          `json:"test_id"`
    Status           TestRunStatus   `json:"status"`       // pending, running, completed, failed, stopped
    TotalVUsers      int             `json:"total_vusers"`
    DurationSeconds  int             `json:"duration_seconds"`
    StartedAt        *time.Time      `json:"started_at,omitempty"`
    FinishedAt       *time.Time      `json:"finished_at,omitempty"`
    ResultSummary    *TestResult     `json:"result_summary,omitempty"`    // JSONB
    ThresholdResults []ThresholdResult `json:"threshold_results,omitempty"` // JSONB
    CreatedAt        time.Time       `json:"created_at"`
}

// TestRunStatus는 테스트 실행 상태다.
type TestRunStatus string

const (
    TestRunPending   TestRunStatus = "pending"
    TestRunRunning   TestRunStatus = "running"
    TestRunCompleted TestRunStatus = "completed"
    TestRunFailed    TestRunStatus = "failed"
    TestRunStopped   TestRunStatus = "stopped"
)

// AgentAssignment은 테스트 실행 시 Agent에 할당된 VUser 정보다.
type AgentAssignment struct {
    AgentID       string `json:"agent_id"`
    AssignedVUsers int   `json:"assigned_vusers"`
}

// AggregatedMetrics는 Controller가 Agent별 메트릭을 병합한 결과다.
type AggregatedMetrics struct {
    TestRunID    string                    `json:"test_run_id"`
    Timestamp    time.Time                 `json:"timestamp"`
    TotalRPS     float64                   `json:"total_rps"`
    AvgLatencyMs float64                   `json:"avg_latency_ms"`
    P50LatencyMs float64                   `json:"p50_latency_ms"`
    P95LatencyMs float64                   `json:"p95_latency_ms"`
    P99LatencyMs float64                   `json:"p99_latency_ms"`
    TotalReqs    int64                     `json:"total_requests"`
    TotalErrors  int64                     `json:"total_errors"`
    ActiveVUsers int                       `json:"active_vusers"`
    PerAgent     map[string]*MetricSnapshot `json:"per_agent"` // agent_id → snapshot
}
```

### 3.2 DB 스키마 (이미 존재: `migrations/001_init.sql`)

| 테이블 | 역할 | Cycle 1 model.go 관계 |
|--------|------|----------------------|
| `agents` | Agent 등록/상태 | → `AgentInfo` |
| `tests` | 테스트 시나리오 정의 | → `TestDefinition` (scenario_yaml은 Cycle 1의 `TestConfig` YAML) |
| `test_runs` | 테스트 실행 이력 | → `TestRun` (result_summary JSONB → Cycle 1의 `TestResult`) |
| `metrics` | 시계열 메트릭 | → `MetricSnapshot` (agent별, 1초 간격) |

**Cycle 1 `TestConfig`는 DB에 YAML 텍스트로 저장**되며, 실행 시 `config.LoadFromString()`으로 파싱한다.

**Cycle 1 `TestResult`는 `test_runs.result_summary`에 JSONB로 저장**된다.

---

## 4. 패키지별 API 명세

### 4.1 `internal/controller/server.go`

```go
// Controller는 분산 테스트의 중앙 제어 서버다.
type Controller struct {
    grpcServer   *grpc.Server
    apiServer    *api.Server
    wsHub        *ws.Hub
    store        *store.Store
    agentManager *AgentManager
    scheduler    *Scheduler
    aggregator   *Aggregator
}

// Config는 Controller 설정이다.
type Config struct {
    GRPCPort    int
    HTTPPort    int
    WSPort      int
    DatabaseURL string
}

// New는 Controller를 생성한다.
func New(cfg Config) (*Controller, error)

// Start는 gRPC, HTTP, WebSocket 서버를 모두 시작한다.
// 각 서버는 별도 goroutine에서 실행된다.
func (c *Controller) Start(ctx context.Context) error

// Shutdown은 graceful shutdown을 수행한다.
func (c *Controller) Shutdown(ctx context.Context) error
```

### 4.2 `internal/controller/agent_manager.go`

```go
// AgentManager는 연결된 Agent를 관리한다.
type AgentManager struct {
    agents map[string]*ConnectedAgent // agent_id → agent
    mu     sync.RWMutex
    store  *store.Store
}

// ConnectedAgent는 gRPC 스트림이 연결된 Agent다.
type ConnectedAgent struct {
    Info       model.AgentInfo
    Stream     omnitestv1.AgentService_StreamMetricsServer // nil if not streaming
    cancelFunc context.CancelFunc                          // StartTest 취소용
}

// Register는 Agent를 등록한다. DB에도 저장한다.
func (m *AgentManager) Register(req *omnitestv1.RegisterRequest) (*omnitestv1.RegisterResponse, error)

// Heartbeat는 Agent 헬스 정보를 업데이트한다.
func (m *AgentManager) Heartbeat(req *omnitestv1.HeartbeatRequest) error

// OnlineAgents는 현재 온라인 Agent 목록을 반환한다.
func (m *AgentManager) OnlineAgents() []model.AgentInfo

// MarkOffline은 헬스체크 타임아웃된 Agent를 offline으로 전환한다.
// 10초 간격 goroutine에서 호출.
func (m *AgentManager) MarkOffline()
```

### 4.3 `internal/controller/scheduler.go`

```go
// Scheduler는 테스트 실행 시 Agent에 VUser를 분배한다.
type Scheduler struct {
    agentManager *AgentManager
}

// Allocate는 온라인 Agent들에게 총 VUser를 균등 분배한다.
// 반환: agent_id → assigned_vusers 맵
// 남은 VUser는 첫 번째 Agent에 추가 (round-robin remainder).
func (s *Scheduler) Allocate(totalVUsers int) ([]model.AgentAssignment, error)

// StartTest는 각 Agent에 gRPC StartTest RPC를 호출한다.
func (s *Scheduler) StartTest(ctx context.Context, run *model.TestRun, assignments []model.AgentAssignment, scenarioYAML string) error

// StopTest는 각 Agent에 gRPC StopTest RPC를 호출한다.
func (s *Scheduler) StopTest(ctx context.Context, testRunID string) error
```

### 4.4 `internal/controller/aggregator.go`

```go
// Aggregator는 Agent별 메트릭을 실시간 병합한다.
type Aggregator struct {
    metrics map[string]map[string]*latestMetric // testRunID → agentID → latest
    mu      sync.RWMutex
    wsHub   *ws.Hub
    store   *store.Store
}

// latestMetric은 Agent의 최신 메트릭 보고다.
type latestMetric struct {
    Report    *omnitestv1.MetricReport
    UpdatedAt time.Time
}

// OnMetricReport는 Agent에서 MetricReport를 수신했을 때 호출된다.
// 1) 로컬 캐시 업데이트, 2) 전체 집계, 3) WebSocket 브로드캐스트, 4) DB 저장
func (a *Aggregator) OnMetricReport(report *omnitestv1.MetricReport) error

// Aggregate는 특정 testRunID의 전체 Agent 메트릭을 병합한다.
// RPS: 합산, Latency: 가중 평균 (요청 수 기반), VUsers: 합산
func (a *Aggregator) Aggregate(testRunID string) *model.AggregatedMetrics

// Cleanup은 완료된 testRun의 메트릭 캐시를 정리한다.
func (a *Aggregator) Cleanup(testRunID string)
```

**메트릭 병합 알고리즘**:
```
RPS = sum(agent.rps)
TotalRequests = sum(agent.total_requests)
TotalErrors = sum(agent.total_errors)
ActiveVUsers = sum(agent.active_vusers)
AvgLatency = sum(agent.avg_latency * agent.total_requests) / sum(agent.total_requests)
P50/P95/P99 = 가중 평균 (요청 수 비례) — HDR Histogram Merge는 원시 히스토그램 전송 필요.
             MVP에서는 가중 평균으로 근사. 정확한 병합은 Cycle 3에서 원시 히스토그램 스트리밍 도입 시 적용.
```

### 4.5 `internal/agent/agent.go`

```go
// Agent는 Controller에 연결하여 명령을 수신하는 Agent 모드다.
type Agent struct {
    agentID      string
    name         string
    maxVUsers    int
    labels       map[string]string
    controllerAddr string
    grpcClient   *grpc.ClientConn
    client       omnitestv1.AgentServiceClient
}

// Config는 Agent 설정이다.
type Config struct {
    ControllerAddr string
    Name           string
    MaxVUsers      int
    Labels         map[string]string
    LogLevel       string
}

// New는 Agent를 생성한다. agent_id는 nanoid로 자동 생성.
func New(cfg Config) *Agent

// Run은 Agent의 메인 루프를 실행한다.
// 1) Controller에 gRPC 연결
// 2) Register RPC 호출
// 3) Heartbeat goroutine 시작 (heartbeat_interval_seconds 간격)
// 4) 명령 대기 루프 (StartTest가 오면 runner.Run 실행)
// ctx 취소 시 graceful shutdown.
func (a *Agent) Run(ctx context.Context) error
```

**Agent가 StartTest를 수신했을 때의 흐름**:

```go
// Agent 내부에서 Cycle 1의 runner.Run()을 재사용한다.
func (a *Agent) executeTest(ctx context.Context, req *StartTestRequest) error {
    // 1) scenario_yaml을 config.LoadFromString()으로 파싱
    cfg, err := config.LoadFromString(req.ScenarioYaml)

    // 2) assigned_vusers로 VUser 수 오버라이드
    cfg.Scenarios[0].VUsers = int(req.AssignedVusers)

    // 3) runner.Run() 실행 (Cycle 1 코드 그대로)
    //    단, Options.Quiet = true (터미널 출력 불필요)
    result, err := runner.Run(ctx, cfg, runner.Options{Quiet: true})

    // 4) 실행 중 1초 간격으로 MetricReport 스트리밍
    //    → collector.Snapshot()을 MetricReport protobuf로 변환하여 전송

    // 5) 완료 시 최종 MetricReport 전송
    return nil
}
```

### 4.6 `internal/agent/reconnect.go`

```go
// Reconnector는 exponential backoff로 Controller에 재연결한다.
type Reconnector struct {
    addr       string
    minDelay   time.Duration // 1s
    maxDelay   time.Duration // 30s
    multiplier float64       // 2.0
}

// Connect는 gRPC 연결을 시도하고, 실패 시 backoff로 재시도한다.
// ctx 취소까지 무한 재시도.
func (r *Reconnector) Connect(ctx context.Context) (*grpc.ClientConn, error)
```

### 4.7 `internal/grpc/server.go`

```go
// Server는 AgentService gRPC 서버 구현이다.
type Server struct {
    omnitestv1.UnimplementedAgentServiceServer
    agentManager *controller.AgentManager
    aggregator   *controller.Aggregator
    scheduler    *controller.Scheduler
}

func (s *Server) Register(ctx context.Context, req *omnitestv1.RegisterRequest) (*omnitestv1.RegisterResponse, error)
func (s *Server) Heartbeat(ctx context.Context, req *omnitestv1.HeartbeatRequest) (*omnitestv1.HeartbeatResponse, error)
func (s *Server) StartTest(ctx context.Context, req *omnitestv1.StartTestRequest) (*omnitestv1.StartTestResponse, error)
func (s *Server) StopTest(ctx context.Context, req *omnitestv1.StopTestRequest) (*omnitestv1.StopTestResponse, error)
func (s *Server) StreamMetrics(stream omnitestv1.AgentService_StreamMetricsServer) error
```

**gRPC 서버 설정**:
```go
grpcServer := grpc.NewServer(
    grpc.KeepaliveParams(keepalive.ServerParameters{
        Time:    10 * time.Second,  // 10초마다 keepalive ping
        Timeout: 5 * time.Second,
    }),
    grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
        MinTime:             5 * time.Second,
        PermitWithoutStream: true,
    }),
)
```

### 4.8 `internal/grpc/client.go`

```go
// NewClientConn은 Agent에서 Controller로의 gRPC 연결을 생성한다.
func NewClientConn(addr string) (*grpc.ClientConn, error)
```

```go
conn, err := grpc.NewClient(addr,
    grpc.WithTransportCredentials(insecure.NewCredentials()), // Cycle 2: 인증 없음
    grpc.WithKeepaliveParams(keepalive.ClientParameters{
        Time:                10 * time.Second,
        Timeout:             5 * time.Second,
        PermitWithoutStream: true,
    }),
)
```

### 4.9 `internal/api/server.go`

```go
// Server는 REST API HTTP 서버다.
type Server struct {
    mux          *http.ServeMux
    store        *store.Store
    agentManager *controller.AgentManager
    scheduler    *controller.Scheduler
    wsHub        *ws.Hub
}

// New는 REST API 서버를 생성하고 라우트를 등록한다.
func New(store *store.Store, am *controller.AgentManager, sched *controller.Scheduler, hub *ws.Hub) *Server

// ListenAndServe는 HTTP 서버를 시작한다.
func (s *Server) ListenAndServe(addr string) error
```

### 4.10 `internal/store/store.go`

```go
// Store는 PostgreSQL 데이터 스토어다.
type Store struct {
    pool *pgxpool.Pool
}

// New는 pgx 커넥션 풀을 생성한다.
func New(ctx context.Context, databaseURL string) (*Store, error)

// Close는 커넥션 풀을 닫는다.
func (s *Store) Close()
```

**pgx 풀 설정**:
```go
poolConfig, _ := pgxpool.ParseConfig(databaseURL)
poolConfig.MaxConns = 20          // Controller 동시 요청 처리
poolConfig.MinConns = 5
poolConfig.MaxConnLifetime = 30 * time.Minute
poolConfig.MaxConnIdleTime = 5 * time.Minute
poolConfig.HealthCheckPeriod = 30 * time.Second
```

### 4.11 `internal/ws/hub.go`

```go
// Hub는 WebSocket 연결 허브다.
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

// NewHub는 새 WebSocket 허브를 생성한다.
func NewHub() *Hub

// Run은 허브의 메인 이벤트 루프를 실행한다.
func (h *Hub) Run(ctx context.Context)

// BroadcastMetrics는 집계된 메트릭을 모든 클라이언트에 브로드캐스트한다.
func (h *Hub) BroadcastMetrics(data *model.AggregatedMetrics)

// BroadcastEvent는 시스템 이벤트(테스트 시작/완료 등)를 브로드캐스트한다.
func (h *Hub) BroadcastEvent(eventType string, payload any)

// HandleWebSocket은 HTTP 요청을 WebSocket으로 업그레이드한다.
// gorilla/websocket 사용.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request)
```

### 4.12 `internal/ws/client.go`

```go
// Client는 개별 WebSocket 연결이다.
type Client struct {
    hub     *Hub
    conn    *websocket.Conn
    send    chan []byte // 버퍼드 채널 (256)
    filters map[string]string // e.g. {"test_run_id": "xxx"} 필터링용
}

// ReadPump은 클라이언트로부터 메시지를 읽는 goroutine.
// 30초 하트비트(pong) 타임아웃.
func (c *Client) ReadPump()

// WritePump은 클라이언트에 메시지를 쓰는 goroutine.
// 30초 간격 ping 전송.
func (c *Client) WritePump()
```

---

## 5. REST API 상세 명세

### 5.1 공통 Envelope 형식

```go
// internal/api/response.go

// Envelope은 모든 API 응답의 wrapper다.
type Envelope struct {
    Data  any    `json:"data,omitempty"`
    Error *Error `json:"error,omitempty"`
    Meta  Meta   `json:"meta"`
}

type Error struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

type Meta struct {
    RequestID string `json:"request_id"`
    Total     int    `json:"total,omitempty"`
    Page      int    `json:"page,omitempty"`
    PerPage   int    `json:"per_page,omitempty"`
}

// JSON은 성공 응답을 반환한다.
func JSON(w http.ResponseWriter, status int, data any, requestID string)

// JSONList는 페이징된 목록 응답을 반환한다.
func JSONList(w http.ResponseWriter, data any, total, page, perPage int, requestID string)

// JSONError는 에러 응답을 반환한다.
func JSONError(w http.ResponseWriter, status int, code, message, details, requestID string)
```

### 5.2 엔드포인트별 Request/Response

#### 테스트 관리

**GET /api/v1/tests**
```
Query: ?page=1&per_page=20&status=running
Response 200:
{
  "data": [TestDefinition...],
  "meta": {"total": 42, "page": 1, "per_page": 20, "request_id": "req-xxx"}
}
```

**POST /api/v1/tests**
```
Request Body:
{
  "name": "user-api-load",
  "scenario_yaml": "version: '1'\ntargets:\n  - name: api\n    base_url: ..."
}
Response 201:
{
  "data": TestDefinition,
  "meta": {"request_id": "req-xxx"}
}
Errors: 400 INVALID_SCENARIO (YAML 파싱 실패)
```

**GET /api/v1/tests/{id}**
```
Response 200: {"data": TestDefinition, "meta": {...}}
Errors: 404 TEST_NOT_FOUND
```

**PUT /api/v1/tests/{id}**
```
Request Body: {"name": "...", "scenario_yaml": "..."}
Response 200: {"data": TestDefinition, "meta": {...}}
Errors: 404 TEST_NOT_FOUND, 400 INVALID_SCENARIO
```

**DELETE /api/v1/tests/{id}**
```
Response 200: {"data": {"deleted": true}, "meta": {...}}
Errors: 404 TEST_NOT_FOUND
```

#### 테스트 실행

**POST /api/v1/tests/{id}/run**
```
Request Body (optional):
{
  "vusers_override": 500,
  "duration_override": "10m"
}
Response 201:
{
  "data": TestRun,
  "meta": {"request_id": "req-xxx"}
}
Errors: 404 TEST_NOT_FOUND, 409 TEST_ALREADY_RUNNING, 409 NO_AGENTS_AVAILABLE
```

**POST /api/v1/tests/{id}/stop**
```
Response 200: {"data": {"stopped": true}, "meta": {...}}
Errors: 404 TEST_NOT_FOUND, 400 INVALID_PARAMETER (테스트가 실행 중이 아님)
```

**GET /api/v1/tests/{id}/status**
```
Response 200:
{
  "data": {
    "test_run": TestRun,
    "agents": [AgentAssignment...],
    "current_metrics": AggregatedMetrics  // running일 때만
  },
  "meta": {...}
}
```

#### 테스트 결과

**GET /api/v1/tests/{id}/results**
```
Query: ?page=1&per_page=10
Response 200: {"data": [TestRun...], "meta": {...}}
```

**GET /api/v1/results/{id}**
```
Response 200: {"data": TestRun (result_summary 포함), "meta": {...}}
Errors: 404 RESULT_NOT_FOUND
```

**GET /api/v1/results/{id}/metrics**
```
Query: ?interval=1s (기본: 1초 간격 시계열)
Response 200:
{
  "data": {
    "aggregated": [AggregatedMetrics...],
    "per_agent": {"agent-1": [MetricSnapshot...], ...}
  },
  "meta": {...}
}
```

**GET /api/v1/results/{id}/report**
```
Query: ?format=json|html (기본: json)
Response 200: JSON 또는 HTML 리포트 (Content-Type에 따라)
```

#### 에이전트

**GET /api/v1/agents**
```
Response 200: {"data": [AgentInfo...], "meta": {...}}
```

**GET /api/v1/agents/{id}**
```
Response 200: {"data": AgentInfo, "meta": {...}}
Errors: 404 AGENT_NOT_FOUND
```

**DELETE /api/v1/agents/{id}**
```
Response 200: {"data": {"disconnected": true}, "meta": {...}}
Errors: 404 AGENT_NOT_FOUND
```

#### 시스템

**GET /api/v1/health**
```
Response 200:
{
  "data": {
    "status": "ok",
    "db": "connected",
    "agents_online": 3,
    "uptime_seconds": 3600
  },
  "meta": {...}
}
```

**GET /api/v1/version**
```
Response 200:
{
  "data": {"version": "0.2.0", "build": "abc123"},
  "meta": {...}
}
```

### 5.3 에러 코드 매핑

| HTTP | 에러 코드 | 트리거 조건 |
|------|-----------|-------------|
| 400 | `INVALID_SCENARIO` | `config.LoadFromString()` 실패 |
| 400 | `INVALID_PARAMETER` | 쿼리 파라미터 검증 실패, 또는 잘못된 상태에서의 요청 |
| 404 | `TEST_NOT_FOUND` | `store.GetTest()` → pgx.ErrNoRows |
| 404 | `RESULT_NOT_FOUND` | `store.GetTestRun()` → pgx.ErrNoRows |
| 404 | `AGENT_NOT_FOUND` | `agentManager.Get()` → nil |
| 409 | `TEST_ALREADY_RUNNING` | 해당 test_id에 status=running인 test_run 존재 |
| 409 | `NO_AGENTS_AVAILABLE` | `agentManager.OnlineAgents()` 결과가 빈 목록 |
| 500 | `INTERNAL_ERROR` | 예상치 못한 에러 (DB 오류 등) |
| 503 | `AGENT_UNAVAILABLE` | gRPC StartTest/StopTest RPC 실패 |

### 5.4 미들웨어 체인

```go
// internal/api/middleware.go

// RequestID는 각 요청에 고유 ID를 부여한다.
// 헤더: X-Request-ID. 없으면 nanoid로 생성.
func RequestID(next http.Handler) http.Handler

// Logger는 요청/응답을 로깅한다.
func Logger(next http.Handler) http.Handler

// CORS는 웹 대시보드 접근을 위한 CORS 헤더를 설정한다.
func CORS(next http.Handler) http.Handler

// 적용 순서: CORS → RequestID → Logger → Handler
```

---

## 6. 실행 흐름 시퀀스

### 6.1 Controller 시작 → Agent 등록 → 테스트 실행 전체 흐름

```
1. Controller 시작
   $ omnitest controller --grpc-port=9090 --http-port=8080 --db-url=postgres://...
   │
   ├─► store.New(dbURL)                    // PostgreSQL 연결 풀 생성
   ├─► controller.New(cfg)                 // AgentManager, Scheduler, Aggregator 생성
   ├─► grpcServer.Serve(:9090)             // gRPC 서버 시작 [goroutine]
   ├─► apiServer.ListenAndServe(:8080)     // REST API 시작 [goroutine]
   ├─► wsHub.Run()                         // WebSocket 허브 시작 [goroutine]
   └─► agentManager.HealthCheckLoop()      // 10초 간격 헬스체크 [goroutine]

2. Agent 등록
   $ omnitest agent --controller=host:9090 --name=agent-1
   │
   ├─► reconnector.Connect(addr)           // gRPC 연결 (backoff 포함)
   ├─► client.Register(RegisterRequest)    // Agent 등록
   │     Controller: agentManager.Register() → DB INSERT → ws.BroadcastEvent("agent_registered")
   ├─► [goroutine] heartbeat ticker        // 10초 간격 Heartbeat RPC
   └─► 명령 대기 (StartTest 수신 대기)

3. 테스트 생성 (REST API)
   POST /api/v1/tests  {"name": "load-test", "scenario_yaml": "..."}
   │
   ├─► config.LoadFromString(yaml)         // YAML 검증
   ├─► store.CreateTest(name, yaml)        // DB INSERT → tests 테이블
   └─► Response 201: TestDefinition

4. 테스트 실행
   POST /api/v1/tests/{id}/run  {"vusers_override": 300}
   │
   ├─► store.GetTest(id)                   // 테스트 조회
   ├─► scheduler.Allocate(300)             // Agent 3대 → 각 100 VUsers
   ├─► store.CreateTestRun(testID, ...)    // DB INSERT → test_runs (status=pending)
   ├─► scheduler.StartTest(run, assignments, yaml)
   │     │
   │     ├─► [Agent-1] gRPC StartTest(run_id, yaml, 100)
   │     │     Agent-1: config.LoadFromString → runner.Run(ctx, cfg, quiet)
   │     │     Agent-1: [goroutine] StreamMetrics → 1초마다 MetricReport 전송
   │     │
   │     ├─► [Agent-2] gRPC StartTest(run_id, yaml, 100)
   │     └─► [Agent-3] gRPC StartTest(run_id, yaml, 100)
   │
   ├─► store.UpdateTestRun(status=running) // DB UPDATE
   └─► ws.BroadcastEvent("test_started")

5. 실시간 메트릭 스트리밍
   Agent-1,2,3 ──[MetricReport]──> Controller (1초 간격)
   │
   ├─► aggregator.OnMetricReport(report)
   │     ├─► 로컬 캐시 업데이트
   │     ├─► aggregator.Aggregate(testRunID)   // 전체 Agent 병합
   │     ├─► wsHub.BroadcastMetrics(aggregated) // WebSocket → 대시보드
   │     └─► store.InsertMetric(report)         // DB INSERT → metrics 테이블
   │
   └─► Web Dashboard: WebSocket onmessage → Recharts 차트 갱신

6. 테스트 완료
   Agent-1: runner.Run() 완료 → 최종 MetricReport 전송
   Agent-2: runner.Run() 완료
   Agent-3: runner.Run() 완료
   │
   ├─► aggregator: 모든 Agent 완료 감지
   ├─► store.UpdateTestRun(status=completed, result_summary=aggregated)
   ├─► aggregator.Cleanup(testRunID)
   └─► ws.BroadcastEvent("test_completed", finalResult)
```

### 6.2 WebSocket 실시간 업데이트 흐름

```
Web Dashboard                    Controller
  │                                  │
  │──[WS Connect /ws/metrics/run123]─>│  wsHub.register(client)
  │                                  │
  │<──[MetricsUpdate]────────────────│  aggregator → wsHub.Broadcast
  │<──[MetricsUpdate]────────────────│  (1초 간격)
  │<──[MetricsUpdate]────────────────│
  │                                  │
  │──[ping]─────────────────────────>│  30초 하트비트
  │<──[pong]─────────────────────────│
  │                                  │
  │<──[TestCompleted event]──────────│  scheduler.onTestComplete
  │                                  │
  │──[WS Close]─────────────────────>│  wsHub.unregister(client)
```

**WebSocket 메시지 형식**:
```json
{
  "type": "metrics_update",
  "data": { AggregatedMetrics }
}
```
```json
{
  "type": "test_completed",
  "data": { "test_run_id": "...", "status": "completed" }
}
```
```json
{
  "type": "agent_status",
  "data": { AgentInfo }
}
```

---

## 7. React 웹 대시보드 구조

### 7.1 기술 스택

| 기술 | 버전 | 역할 |
|------|------|------|
| React | 18 | UI 프레임워크 |
| TypeScript | 5.x | 타입 안전성 |
| Vite | 5.x | 빌드 도구 |
| React Router | 6.x | 라우팅 |
| TanStack Query | 5.x | 서버 상태 관리 (REST API) |
| Zustand | 4.x | 클라이언트 상태 관리 (WebSocket 데이터) |
| Recharts | 2.x | 차트 라이브러리 |
| shadcn/ui | latest | UI 컴포넌트 (Radix + Tailwind) |
| Tailwind CSS | 3.x | 스타일링 |

### 7.2 페이지 라우팅

| 경로 | 컴포넌트 | 설명 |
|------|----------|------|
| `/` | `TestListPage` | 테스트 목록/관리 (Pitch 화면 1) |
| `/tests/:id/run/:runId` | `TestRunPage` | 실시간 메트릭 차트 (Pitch 화면 2) |
| `/agents` | `AgentsPage` | 에이전트 상태 모니터링 (Pitch 화면 3) |

### 7.3 컴포넌트 구조

```
App.tsx
├── Layout.tsx (사이드바 네비게이션 + 헤더)
│
├── TestListPage.tsx
│   ├── TestTable.tsx              # 테스트 목록 (TanStack Query로 GET /tests)
│   ├── CreateTestDialog.tsx       # 테스트 생성 모달 (POST /tests)
│   └── RecentResults.tsx          # 최근 결과 요약
│
├── TestRunPage.tsx
│   ├── MetricsChart.tsx           # RPS/ErrorRate 실시간 라인 차트 (Recharts)
│   ├── LatencyChart.tsx           # Latency Over Time 라인 차트
│   ├── LatencyDistribution.tsx    # P50/P95/P99 바 차트
│   ├── RunHeader.tsx              # 테스트명, 상태, 경과시간, Stop 버튼
│   └── VUserGauge.tsx             # Active VUsers 표시
│
└── AgentsPage.tsx
    ├── AgentTable.tsx             # 에이전트 목록 + 상태
    └── AgentHealthTimeline.tsx    # 에이전트 헬스 타임라인
```

### 7.4 API 클라이언트 + WebSocket 연결

```typescript
// web/src/api/client.ts
const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

interface Envelope<T> {
  data: T;
  error?: { code: string; message: string; details?: string };
  meta: { request_id: string; total?: number; page?: number; per_page?: number };
}

async function fetchAPI<T>(path: string, options?: RequestInit): Promise<Envelope<T>> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...options?.headers },
  });
  const json = await res.json();
  if (!res.ok) throw new APIError(json.error, res.status);
  return json;
}
```

```typescript
// web/src/api/hooks.ts (TanStack Query)
export function useTests(page = 1) {
  return useQuery({
    queryKey: ['tests', page],
    queryFn: () => fetchAPI<TestDefinition[]>(`/api/v1/tests?page=${page}`),
    refetchInterval: 5000, // 5초 폴링 (WebSocket 없는 화면)
  });
}

export function useRunTest(testId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body?: { vusers_override?: number }) =>
      fetchAPI<TestRun>(`/api/v1/tests/${testId}/run`, { method: 'POST', body: JSON.stringify(body) }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tests'] }),
  });
}
```

```typescript
// web/src/ws/useWebSocket.ts
export function useMetricsStream(testRunId: string) {
  const addMetric = useStore(s => s.addMetric);
  const wsUrl = `${WS_BASE}/ws/metrics/${testRunId}`;

  useEffect(() => {
    const ws = new WebSocket(wsUrl);
    ws.onmessage = (e) => {
      const msg = JSON.parse(e.data);
      if (msg.type === 'metrics_update') addMetric(msg.data);
    };
    ws.onclose = () => {
      // 3초 후 재연결 시도
      setTimeout(() => { /* reconnect */ }, 3000);
    };
    return () => ws.close();
  }, [testRunId]);
}
```

### 7.5 상태 관리 패턴

```typescript
// web/src/store/useStore.ts (Zustand)
interface AppState {
  // 실시간 메트릭 (WebSocket으로 수신)
  metricsHistory: AggregatedMetrics[];
  addMetric: (m: AggregatedMetrics) => void;
  clearMetrics: () => void;

  // 에이전트 상태 (WebSocket으로 수신)
  agents: Map<string, AgentInfo>;
  updateAgent: (a: AgentInfo) => void;
}

export const useStore = create<AppState>((set) => ({
  metricsHistory: [],
  addMetric: (m) => set((s) => ({
    metricsHistory: [...s.metricsHistory.slice(-300), m], // 최근 300개 (5분)
  })),
  clearMetrics: () => set({ metricsHistory: [] }),

  agents: new Map(),
  updateAgent: (a) => set((s) => {
    const next = new Map(s.agents);
    next.set(a.agent_id, a);
    return { agents: next };
  }),
}));
```

### 7.6 디자인 토큰

```typescript
// Pitch UX 섹션 색상 코드
const colors = {
  success: '#22c55e',   // 정상/통과
  error:   '#ef4444',   // 오류/실패
  warning: '#f59e0b',   // 경고/주의
  active:  '#3b82f6',   // 실행 중/활성
  idle:    '#6b7280',   // 비활성/대기
};
```

- 다크 테마 기본 (Tailwind `dark` mode)
- 1280px+ 데스크톱 우선 반응형

---

## 8. Docker Compose 구성

### 8.1 현재 상태 (이미 존재: `docker-compose.yml`)

현재 docker-compose.yml에는 **postgres**, **controller**, **agent-1**, **agent-2** 서비스가 정의되어 있다.

### 8.2 Pitch 대비 보완 필요사항

| 항목 | 현재 상태 | Pitch 요구 | 보완 |
|------|-----------|------------|------|
| Agent 수 | 2대 | 3대 | `agent-3` 서비스 추가 |
| Dashboard | 없음 | React SPA | `dashboard` 서비스 추가 |
| PostgreSQL | 15-alpine | 16-alpine | 버전 업그레이드 |
| WebSocket 포트 | 8081 (별도) | 8080 (통합) | REST + WS를 같은 포트에서 처리 권장 |
| 볼륨 | postgres-data만 | 동일 | OK |

### 8.3 최종 Docker Compose 구성

```yaml
version: "3.9"

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: omnitest
      POSTGRES_PASSWORD: omnitest
      POSTGRES_DB: omnitest
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d/migrations
      - ./docker/postgres/init.sql:/docker-entrypoint-initdb.d/00-init.sql
    networks:
      - omnitest-net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U omnitest"]
      interval: 5s
      timeout: 5s
      retries: 5

  controller:
    build:
      context: .
      dockerfile: docker/Dockerfile
    command: ["controller"]
    ports:
      - "8080:8080"   # REST API + WebSocket
      - "9090:9090"   # gRPC
    environment:
      OMNITEST_DB_URL: postgres://omnitest:omnitest@postgres:5432/omnitest?sslmode=disable
      OMNITEST_GRPC_PORT: "9090"
      OMNITEST_HTTP_PORT: "8080"
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - omnitest-net

  agent-1:
    build:
      context: .
      dockerfile: docker/Dockerfile
    command: ["agent", "--controller=controller:9090", "--name=agent-1"]
    environment:
      OMNITEST_MAX_VUSERS: "5000"
    depends_on:
      - controller
    networks:
      - omnitest-net

  agent-2:
    build:
      context: .
      dockerfile: docker/Dockerfile
    command: ["agent", "--controller=controller:9090", "--name=agent-2"]
    environment:
      OMNITEST_MAX_VUSERS: "5000"
    depends_on:
      - controller
    networks:
      - omnitest-net

  agent-3:
    build:
      context: .
      dockerfile: docker/Dockerfile
    command: ["agent", "--controller=controller:9090", "--name=agent-3"]
    environment:
      OMNITEST_MAX_VUSERS: "5000"
    depends_on:
      - controller
    networks:
      - omnitest-net

  dashboard:
    build:
      context: ./web
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      VITE_API_URL: http://controller:8080
      VITE_WS_URL: ws://controller:8080
    depends_on:
      - controller
    networks:
      - omnitest-net

volumes:
  pgdata:

networks:
  omnitest-net:
    driver: bridge
```

### 8.4 Dockerfile 업데이트

현재 Dockerfile은 단일 바이너리만 빌드한다. `ENTRYPOINT ["omnitest"]`에 `command`로 서브커맨드를 전달하는 방식이므로 변경 불필요.

**web/Dockerfile** (신규):
```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 3000
```

---

## 9. Rabbit Hole 기술 검증

### 9.1 gRPC 양방향 스트리밍 동시성

**결론**: 클라이언트 스트리밍(Agent → Controller)으로 충분. 양방향은 불필요.

- `StreamMetrics`는 Agent가 1초 간격으로 `MetricReport`를 push하는 클라이언트 스트리밍
- Controller → Agent 제어 명령은 별도 RPC (`StartTest`, `StopTest`)로 처리
- 진정한 양방향 스트리밍은 과도한 복잡도. Agent가 Controller를 폴링하지 않으므로 단방향으로 충분

**동시성 안전**:
- `aggregator.OnMetricReport()`에 `sync.RWMutex` 필수 (여러 Agent가 동시에 스트리밍)
- 각 Agent의 `StreamMetrics` goroutine은 gRPC가 자동 관리
- Agent 3대 정도에서 동시성 이슈는 발생하지 않으나, Mutex 보호가 안전

### 9.2 PostgreSQL 연결 풀 설정

**결론**: pgx pool `MaxConns=20`으로 충분.

- Controller는 단일 프로세스. REST API 핸들러 + 메트릭 저장이 주요 소비자
- Agent 3대 × 1초 간격 메트릭 INSERT = 3 QPS (매우 낮음)
- REST API는 웹 대시보드 단일 클라이언트가 주 사용자
- `MinConns=5`: 시작 시 웜업. `MaxConnLifetime=30m`: PostgreSQL 메모리 관리

### 9.3 WebSocket 브로드캐스트 성능

**결론**: gorilla/websocket + 클라이언트당 goroutine 1개로 충분.

- MVP 대시보드는 동시 클라이언트 수가 적음 (1-5명)
- 1초 간격 브로드캐스트: `AggregatedMetrics` JSON 직렬화 → 각 클라이언트 `send` 채널에 push
- 클라이언트 `WritePump` goroutine이 채널에서 읽어 전송
- `send` 채널 버퍼 256: 일시적 느린 클라이언트 대응. 버퍼 초과 시 클라이언트 연결 종료

### 9.4 Cycle 1 runner.go 재사용 방안

**결론**: `runner.Run()`을 Agent 모드에서 그대로 호출. 수정 최소화.

현재 `runner.Run(ctx, cfg, opts)` 시그니처:
- `ctx`: Agent에서 `context.WithCancel` 전달 (StopTest 시 cancel)
- `cfg`: `config.LoadFromString(scenario_yaml)`으로 생성 + VUsers 오버라이드
- `opts.Quiet = true`: 터미널 출력 비활성화

**필요한 수정**:
1. `config` 패키지에 `LoadFromString(yaml string) (*model.TestConfig, error)` 추가
2. `runner.Run()`에 **메트릭 콜백** 옵션 추가:

```go
type Options struct {
    Quiet      bool
    NoColor    bool
    OnSnapshot func(snap model.MetricSnapshot) // Agent 모드: 스냅샷을 gRPC로 전송
}
```

`runner.Run()` 내부의 snapshot 루프에서 `OnSnapshot`이 설정되어 있으면 호출. Agent는 이 콜백에서 `MetricReport`로 변환하여 gRPC 스트림에 전송한다.

이 접근법은 runner.go의 기존 로직을 거의 변경하지 않으면서 Agent 모드를 지원한다.

### 9.5 ID 생성 전략

**결론**: nanoid prefix 패턴 사용.

```go
import "github.com/jaevor/go-nanoid"

func NewTestID() string    { return "test-" + nanoid(12) }
func NewRunID() string     { return "run-" + nanoid(12) }
func NewAgentID() string   { return "agent-" + nanoid(8) }
func NewRequestID() string { return "req-" + nanoid(12) }
```

DB의 `UUID` 타입은 유지하되, application 레벨에서는 prefix+nanoid를 사용. DB에 저장 시 UUID 변환이 필요하므로, **DB PK는 UUID, 외부 노출 ID는 prefix+nanoid 또는 UUID 문자열 그대로 사용**하는 방식을 검토. MVP에서는 단순히 UUID 문자열을 그대로 사용하는 것이 현실적.

---

## 10. 의존성 목록 (Cycle 2 추가분)

### go.mod 추가 의존성

```
require (
    // Cycle 1 (유지)
    github.com/spf13/cobra               v1.10.2
    gopkg.in/yaml.v3                      v3.0.1
    github.com/HdrHistogram/hdrhistogram-go  v1.1.2

    // Cycle 2 (신규)
    google.golang.org/grpc                v1.62.0      // gRPC 서버/클라이언트
    google.golang.org/protobuf            v1.33.0      // protobuf 런타임
    github.com/jackc/pgx/v5              v5.5.3       // PostgreSQL 드라이버
    github.com/gorilla/websocket         v1.5.1       // WebSocket
    github.com/golang-migrate/migrate/v4 v4.17.0      // DB 마이그레이션
)
```

### 프론트엔드 (web/package.json)

```json
{
  "dependencies": {
    "react": "^18.3.0",
    "react-dom": "^18.3.0",
    "react-router-dom": "^6.22.0",
    "@tanstack/react-query": "^5.28.0",
    "zustand": "^4.5.0",
    "recharts": "^2.12.0"
  },
  "devDependencies": {
    "typescript": "^5.4.0",
    "vite": "^5.2.0",
    "@vitejs/plugin-react": "^4.2.0",
    "tailwindcss": "^3.4.0",
    "autoprefixer": "^10.4.0",
    "postcss": "^8.4.0"
  }
}
```

shadcn/ui는 패키지가 아니라 CLI로 컴포넌트를 복사하는 방식이므로 별도 dependency 불필요.

---

## 11. Makefile 추가 타겟

```makefile
# Cycle 2 추가
proto:
	buf generate

dashboard-dev:
	cd web && npm run dev

dashboard-build:
	cd web && npm run build

docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1
```

---

## 12. 구현 우선순위 (Phase 분해)

의존성 순서를 고려하여 아래 순서로 구현한다:

```
Phase 1: 기반 인프라 (Week 1 전반)
  ├── proto 코드 생성 (buf generate → omnitestv1/)
  ├── internal/store (PostgreSQL 연결 + CRUD)
  ├── internal/grpc/server.go + client.go (스켈레톤)
  ├── config.LoadFromString() 추가
  └── runner.Options.OnSnapshot 콜백 추가

Phase 2: Agent 모드 (Week 1 후반)
  ├── internal/agent/agent.go (Controller 연결, Register, Heartbeat)
  ├── internal/agent/reconnect.go (exponential backoff)
  ├── internal/grpc/client.go (Agent gRPC 클라이언트 완성)
  ├── Agent의 executeTest() (runner.Run 호출 + StreamMetrics)
  └── cmd/omnitest: "agent" 서브커맨드 추가

Phase 3: Controller 코어 (Week 2 전반)
  ├── internal/controller/server.go (통합 서버)
  ├── internal/controller/agent_manager.go (등록/헬스체크)
  ├── internal/controller/scheduler.go (VUser 분배)
  ├── internal/controller/aggregator.go (메트릭 병합)
  ├── internal/grpc/server.go (AgentService 구현 완성)
  └── cmd/omnitest: "controller" 서브커맨드 추가

Phase 4: REST API (Week 2 후반)
  ├── internal/api/server.go + middleware.go + response.go
  ├── internal/api/test_handler.go (CRUD + run/stop)
  ├── internal/api/result_handler.go
  ├── internal/api/agent_handler.go
  └── internal/api/system_handler.go

Phase 5: WebSocket + 대시보드 (Week 3)
  ├── internal/ws/hub.go + client.go
  ├── web/ React 프로젝트 초기화
  ├── TestListPage + TestRunPage + AgentsPage
  ├── MetricsChart + LatencyDistribution
  └── WebSocket 실시간 연동

Phase 6: Docker Compose 통합 + 안정화 (Week 4)
  ├── docker-compose.yml 최종 구성
  ├── web/Dockerfile
  ├── 통합 테스트 (Controller + Agent 3대 + DB)
  ├── AC-1 ~ AC-5 수용 기준 검증
  └── 엣지케이스 처리 (Agent 연결 끊김, DB 오류 등)
```

### Hill Chart 체크포인트

| 시점 | 달성 기준 | Hill 위치 |
|------|-----------|-----------|
| Week 1 종료 | Controller-Agent gRPC 핸드셰이크(Register+Heartbeat) 성공 | 정상 도달 (50%) |
| Week 2 종료 | REST API로 테스트 생성 → Agent 분산 실행 → 메트릭 집계 | 내리막 30% |
| Week 3 종료 | 웹 대시보드 3개 화면 + WebSocket 실시간 차트 | 내리막 80% |
| Week 4 종료 | `docker-compose up` 풀스택 배포 + AC 전체 통과 | 완료 (100%) |
