# OmniTest 레퍼런스 문서

> 작성일: 2026-03-17
> 목적: OmniTest 개발에 참고할 오픈소스, 라이브러리, 아키텍처 문서, 패턴 등의 종합 레퍼런스

---

## 1. 경쟁/참고 오픈소스 프로젝트

### 1.1 Go 기반 성능 테스트 도구

| 프로젝트 | GitHub | 설명 | 활용 포인트 |
|----------|--------|------|-------------|
| **k6** | https://github.com/grafana/k6 | Grafana Labs Go 부하 테스트. JS ES6 스크립트, CLI 중심 | 아키텍처 벤치마크. Extension 시스템, 메트릭 파이프라인 참고. 분산 테스트 유료가 OmniTest 차별점 |
| **Vegeta** | https://github.com/tsenart/vegeta | Go HTTP 부하 테스트. 일정 비율 부하 생성 특화 | `attack` 패턴, HDR Histogram 기반 분석, 라이브러리로도 사용 가능한 설계 |
| **Hey** | https://github.com/rakyll/hey | Go 경량 HTTP 벤치마크 (ab 대체) | `net/http` 최적화, 커넥션 풀 관리, 결과 리포팅 |
| **Bombardier** | https://github.com/codesenberg/bombardier | Go 고성능 HTTP 벤치마크. fasthttp 사용 | fasthttp 활용, 터미널 실시간 프로그레스 바 |
| **ghz** | https://github.com/bojand/ghz | Go gRPC 벤치마크 및 부하 테스트 | gRPC 프로토콜 부하 테스트 구현 직접 참고. protobuf reflection |
| **ali** | https://github.com/nakabonne/ali | Go 실시간 HTTP 부하 테스트. 터미널 실시간 차트 | TUI 실시간 메트릭 시각화 참고 |
| **plow** | https://github.com/six-ddc/plow | Go 고성능 HTTP 벤치마크. 실시간 웹 UI 내장 | 로컬 모드 브라우저 실시간 차트 패턴 |
| **ddosify (Anteon)** | https://github.com/ddosify/ddosify | Go 고성능 부하 테스트. JSON 시나리오, 분산 모드 | JSON/YAML 시나리오 엔진, 분산 모드 아키텍처. OmniTest와 가장 유사 |
| **Fortio** | https://github.com/fortio/fortio | Go 부하 테스트 (Istio). 내장 웹 UI, 히스토그램 | Istio/서비스 메시 테스트, 내장 웹 서버+차트 |
| **cassowary** | https://github.com/rogerwelin/cassowary | Go RESTful API 벤치마크 | 경량 CLI 설계 |

### 1.2 다른 언어 기반

| 프로젝트 | GitHub | 설명 | 활용 포인트 |
|----------|--------|------|-------------|
| **Locust** | https://github.com/locustio/locust | Python 분산 부하 테스트. 웹 UI, master-worker | 웹 UI UX 플로우, Master-Worker 분산 패턴 |
| **Gatling** | https://github.com/gatling/gatling | Scala 성능 테스트. 상세 HTML 리포트 | HTML 리포트 디자인 골드 스탠다드. 시나리오 DSL |
| **Artillery** | https://github.com/artilleryio/artillery | Node.js 부하 테스트. YAML 시나리오, 다중 프로토콜 | YAML 시나리오 문법 핵심 참고 (`phases`, `scenarios`, `flow`) |
| **Tsung** | https://github.com/processone/tsung | Erlang 분산 부하 테스트. 다중 프로토콜 | 대규모 분산 아키텍처, 다중 프로토콜 지원 |
| **JMeter** | https://github.com/apache/jmeter | Java 부하 테스트 업계 표준 | 플러그인 아키텍처 참고. "하지 말아야 할 것"의 참고 |
| **nGrinder** | https://github.com/naver/ngrinder | Java 엔터프라이즈 분산 성능 테스트 | 직접 대체 대상. Controller-Agent, 웹 UI, 스케줄링 참고 |
| **Oha** | https://github.com/hatoo/oha | Rust HTTP 부하 생성기. TUI 시각화 | Rust 고성능 참고, TUI 디자인 |
| **Hyperfoil** | https://github.com/Hyperfoil/Hyperfoil | Vert.x 분산 벤치마크. K8s Operator 패턴 | K8s Operator 기반 분산 테스트 아키텍처 직접 참고 |

