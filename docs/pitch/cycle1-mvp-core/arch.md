# Architecture Guide: Cycle 1 - MVP Core Engine

> Builder가 즉시 구현에 착수할 수 있는 기술 명세

---

## 1. 프로젝트 구조

```
omnitest/
├── cmd/
│   └── omnitest/
│       └── main.go              # CLI 엔트리포인트 (cobra root/sub commands)
├── internal/
│   ├── config/
│   │   └── config.go            # YAML 파싱 + 검증 + 환경변수 치환
│   ├── runner/
│   │   └── runner.go            # 테스트 오케스트레이션 (시나리오 → worker 조율)
│   ├── worker/
│   │   └── worker.go            # goroutine 기반 VUser 워커 풀
│   ├── http/
│   │   └── client.go            # net/http 래퍼 (Transport 설정, 요청 실행)
│   ├── metrics/
│   │   └── collector.go         # HDR Histogram 기반 메트릭 수집기
│   ├── output/
│   │   └── printer.go           # 터미널 실시간 출력 (ANSI 제어)
│   └── report/
│       ├── json.go              # JSON 리포트 생성
│       └── html.go              # HTML 리포트 생성 (html/template)
├── pkg/
│   └── model/
│       └── model.go             # 공개 데이터 모델 (TestConfig, TestResult 등)
├── testdata/
│   └── sample.yaml              # 예제 시나리오 파일
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
└── .golangci.yml
```

### 패키지 역할 요약

| 패키지 | 레이어 | 역할 |
|--------|--------|------|
| `cmd/omnitest` | Entrypoint | cobra CLI 명령어 정의, 플래그 파싱, 각 패키지 연결 |
| `internal/config` | Input | YAML 파일 로드, 구조체 파싱, 환경변수 `${VAR}` 치환, 스키마 검증 |
| `internal/runner` | Orchestration | 시나리오별 워커 풀 생성, Ramp-up 제어, 종료 조건 판정, 결과 집계 |
| `internal/worker` | Execution | goroutine 풀 관리, VUser 생명주기, 요청 루프 실행 |
| `internal/http` | I/O | HTTP 클라이언트 설정, 요청 빌드/실행, 응답 측정 |
| `internal/metrics` | Collection | HDR Histogram 래핑, RPS/latency/error 실시간 집계, 스냅샷 제공 |
| `internal/output` | Presentation | 터미널 실시간 테이블 갱신, 프로그레스 바, 최종 요약 출력 |
| `internal/report` | Output | JSON/HTML 리포트 파일 생성, threshold 평가 |
| `pkg/model` | Shared | 패키지 간 공유 데이터 구조체 (config, result, metrics) |

---

## 2. 핵심 데이터 모델

> `pkg/model/model.go`에 정의. Pitch의 YAML 스키마를 반영하여 기존 모델을 확장.

```go
package model

import "time"

// ─── YAML 파싱 대상 (Input) ───

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
    Target   string        `yaml:"target"`   // Target.Name 참조
    VUsers   int           `yaml:"vusers"`
    Duration time.Duration `yaml:"duration"`
    RampUp   time.Duration `yaml:"ramp_up,omitempty"`
    Requests []Request     `yaml:"requests"`
}

// Request는 시나리오 내 개별 HTTP 요청을 정의한다.
type Request struct {
    Method string            `yaml:"method"` // GET, POST, PUT, DELETE
    Path   string            `yaml:"path"`
    Headers map[string]string `yaml:"headers,omitempty"`
    Body   map[string]any    `yaml:"body,omitempty"`
}

// Threshold는 Pass/Fail 판정 기준이다.
type Threshold struct {
    Metric    string `yaml:"metric"`    // e.g. "http_req_duration_p99"
    Condition string `yaml:"condition"` // e.g. "< 200ms"
}

// ─── 실시간 메트릭 (Runtime) ───

// MetricSnapshot은 특정 시점의 메트릭 스냅샷이다.
// output 패키지가 1초마다 이 구조체를 받아 터미널에 렌더링한다.
type MetricSnapshot struct {
    Timestamp   time.Time
    ElapsedSec  float64
    ActiveVUsers int
    TotalVUsers  int
    RPS         float64
    AvgLatency  time.Duration
    P50Latency  time.Duration
    P95Latency  time.Duration
    P99Latency  time.Duration
    ErrorRate   float64       // 0.0 ~ 1.0
    TotalReqs   int64
    TotalErrors int64
    BytesIn     int64
}

// ─── 최종 결과 (Output) ───

// TestResult는 테스트 완료 후 최종 결과다.
// report 패키지가 이 구조체로 JSON/HTML을 생성한다.
type TestResult struct {
    ScenarioName string
    StartTime    time.Time
    EndTime      time.Time
    Duration     time.Duration

    TotalRequests  int64
    SuccessCount   int64
    ErrorCount     int64
    ErrorRate      float64

    AvgLatency time.Duration
    P50Latency time.Duration
    P95Latency time.Duration
    P99Latency time.Duration
    MaxLatency time.Duration
    MinLatency time.Duration

    AvgRPS float64
    MaxRPS float64

    ThresholdResults []ThresholdResult

    // 시계열 스냅샷 (HTML 차트용)
    Snapshots []MetricSnapshot
}

// ThresholdResult는 개별 threshold 평가 결과다.
type ThresholdResult struct {
    Metric    string
    Condition string
    Actual    string  // e.g. "245ms", "0.08%"
    Passed    bool
}

// ─── 내부 전달용 ───

// RequestResult는 개별 HTTP 요청의 결과다.
// worker → metrics collector로 전달된다.
type RequestResult struct {
    StatusCode int
    Latency    time.Duration
    BytesIn    int64
    Error      error
    Timestamp  time.Time
}
```

