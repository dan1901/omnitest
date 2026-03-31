package model

import "time"

// --- YAML 파싱 대상 (Input) ---

// TestConfig는 YAML 파일의 최상위 구조체다.
type TestConfig struct {
	Version    string      `yaml:"version"`
	Targets    []Target    `yaml:"targets"`
	Scenarios  []Scenario  `yaml:"scenarios"`
	Thresholds []Threshold `yaml:"thresholds,omitempty"`
}

// Target은 테스트 대상 서버를 정의한다.
type Target struct {
	Name    string            `yaml:"name"`
	BaseURL string            `yaml:"base_url"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

// Scenario는 부하 시나리오를 정의한다.
type Scenario struct {
	Name     string        `yaml:"name"`
	Target   string        `yaml:"target"`
	VUsers   int           `yaml:"vusers"`
	Duration time.Duration `yaml:"duration"`
	RampUp   time.Duration `yaml:"ramp_up,omitempty"`
	Requests []Request     `yaml:"requests"`
}

// Request는 시나리오 내 개별 HTTP 요청을 정의한다.
type Request struct {
	Method  string            `yaml:"method"`
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    map[string]any    `yaml:"body,omitempty"`
}

// Threshold는 Pass/Fail 판정 기준이다.
type Threshold struct {
	Metric    string `yaml:"metric"`
	Condition string `yaml:"condition"`
}

// --- 실시간 메트릭 (Runtime) ---

// MetricSnapshot은 특정 시점의 메트릭 스냅샷이다.
type MetricSnapshot struct {
	Timestamp    time.Time
	ElapsedSec   float64
	ActiveVUsers int
	TotalVUsers  int
	RPS          float64
	AvgLatency   time.Duration
	P50Latency   time.Duration
	P95Latency   time.Duration
	P99Latency   time.Duration
	ErrorRate    float64
	TotalReqs    int64
	TotalErrors  int64
	BytesIn      int64
}

// --- 최종 결과 (Output) ---

// TestResult는 테스트 완료 후 최종 결과다.
type TestResult struct {
	ScenarioName string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration

	TotalRequests int64
	SuccessCount  int64
	ErrorCount    int64
	ErrorRate     float64

	AvgLatency time.Duration
	P50Latency time.Duration
	P95Latency time.Duration
	P99Latency time.Duration
	MaxLatency time.Duration
	MinLatency time.Duration

	AvgRPS float64
	MaxRPS float64

	ThresholdResults []ThresholdResult

	Snapshots []MetricSnapshot
}

// ThresholdResult는 개별 threshold 평가 결과다.
type ThresholdResult struct {
	Metric    string
	Condition string
	Actual    string
	Passed    bool
}

// --- 내부 전달용 ---

// RequestResult는 개별 HTTP 요청의 결과다.
type RequestResult struct {
	StatusCode int
	Latency    time.Duration
	BytesIn    int64
	Error      error
	Timestamp  time.Time
}

// --- Cycle 2: Controller 모델 ---

// AgentInfo는 등록된 Agent의 상태 정보다.
type AgentInfo struct {
	AgentID       string            `json:"agent_id"`
	Hostname      string            `json:"hostname"`
	MaxVUsers     int               `json:"max_vusers"`
	Labels        map[string]string `json:"labels"`
	Status        AgentStatus       `json:"status"`
	ActiveVUsers  int               `json:"active_vusers"`
	CPUUsage      float64           `json:"cpu_usage"`
	MemoryUsage   float64           `json:"memory_usage"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	RegisteredAt  time.Time         `json:"registered_at"`
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
	ID              string            `json:"id"`
	TestID          string            `json:"test_id"`
	Status          TestRunStatus     `json:"status"`
	TotalVUsers     int               `json:"total_vusers"`
	DurationSeconds int               `json:"duration_seconds"`
	StartedAt       *time.Time        `json:"started_at,omitempty"`
	FinishedAt      *time.Time        `json:"finished_at,omitempty"`
	ResultSummary   *TestResult       `json:"result_summary,omitempty"`
	ThresholdResults []ThresholdResult `json:"threshold_results,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
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
	AgentID        string `json:"agent_id"`
	AssignedVUsers int    `json:"assigned_vusers"`
}

// AggregatedMetrics는 Controller가 Agent별 메트릭을 병합한 결과다.
type AggregatedMetrics struct {
	TestRunID    string                       `json:"test_run_id"`
	Timestamp    time.Time                    `json:"timestamp"`
	TotalRPS     float64                      `json:"total_rps"`
	AvgLatencyMs float64                      `json:"avg_latency_ms"`
	P50LatencyMs float64                      `json:"p50_latency_ms"`
	P95LatencyMs float64                      `json:"p95_latency_ms"`
	P99LatencyMs float64                      `json:"p99_latency_ms"`
	TotalReqs    int64                        `json:"total_requests"`
	TotalErrors  int64                        `json:"total_errors"`
	ActiveVUsers int                          `json:"active_vusers"`
	PerAgent     map[string]*MetricSnapshot   `json:"per_agent,omitempty"`
}