### 1.3 K8s 네이티브 테스트 도구

| 프로젝트 | GitHub | 설명 | 활용 포인트 |
|----------|--------|------|-------------|
| **k6-operator** | https://github.com/grafana/k6-operator | k6 Kubernetes Operator | K8s CRD 설계, TestRun CRD, Runner Pod 관리 핵심 참고 |
| **Testkube** | https://github.com/kubeshop/testkube | K8s 네이티브 테스트 오케스트레이터 | K8s 네이티브 테스트 실행, CRD 기반 정의, 결과 수집 |

---

## 2. Go 핵심 라이브러리

### 2.1 HTTP 클라이언트

| 라이브러리 | URL | 활용 포인트 |
|-----------|-----|-------------|
| **net/http** (표준) | https://pkg.go.dev/net/http | 기본 HTTP 엔진. Transport 커스터마이징, 커넥션 풀, HTTP/2 |
| **fasthttp** | https://github.com/valyala/fasthttp | 고성능 대안 (HTTP/2 미지원). 벤치마크 비교 후 결정 |
| **resty** | https://github.com/go-resty/resty | 테스트 시나리오 HTTP 요청 빌더, 미들웨어 체인 패턴 |

### 2.2 gRPC & Protocol Buffers

| 라이브러리 | URL | 활용 포인트 |
|-----------|-----|-------------|
| **grpc-go** | https://github.com/grpc/grpc-go | Controller-Agent 핵심 통신. 양방향 스트리밍 |
| **protobuf-go** | https://github.com/protocolbuffers/protobuf-go | 메트릭/제어 명령 직렬화 |
| **buf** | https://github.com/bufbuild/buf | proto 린팅, 브레이킹 체인지 감지, CI 호환성 |
| **connect-go** | https://github.com/connectrpc/connect-go | gRPC+REST 동시 제공 대안. gRPC-Web 프록시 불필요 |
| **grpc-gateway** | https://github.com/grpc-ecosystem/grpc-gateway | gRPC → RESTful JSON API 자동 변환 |

### 2.3 YAML & 설정

| 라이브러리 | URL | 활용 포인트 |
|-----------|-----|-------------|
| **yaml.v3** | gopkg.in/yaml.v3 | 테스트 시나리오 파싱 핵심 |
| **Viper** | https://github.com/spf13/viper | Controller/Agent 설정. 환경변수 오버라이드, 핫 리로드 |
| **koanf** | https://github.com/knadh/koanf | 경량 Viper 대안 |
| **envconfig** | https://github.com/kelseyhightower/envconfig | K8s ConfigMap/Secret 기반 설정 |

### 2.4 CLI & 터미널 UI

| 라이브러리 | URL | 활용 포인트 |
|-----------|-----|-------------|
| **Cobra** | https://github.com/spf13/cobra | CLI 핵심. 서브커맨드, 플래그, 자동 완성 |
| **Bubble Tea** | https://github.com/charmbracelet/bubbletea | TUI 실시간 대시보드 (RPS/레이턴시/에러율) |
| **Lip Gloss** | https://github.com/charmbracelet/lipgloss | CLI 출력 스타일링 |
| **Huh** | https://github.com/charmbracelet/huh | `omnitest init` 대화형 입력 |
| **glamour** | https://github.com/charmbracelet/glamour | 터미널 마크다운 렌더링 |

### 2.5 메트릭 & 히스토그램

| 라이브러리 | URL | 활용 포인트 |
|-----------|-----|-------------|
| **HDR Histogram (Go)** | https://github.com/HdrHistogram/hdrhistogram-go | p50/p95/p99/p999 계산 핵심. Vegeta, k6에서도 사용 |
| **prometheus/client_golang** | https://github.com/prometheus/client_golang | /metrics 엔드포인트 노출 |
| **tdigest** | https://github.com/caio/go-tdigest | 분산 환경 히스토그램 병합 시 HDR 대안 |