**기존 모델 대비 변경점**:
- `TestConfig`: Pitch의 YAML 스키마에 맞춰 `version`, `targets`(name+base_url 구조), `scenarios`(requests 포함), `thresholds` 추가
- `Target`: 기존 `URL`+`Method` → `Name`+`BaseURL`로 변경 (요청은 Scenario.Requests로 이동)
- `Scenario`: `Requests []Request`, `RampUp`, `Target` 필드 추가
- `MetricSnapshot`: 실시간 출력용 새 구조체
- `TestResult`: 리포트 생성용 확장 (threshold 결과, 시계열 스냅샷 포함)
- `RequestResult`: 기존 `Result` → `RequestResult`로 명확히 리네이밍

---

## 3. 패키지별 API 명세

### 3.1 `internal/config`

```go
// Load는 YAML 파일을 읽고 파싱하여 TestConfig를 반환한다.
// 환경변수 ${VAR} 치환을 수행한 후 스키마 검증을 실행한다.
func Load(path string) (*model.TestConfig, error)

// Validate는 TestConfig의 필수 필드와 참조 무결성을 검증한다.
// (내부 함수, Load에서 자동 호출)
func validate(cfg *model.TestConfig) error

// expandEnvVars는 문자열 내 ${VAR}를 os.Getenv로 치환한다.
func expandEnvVars(s string) string
```

**구현 포인트**:
- `${VAR}` 치환: `regexp.MustCompile(`\$\{(\w+)\}`)` + `os.Getenv`로 단순 구현
- 검증: targets 존재, scenarios 존재, scenario.target이 targets에 존재하는지, vusers > 0, duration > 0
- 에러 메시지: Pitch UX 섹션의 What→Why→How 3단 구조 준수

### 3.2 `internal/runner`

```go
// Run은 테스트의 전체 생명주기를 오케스트레이션한다.
// config → worker pool 생성 → 실행 → metrics 수집 → 결과 반환
func Run(ctx context.Context, cfg *model.TestConfig) (*model.TestResult, error)
```

**내부 흐름**:
1. `cfg.Scenarios[0]`에서 target을 찾아 base_url 결정 (MVP는 단일 시나리오)
2. `metrics.NewCollector()` 생성
3. `worker.NewPool()` 생성 및 시작
4. `output.NewPrinter()` goroutine으로 실시간 출력 시작
5. duration 타이머 또는 ctx.Done() 대기
6. worker pool 종료 → collector에서 최종 결과 수집
7. `*model.TestResult` 반환

### 3.3 `internal/worker`