### 2.6 WebSocket

| 라이브러리 | URL | 활용 포인트 |
|-----------|-----|-------------|
| **gorilla/websocket** | https://github.com/gorilla/websocket | 웹 대시보드 실시간 메트릭 스트리밍 |
| **coder/websocket** | https://github.com/coder/websocket | 현대적 대안. context 기반 취소 |

### 2.7 Kubernetes

| 라이브러리 | URL | 활용 포인트 |
|-----------|-----|-------------|
| **client-go** | https://github.com/kubernetes/client-go | Agent Pod/Job 관리 핵심 |
| **controller-runtime** | https://github.com/kubernetes-sigs/controller-runtime | Operator Reconciler 패턴, CRD 관리 |
| **kubebuilder** | https://github.com/kubernetes-sigs/kubebuilder | Operator 스캐폴딩, CRD 자동 생성 |

### 2.8 데이터베이스

| 라이브러리 | URL | 활용 포인트 |
|-----------|-----|-------------|
| **pgx** | https://github.com/jackc/pgx | PostgreSQL 고성능 드라이버 |
| **sqlc** | https://github.com/sqlc-dev/sqlc | SQL → 타입 안전 Go 코드 생성 |
| **go-migrate** | https://github.com/golang-migrate/migrate | DB 마이그레이션 |
| **bbolt** | https://github.com/etcd-io/bbolt | 임베디드 KV DB. 에이전트 메트릭 버퍼링 |
| **modernc.org/sqlite** | https://gitlab.com/nicedoc/modernc/sqlite | Pure Go SQLite (CGO 불필요). 단일 바이너리 목표에 부합 |

### 2.9 리포트 & 차트

| 라이브러리 | URL | 활용 포인트 |
|-----------|-----|-------------|
| **go-echarts** | https://github.com/go-echarts/go-echarts | HTML 리포트 인터랙티브 차트 |
| **go-chart** | https://github.com/wcharczuk/go-chart | 정적 차트 PNG/SVG (PDF용) |
| **chromedp** | https://github.com/chromedp/chromedp | HTML→PDF 변환 (headless Chrome) |

### 2.10 기타 핵심 유틸리티

| 라이브러리 | URL | 활용 포인트 |
|-----------|-----|-------------|
| **Yaegi** | https://github.com/traefik/yaegi | Go 인터프리터. Go 스크립트 시나리오 실행 엔진 |
| **NATS** | https://github.com/nats-io/nats.go | 에이전트 디스커버리, 이벤트 브로드캐스트 |
| **expr** | https://github.com/antonmedv/expr | 임계값 조건 평가 (`p99 < 200ms`) |
| **zap** | https://github.com/uber-go/zap | 고성능 구조화 로깅 |
| **GoReleaser** | https://github.com/goreleaser/goreleaser | 크로스 플랫폼 빌드, GitHub Release, Homebrew, Docker |
| **cron** | https://github.com/robfig/cron | 테스트 스케줄링 |
| **Wazero** | https://github.com/tetratelabs/wazero | Pure Go WASM 런타임. Yaegi 대안 플러그인 샌드박스 |
| **go-plugin** | https://github.com/hashicorp/go-plugin | gRPC over stdio 플러그인. 프로세스 격리 |

---

## 3. 아키텍처/설계 참고 문서

### 3.1 분산 부하 테스트 아키텍처

| 자료 | URL | 활용 포인트 |
|------|-----|-------------|
| **k6 아키텍처 소스** | https://github.com/grafana/k6/tree/master/execution | VU 스케줄링, Ramp-up/down 알고리즘, 메트릭 파이프라인 |
| **Locust 분산 문서** | https://docs.locust.io/en/stable/running-distributed.html | 부하 분배, Worker 헬스체크, 결과 집계 |
| **nGrinder 아키텍처** | https://github.com/naver/ngrinder/wiki/Architecture | 대체 대상 아키텍처 분석 |
| **GCP 분산 부하 테스트** | https://cloud.google.com/architecture/distributed-load-testing-using-gke | GKE 레퍼런스 아키텍처 |