```go
// WorkerConfig는 워커 풀 설정이다.
type WorkerConfig struct {
    VUsers    int
    RampUp    time.Duration
    BaseURL   string
    Requests  []model.Request
    Headers   map[string]string // target-level 공통 헤더
}

// Pool은 goroutine 기반 VUser 워커 풀이다.
type Pool struct { ... }

// NewPool은 새 워커 풀을 생성한다.
func NewPool(cfg WorkerConfig, resultCh chan<- model.RequestResult) *Pool

// Start는 워커 풀을 시작한다. Ramp-up에 따라 점진적으로 VUser를 증가시킨다.
// ctx가 취소되면 모든 워커를 gracefully 종료한다.
func (p *Pool) Start(ctx context.Context)

// Wait는 모든 워커가 종료될 때까지 대기한다.
func (p *Pool) Wait()
```

**구현 포인트**:
- 각 VUser = 1 goroutine, requests를 순차 반복 실행
- Ramp-up: 선형. `interval = rampUp / vusers`, ticker로 하나씩 시작
- `resultCh`로 매 요청 결과를 metrics collector에 전달
- `sync.WaitGroup`으로 종료 대기

### 3.4 `internal/http`

```go
// Client는 부하 테스트용 HTTP 클라이언트다.
type Client struct { ... }

// NewClient는 최적화된 Transport 설정으로 HTTP 클라이언트를 생성한다.
func NewClient() *Client

// Do는 HTTP 요청을 실행하고 RequestResult를 반환한다.
func (c *Client) Do(ctx context.Context, baseURL string, req model.Request, headers map[string]string) model.RequestResult
```

**Transport 설정** (성능 최적화):
```go
transport := &http.Transport{
    MaxIdleConns:        1000,
    MaxIdleConnsPerHost: 1000,
    MaxConnsPerHost:     1000,
    IdleConnTimeout:     90 * time.Second,
    DisableCompression:  true,
}
client := &http.Client{
    Transport: transport,
    Timeout:   30 * time.Second,
}
```

### 3.5 `internal/metrics`

```go
// Collector는 HDR Histogram 기반 실시간 메트릭 수집기다.
type Collector struct { ... }

// NewCollector는 새 메트릭 수집기를 생성한다.
func NewCollector() *Collector

// Record는 개별 요청 결과를 기록한다. 동시성 안전.
func (c *Collector) Record(r model.RequestResult)

// Snapshot은 현재 시점의 메트릭 스냅샷을 반환한다.
// output 패키지가 1초마다 호출한다.
func (c *Collector) Snapshot(activeVUsers, totalVUsers int) model.MetricSnapshot

// Result는 최종 집계 결과를 반환한다.
func (c *Collector) Result(scenarioName string, start, end time.Time, snapshots []model.MetricSnapshot) *model.TestResult
```

**HDR Histogram 동시성**:
- `hdrhistogram.Histogram` 자체는 thread-safe하지 않음
- `sync.Mutex`로 `Record()`와 `Snapshot()` 보호 필수
- 대안: 1초 간격 윈도우 히스토그램 + 글로벌 누적 히스토그램 (interval snapshot 패턴)

### 3.6 `internal/output`

```go
// Printer는 터미널 실시간 메트릭 출력기다.
type Printer struct { ... }

// NewPrinter는 새 출력기를 생성한다.
func NewPrinter(snapshotFn func() model.MetricSnapshot, noColor bool) *Printer

// Start는 1초 간격으로 터미널을 갱신하는 goroutine을 시작한다.
func (p *Printer) Start(ctx context.Context)

// PrintSummary는 테스트 완료 후 최종 요약을 출력한다.
func (p *Printer) PrintSummary(result *model.TestResult)

// PrintThresholds는 threshold 평가 결과를 출력한다.
func (p *Printer) PrintThresholds(results []model.ThresholdResult)
```

**터미널 갱신 방식**:
- ANSI escape: `\033[{N}A` (커서 N줄 위로) + `\033[K` (라인 지우기)로 in-place 갱신
- `--no-color` 플래그: ANSI 색상 코드 비활성화
- `--quiet` 플래그: Start() 건너뛰고 PrintSummary()만 실행

### 3.7 `internal/report`

```go
// Generate는 TestResult를 지정된 형식으로 리포트 파일을 생성한다.
func Generate(result *model.TestResult, formats []string, outDir string) error

// generateJSON은 JSON 리포트를 생성한다.
func generateJSON(result *model.TestResult, outDir string) (string, error)

// generateHTML은 HTML 리포트를 생성한다.
func generateHTML(result *model.TestResult, outDir string) (string, error)
```

**HTML 리포트**:
- `html/template` 사용, 인라인 CSS/JS로 단일 HTML 파일
- 차트: 인라인 SVG로 latency 시계열 그래프 생성 (외부 의존성 없음)
- 파일명: `report-{yyyyMMdd}-{HHmmss}.{json|html}`

---

## 4. 실행 흐름 시퀀스

```
User
  │
  │  $ omnitest run load-test.yaml --out json --out html
  ▼
cmd/omnitest/main.go
  │  cobra 플래그 파싱 (--vusers, --duration, --out, --out-dir, --quiet, --no-color)
  │
  ▼
config.Load("load-test.yaml")
  │  1. os.ReadFile
  │  2. expandEnvVars (${TOKEN} → os.Getenv)
  │  3. yaml.Unmarshal → *TestConfig
  │  4. validate (필수 필드, 참조 무결성)
  │  5. CLI 플래그로 오버라이드 (--vusers, --duration)
  │
  ▼
runner.Run(ctx, cfg)
  │
  ├─► metrics.NewCollector()
  │     HDR Histogram 초기화 (1μs ~ 1h 범위, 유효 숫자 3자리)
  │
  ├─► resultCh := make(chan model.RequestResult, bufferSize)
  │
  ├─► worker.NewPool(workerCfg, resultCh)
  │     VUser goroutine 풀 준비
  │
  ├─► [goroutine] metrics 수집 루프
  │     for r := range resultCh {
  │         collector.Record(r)
  │     }
  │
  ├─► [goroutine] output.Printer.Start(ctx)
  │     1초마다 collector.Snapshot() 호출 → 터미널 갱신
  │     스냅샷을 snapshots 슬라이스에 축적 (HTML 차트용)
  │
  ├─► pool.Start(ctx)
  │     │  Ramp-up: interval = rampUp / vusers
  │     │  ticker마다 goroutine 1개 시작
  │     │
  │     │  각 VUser goroutine:
  │     │    for ctx not done {
  │     │      for _, req := range requests {
  │     │        result := httpClient.Do(ctx, baseURL, req, headers)
  │     │        resultCh <- result
  │     │      }
  │     │    }
  │     │
  │     ▼  duration 경과 → ctx cancel
  │
  ├─► pool.Wait()           // 모든 VUser 종료 대기
  ├─► close(resultCh)       // metrics 수집 루프 종료
  ├─► collector.Result()    // 최종 결과 집계
  │
  ▼
runner.Run 반환 → *TestResult
  │
  ▼
cmd/omnitest (후처리)
  │
  ├─► output.PrintSummary(result)      // 터미널 최종 요약
  ├─► output.PrintThresholds(result)   // ✓ PASS / ✗ FAIL
  ├─► report.Generate(result, formats) // JSON/HTML 파일 생성
  │
  ▼
os.Exit(exitCode)
  0 = 모든 threshold 통과
  1 = threshold 실패
  2 = 시나리오 파일 오류
  3 = 연결/네트워크 오류
  99 = 내부 오류
```

### goroutine 구조

```
main goroutine
  │
  ├── metrics 수집 goroutine (resultCh 소비)
  ├── output printer goroutine (1초 ticker)
  │
  └── worker pool
        ├── VUser #1 goroutine (requests 반복)
        ├── VUser #2 goroutine
        ├── ...
        └── VUser #N goroutine
```

---

## 5. Rabbit Hole 기술 검증

### 5.1 HDR Histogram 동시성 안전성

**결론**: `sync.Mutex` 래핑 필수.

`hdrhistogram-go`의 `Histogram.RecordValue()`는 thread-safe하지 않다. 여러 VUser goroutine이 동시에 Record하므로 반드시 동기화가 필요하다.

**채택 패턴**: Channel + 단일 소비자
```
VUser goroutines → resultCh (buffered channel) → 단일 goroutine에서 collector.Record()
```
- Channel이 직렬화를 보장하므로 Mutex 불필요 (채널 자체가 동기화)
- `resultCh` 버퍼 크기: `vusers * 100` (back-pressure 방지)
- Snapshot()은 별도 Mutex로 보호 (수집 goroutine과 output goroutine 간)

### 5.2 net/http vs fasthttp

**결론**: MVP는 `net/http` 확정.

| 기준 | net/http | fasthttp |
|------|----------|----------|
| HTTP/2 | O | X |
| 안정성 | 표준 라이브러리 | 서드파티 |
| 커넥션 풀 | Transport로 제어 | 자체 구현 |
| 성능 (RPS) | ~50,000 RPS/core | ~100,000 RPS/core |