### 3.2 Go 동시성 패턴

| 자료 | URL | 활용 포인트 |
|------|-----|-------------|
| **Go Concurrency Patterns** | https://go.dev/talks/2012/concurrency.slide | Generator, Fan-in/out. Worker Pool 설계 기본 |
| **Advanced Go Concurrency** | https://go.dev/talks/2013/advconc.slide | Timeout, Cancellation, Pipeline |
| **Go Pipelines** | https://go.dev/blog/pipelines | 메트릭 수집→집계→전송 파이프라인 |
| **errgroup** | https://pkg.go.dev/golang.org/x/sync/errgroup | Worker goroutine 그룹 관리 |
| **semaphore** | https://pkg.go.dev/golang.org/x/sync/semaphore | VUser 동시성 제어, Ramp-up |
| **Concurrency in Go (O'Reilly)** | Katherine Cox-Buday 저 | 복잡한 동시성 시나리오 가이드 |

### 3.3 gRPC 양방향 스트리밍

| 자료 | URL | 활용 포인트 |
|------|-----|-------------|
| **gRPC Bidirectional Streaming** | https://grpc.io/docs/languages/go/basics/#bidirectional-streaming-rpc | Controller-Agent 양방향 구현 |
| **gRPC keepalive** | https://github.com/grpc/grpc-go/blob/master/Documentation/keepalive.md | 장시간 테스트 중 연결 유지 |
| **gRPC Load Balancing** | https://grpc.io/blog/grpc-load-balancing/ | 다수 Agent의 Controller 연결 전략 |
| **gRPC Health Checking** | https://github.com/grpc/grpc/blob/master/doc/health-checking.md | Agent 헬스체크, K8s probe 연동 |

### 3.4 시계열 데이터 & HDR Histogram

| 자료 | URL | 활용 포인트 |
|------|-----|-------------|
| **HDR Histogram 공식** | https://hdrhistogram.org/ | 백분위 측정 이론적 배경 |
| **"How NOT to Measure Latency"** | Gil Tene (YouTube) | Coordinated Omission 문제. 성능 결과 정확성 보장 |
| **Gorilla 논문 (VLDB 2015)** | Facebook 시계열 압축 논문 | 에이전트 로컬 메트릭 버퍼링 압축 알고리즘 |

---

## 4. 클라우드 네이티브

### 4.1 Kubernetes Operator

| 자료 | URL | 활용 포인트 |
|------|-----|-------------|
| **K8s Operator 공식 문서** | https://kubernetes.io/docs/concepts/extend-kubernetes/operator/ | 설계 기본 원칙 |
| **kubebuilder book** | https://book.kubebuilder.io/ | CRD, Reconciler, Webhook, RBAC |
| **k6-operator 소스** | https://github.com/grafana/k6-operator | TestRun CRD, Runner Pod 관리 직접 참고 |
| **Hyperfoil Operator** | https://github.com/Hyperfoil/hyperfoil-operator | 성능 테스트 Operator 사례 |

### 4.2 Helm & 오토스케일링

| 자료 | URL | 활용 포인트 |
|------|-----|-------------|
| **Helm Best Practices** | https://helm.sh/docs/chart_best_practices/ | Chart 구조, 네이밍, 라벨 |
| **Grafana Helm Charts** | https://github.com/grafana/helm-charts | 성숙한 Chart 설계 참고 |
| **Prometheus Adapter** | https://github.com/kubernetes-sigs/prometheus-adapter | 커스텀 메트릭 기반 HPA |
| **KEDA** | https://github.com/kedacore/keda | 이벤트 드리븐 오토스케일 대안 |

---

## 5. 모니터링/리포트

### 5.1 Prometheus

| 자료 | URL | 활용 포인트 |
|------|-----|-------------|
| **메트릭 네이밍** | https://prometheus.io/docs/practices/naming/ | `omnitest_http_*` 네이밍 규칙 |
| **메트릭 타입** | https://prometheus.io/docs/concepts/metric_types/ | Counter(요청수), Gauge(VUser), Histogram(응답시간) |
| **계측 모범 사례** | https://prometheus.io/docs/practices/instrumentation/ | 라벨 카디널리티, 히스토그램 버킷 |

### 5.2 Grafana & VictoriaMetrics

| 자료 | URL | 활용 포인트 |
|------|-----|-------------|
| **k6 Grafana Dashboard** | https://grafana.com/grafana/dashboards/2587 | RPS, 응답 시간, 에러율 패널 구성 참고 |
| **Grafana Provisioning** | https://grafana.com/docs/grafana/latest/administration/provisioning/ | Helm Chart에 대시보드 자동 프로비저닝 |
| **VictoriaMetrics 공식** | https://docs.victoriametrics.com/ | 메트릭 저장소 운영 가이드 |
| **VictoriaMetrics GitHub** | https://github.com/VictoriaMetrics/VictoriaMetrics | Go 시계열 DB 구현 참고 |
| **Remote Write Spec** | https://prometheus.io/docs/concepts/remote_write_spec/ | 메트릭 Push 표준 프로토콜 |

---

## 6. Shape Up 개발 방법론

| 자료 | URL | 활용 포인트 |
|------|-----|-------------|
| **Shape Up (원서)** | https://basecamp.com/shapeup | 전체 프레임워크. 6주 사이클, Appetite, Pitch, Hill Chart |
| **Fat Marker Sketches** | https://basecamp.com/shapeup/1.3-chapter-04 | 추상적 솔루션 스케치 방법 |
| **Appetite vs Estimate** | https://basecamp.com/shapeup/1.2-chapter-03 | 시간 예산 우선, 범위 조절 원칙 |
| **Hill Charts** | https://basecamp.com/shapeup/3.4-chapter-13 | 불확실성 해소 vs 실행 단계 추적 |
| **Scopes** | https://basecamp.com/shapeup/3.3-chapter-12 | 작업을 의미 있는 단위로 분리 |
| **ADR** | https://adr.github.io/ | 아키텍처 결정 문서화 |

---

## 7. 프론트엔드 (Web UI)

| 라이브러리 | URL | 활용 포인트 |
|-----------|-----|-------------|
| **Recharts** | https://github.com/recharts/recharts | 실시간 메트릭 차트 |
| **echarts-for-react** | https://github.com/hustcc/echarts-for-react | 히트맵, 게이지 등 고급 차트 |
| **TanStack Table** | https://github.com/TanStack/table | 테스트/에이전트/결과 목록 |
| **TanStack Query** | https://github.com/TanStack/query | API 페칭, 캐싱, 실시간 폴링 |
| **Zustand** | https://github.com/pmndrs/zustand | 경량 글로벌 상태 관리 |
| **shadcn/ui** | https://github.com/shadcn-ui/ui | Radix UI + Tailwind CSS 컴포넌트 |

---

## 8. 우선순위 요약

### Tier 1 — 반드시 깊이 분석
1. **k6** - 아키텍처, VU 스케줄링, 메트릭 파이프라인
2. **k6-operator** - K8s CRD/Operator 패턴
3. **Vegeta** - Go 부하 생성 엔진, HDR Histogram
4. **Locust** - 웹 UI/UX, 분산 아키텍처
5. **Artillery** - YAML 시나리오 문법
6. **nGrinder** - 대체 대상 분석, Controller-Agent
7. **Shape Up 원서** - 개발 방법론

### Tier 2 — 주요 참고
8. **ddosify/Anteon** - 가장 유사한 비전의 Go 도구
9. **Gatling** - HTML 리포트 디자인
10. **Hyperfoil** - K8s 네이티브 벤치마크
11. **VictoriaMetrics** - 시계열 DB
12. **kubebuilder book** - Operator 개발
13. **Charmbracelet** (bubbletea, lipgloss) - CLI/TUI

### Tier 3 — 선택적 참고
14. **Bombardier** - fasthttp 벤치마크
15. **ghz** - gRPC 부하 테스트
16. **Testkube** - K8s 테스트 오케스트레이션
17. **KEDA** - 이벤트 기반 오토스케일링
18. **Wazero** - WASM 플러그인 샌드박스