MVP에서 50,000 RPS/core는 충분하다. HTTP/2 지원과 안정성이 더 중요.
`Transport` 설정 최적화(`MaxIdleConnsPerHost`, `DisableCompression`)로 성능 확보.

### 5.3 goroutine당 메모리 / 10,000 VUser 실현 가능성

**결론**: 실현 가능.

- goroutine 초기 스택: ~2KB (동적 확장, 최대 1GB)
- 10,000 goroutine 기본 메모리: ~20MB
- HTTP 요청당 추가 메모리 (버퍼, 응답 읽기): ~10-50KB
- 10,000 VUser 총 예상 메모리: **200-700MB** (응답 크기에 따라)

**리스크**: 응답 body를 전부 읽으면 메모리 폭증 가능
**대응**: `io.Copy(io.Discard, resp.Body)` 또는 `io.LimitReader`로 body 읽기 제한 (MVP에서는 body 내용 불필요, latency만 측정)

### 5.4 YAML 파싱 에러 핸들링 전략

**결론**: Pitch UX 섹션의 What→Why→How 3단 에러 구조 적용.

```go
// 파일 읽기 실패
fmt.Errorf("✗ Error: failed to read scenario file\n  → %s: %w\n  → Check the file path and permissions", path, err)

// YAML 문법 오류 (yaml.v3 TypeError 활용)
var typeErr *yaml.TypeError
if errors.As(err, &typeErr) {
    // typeErr.Errors에서 라인 번호와 필드명 추출
}

// 검증 오류
fmt.Errorf("✗ Error: invalid scenario configuration\n  → scenario[%d]: vusers must be > 0 (got %d)\n  → Set a positive integer for vusers", idx, s.VUsers)
```

### 5.5 Ramp-up 구현

**결론**: 선형 Ramp-up만 구현.

```go
// interval = rampUp / vusers
// 예: rampUp=30s, vusers=100 → 300ms마다 1 VUser 추가
func (p *Pool) Start(ctx context.Context) {
    interval := p.cfg.RampUp / time.Duration(p.cfg.VUsers)
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for i := 0; i < p.cfg.VUsers; i++ {
        p.wg.Add(1)
        go p.runVUser(ctx)
        if i < p.cfg.VUsers-1 {
            select {
            case <-ticker.C:
            case <-ctx.Done():
                return
            }
        }
    }
}
```

RampUp이 0이면 모든 VUser를 즉시 시작.

### 5.6 환경변수 치환

**결론**: 단순 정규식 치환.

```go
var envVarRe = regexp.MustCompile(`\$\{(\w+)\}`)

func expandEnvVars(s string) string {
    return envVarRe.ReplaceAllStringFunc(s, func(match string) string {
        key := envVarRe.FindStringSubmatch(match)[1]
        if val, ok := os.LookupEnv(key); ok {
            return val
        }
        return match // 치환 실패 시 원본 유지
    })
}
```

YAML Unmarshal 전에 raw bytes에 대해 수행. Go template 등 복잡한 엔진 도입하지 않음.

---

## 6. 의존성 목록

### go.mod 직접 의존성 (3개만)

```
require (
    github.com/spf13/cobra          v1.10.2
    gopkg.in/yaml.v3                v3.0.1
    github.com/HdrHistogram/hdrhistogram-go  v1.1.2
)
```

**의도적으로 제외한 것**:
- `bubbletea/lipgloss`: MVP는 ANSI escape 직접 제어. TUI 프레임워크 불필요
- `viper`: CLI 플래그와 YAML 파싱만으로 충분. 설정 핫 리로드 불필요
- `zap/zerolog`: MVP는 `fmt`/`log` 표준 패키지로 충분
- `go-echarts`: HTML 차트는 인라인 SVG로 구현하여 외부 의존성 제거

### 개발 도구 (go.mod 외)

| 도구 | 용도 |
|------|------|
| `golangci-lint` | 린팅 (`.golangci.yml` 이미 설정됨) |
| `goreleaser` | 크로스 플랫폼 바이너리 빌드 (Cycle 1 후반) |

---

## 7. 테스트 전략

### 7.1 단위 테스트 대상

| 패키지 | 테스트 파일 | 핵심 테스트 케이스 |
|--------|------------|-------------------|
| `internal/config` | `config_test.go` | YAML 파싱 성공, 필수 필드 누락 에러, 환경변수 치환, 잘못된 YAML 문법 |
| `internal/metrics` | `collector_test.go` | Record/Snapshot 정확성, 백분위 계산 (p50/p95/p99), 빈 히스토그램 |
| `internal/http` | `client_test.go` | 성공 요청, 타임아웃, 4xx/5xx 응답 처리 (`httptest.NewServer` 사용) |
| `internal/worker` | `worker_test.go` | VUser 수 일치, Ramp-up 타이밍, graceful shutdown |
| `internal/report` | `report_test.go` | JSON 구조 검증, HTML 생성 확인, 파일 생성 경로 |
| `pkg/model` | (구조체만, 로직 없음) | 해당 없음 |

### 7.2 통합 테스트 시나리오

| 시나리오 | 설명 | 검증 포인트 |
|----------|------|------------|
| **E2E 기본 실행** | `httptest.Server` + sample.yaml → `runner.Run()` | 결과에 요청 수 > 0, latency > 0, error rate 산출 |
| **Threshold Pass/Fail** | p99 < 1s threshold → exit code 0 / p99 < 1ms threshold → exit code 1 | exit code 정확성 |
| **Ramp-up** | ramp_up: 2s, vusers: 10 → 초반 RPS가 점진적 증가 확인 | 스냅샷 시계열에서 VUser 증가 패턴 |
| **Graceful Shutdown** | ctx cancel → 모든 goroutine 종료 확인 | goroutine leak 없음 (`runtime.NumGoroutine`) |
| **잘못된 YAML** | 존재하지 않는 target 참조 → 에러 메시지 검증 | 에러 메시지에 What/Why/How 포함 |

### 7.3 테스트 실행

```bash
# 단위 테스트 (race detector 포함)
make test    # go test -v -race ./...

# 특정 패키지
go test -v -race ./internal/config/...
go test -v -race ./internal/metrics/...
```

### 7.4 커버리지 목표

- Cycle 1 Build 완료 시: **70%+**
- Cooldown 후: **80%+**
- 핵심 패키지 (`config`, `metrics`, `worker`): **90%+**

---

## 8. CLI 플래그 → 패키지 매핑

| 플래그 | 처리 위치 | 동작 |
|--------|----------|------|
| `--vusers N` | `cmd/omnitest` → `cfg.Scenarios[0].VUsers` 오버라이드 | |
| `--duration 5m` | `cmd/omnitest` → `cfg.Scenarios[0].Duration` 오버라이드 | |
| `--ramp-up 30s` | `cmd/omnitest` → `cfg.Scenarios[0].RampUp` 오버라이드 | |
| `--out json` | `cmd/omnitest` → `report.Generate(formats)` | repeatable |
| `--out-dir ./reports` | `cmd/omnitest` → `report.Generate(outDir)` | 기본값: `./reports` |
| `--quiet` | `cmd/omnitest` → `output.Printer` 비활성화 | PrintSummary만 실행 |
| `--no-color` | `cmd/omnitest` → `output.NewPrinter(noColor: true)` | ANSI 코드 제거 |
| `--verbose` | `cmd/omnitest` → `log` 레벨 debug 활성화 | |
| `--env KEY=VALUE` | `cmd/omnitest` → `os.Setenv` 후 config.Load | repeatable |

---

## 9. 구현 우선순위 (권장)

Builder가 착수할 때 아래 순서로 구현하면 의존성 충돌 없이 진행 가능:

```
Phase 1: 기반 (Day 1-2)
  ├── pkg/model (데이터 모델 확장)
  ├── internal/config (YAML 파싱 + 검증 + envvar)
  └── internal/http (HTTP 클라이언트)

Phase 2: 엔진 (Day 3-5)
  ├── internal/metrics (HDR Histogram 수집기)
  ├── internal/worker (VUser 풀 + ramp-up)
  └── internal/runner (오케스트레이션)

Phase 3: 출력 (Day 6-7)
  ├── internal/output (터미널 실시간 + 요약)
  └── internal/report (JSON + HTML)

Phase 4: CLI 통합 (Day 8-9)
  └── cmd/omnitest (플래그, validate 명령, exit code)

Phase 5: 안정화 (Day 10+)
  ├── 통합 테스트
  ├── 엣지케이스 처리
  └── GoReleaser 설정
```
