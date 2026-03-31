# Pitch: Cycle 2 - 분산 아키텍처

> **Quick Gate**: `docker-compose up`으로 Controller + Agent + PostgreSQL + 웹 대시보드를 배포하고, REST API 또는 웹 UI에서 분산 부하 테스트를 생성/실행하며, WebSocket 실시간 차트로 에이전트별 메트릭을 모니터링하는 분산 성능 테스트 플랫폼

---

## 1. Problem

### Cycle 1(로컬 단독 실행)의 한계

Cycle 1 MVP는 `omnitest run test.yaml` 한 줄로 로컬 부하 테스트를 성공적으로 수행한다. 그러나 프로덕션 수준의 성능 테스트에는 근본적 한계가 있다:

| 한계 | 설명 | 영향 |
|------|------|------|
| **수평 확장 불가** | 단일 머신의 CPU/네트워크 대역폭이 부하 생성의 물리적 상한. goroutine 10,000개로도 단일 머신에서 생성 가능한 RPS에 한계가 있다 | 대규모 부하 테스트 불가능 (10,000+ VUser) |
| **중앙 관리 부재** | 테스트 정의/결과가 로컬 파일시스템에 산재. 팀 단위 테스트 관리, 이력 추적, 결과 공유 불가 | 팀 협업 불가, 테스트 자산 유실 위험 |
| **실시간 모니터링 제약** | 터미널 TUI 출력만 가능. 원격 모니터링, 다수 에이전트의 통합 뷰 불가 | QA/성능 엔지니어 워크플로우 미지원 |
| **자동화 제약** | REST API 없이 CLI만 제공. 외부 시스템 연동(Slack 알림, 대시보드 연동 등) 어려움 | 엔터프라이즈 환경 적용 불가 |
| **결과 영속성 없음** | 테스트 결과가 파일로만 저장. 히스토리 비교, 트렌드 분석 불가 | 성능 회귀 추적 불가 |

### 경쟁 도구 대비 갭

- **k6**: CLI 단독 실행은 우수하나, 분산 테스트는 k6 Cloud(유료)에서만 가능. 웹 UI 없음
- **nGrinder**: 분산 테스트 + 웹 UI를 제공하지만, 설치 복잡(JVM/Tomcat/DB), 메모리 비효율, DX 부족
- **Locust**: Python 기반 분산 + 웹 UI 제공. 하지만 Python GIL 한계로 에이전트당 VUser 밀도 낮음

**OmniTest Cycle 2의 기회**: k6 수준의 DX(Go 단일 바이너리, YAML 시나리오)를 유지하면서, nGrinder의 분산 테스트 + 웹 모니터링 역량을 현대적 기술 스택(gRPC, React, WebSocket, Docker Compose)으로 제공한다. 특히 **무료 오픈소스에서 분산 테스트 + 웹 대시보드**를 갖춘 도구는 현재 시장에 부재하다.

---

## 2. Appetite

**Big Batch** - 이번 세션 (4주 Build)

Cycle 2 전체를 투자하여 Controller-Agent 분산 아키텍처, 기본 웹 대시보드, REST API, PostgreSQL 저장소를 완성한다. Kubernetes 네이티브, CI/CD 통합, 인증/권한 등은 Cycle 3 이후로 미룬다.

---

## 3. Solution (Fat Marker Sketch)

### 3.1 Controller-Agent 상호작용 플로우

```
                                  ┌─────────────────────────┐
                                  │     Web Dashboard       │
                                  │  (React + TypeScript)   │
                                  └───────────┬─────────────┘
                                              │ REST + WebSocket
                                              v
┌──────────┐  gRPC       ┌─────────────────────────────────────────┐
│   CLI    │────────────>│              Controller                  │
│(omnitest │  (기존 run  │                                         │
│ agent)   │   + agent)  │  ┌───────────┐ ┌──────────┐ ┌────────┐│     ┌────────────┐
└──────────┘             │  │API Server │ │ Agent    │ │Sched-  ││────>│ PostgreSQL │
                         │  │(REST+WS)  │ │ Manager  │ │uler    ││     │ (메타데이터) │
                         │  └───────────┘ └──────────┘ └────────┘│     └────────────┘
                         └──────┬──────────────┬─────────────┬────┘
                                │              │             │
                          gRPC  │        gRPC  │       gRPC  │
                     (bidirectional streaming)  │             │
                                v              v             v
                         ┌──────────┐  ┌──────────┐  ┌──────────┐
                         │ Agent 1  │  │ Agent 2  │  │ Agent N  │
                         │          │  │          │  │          │
                         │ Workers  │  │ Workers  │  │ Workers  │
                         │ Metrics  │  │ Metrics  │  │ Metrics  │
                         └────┬─────┘  └────┬─────┘  └────┬─────┘
                              │             │             │
                              v             v             v
                         ┌──────────────────────────────────────┐
                         │       Target System Under Test       │
                         └──────────────────────────────────────┘
```

**상호작용 시퀀스**:

```
1. Agent 등록
   Agent ──[RegisterAgent(id, capabilities)]──> Controller
   Controller ──[RegisterResponse(accepted, config)]──> Agent

2. 테스트 실행 명령
   User ──[POST /api/v1/tests/{id}/run]──> Controller API
   Controller ──[StartTest(scenario, assigned_vusers)]──> Agent 1..N  (gRPC)

3. 실시간 메트릭 스트리밍 (양방향)
   Agent 1..N ──[MetricStream(rps, latency, errors)]──> Controller  (1초 간격)
   Controller ──[AggregatedMetrics]──> Web Dashboard  (WebSocket)

4. 제어 명령 (테스트 중)
   Controller ──[StopTest / AdjustVUsers]──> Agent  (gRPC)

5. 테스트 완료
   Agent ──[TestComplete(final_metrics)]──> Controller
   Controller ──[결과 저장]──> PostgreSQL
   Controller ──[TestFinished event]──> Web Dashboard  (WebSocket)

6. 헬스체크 (상시)
   Controller ──[HealthCheck ping]──> Agent  (10초 간격)
   Agent ──[HealthResponse(status, load)]──> Controller
```

### 3.2 Agent 모드 분리

기존 `omnitest run` (standalone 모드)를 유지하면서, agent 모드를 추가한다:

```bash
# Standalone 모드 (Cycle 1 그대로)
$ omnitest run load-test.yaml

# Agent 모드 (Cycle 2 신규) - Controller에 연결하여 명령 대기
$ omnitest agent --controller=controller-host:9090
  → Agent registered: agent-abc123
  → Waiting for commands from controller...

# Agent 모드 + 이름 지정
$ omnitest agent --controller=controller-host:9090 --name="agent-seoul-01"

# Controller 서버 시작
$ omnitest controller --port=9090 --db-url=postgres://...
  → Controller started on :9090
  → API server on :8080
  → Dashboard on :3000
```

### 3.3 웹 대시보드 핵심 3개 화면 (와이어프레임)

#### 화면 1: 테스트 목록/관리

```
┌─────────────────────────────────────────────────────────────┐
│  OmniTest                            [Agents: 3 online]    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Tests                                    [+ New Test]      │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Name              Status     Last Run     Duration  │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ user-api-load     ● Running  2026-03-17   2m/5m     │    │
│  │ payment-stress    ○ Idle     2026-03-16   10m       │    │
│  │ search-benchmark  ✓ Passed   2026-03-15   3m        │    │
│  │ cart-regression   ✗ Failed   2026-03-14   5m        │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  Recent Results                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Test              RPS     P99      Errors   Status  │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ search-benchmark  2,340   89ms     0.02%    PASS    │    │
│  │ cart-regression   1,120   520ms    3.2%     FAIL    │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### 화면 2: 실시간 메트릭 차트 (테스트 실행 중)

```
┌─────────────────────────────────────────────────────────────┐
│  ← Tests / user-api-load              ● Running  02:30/05:00│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─ RPS ──────────────────────────┐  ┌─ Error Rate ───────┐│
│  │     ╱╲    ╱╲                   │  │                     ││
│  │   ╱╱  ╲╲╱╱  ╲  ← 1,284/s     │  │  ─────── 0.1% ──── ││
│  │  ╱╱              current      │  │                     ││
│  │ ╱                              │  │                     ││
│  └────────────────────────────────┘  └─────────────────────┘│
│                                                             │
│  ┌─ Latency Distribution ─────────────────────────────────┐ │
│  │  P50: ████████░░░░░░░░ 38ms                            │ │
│  │  P95: ████████████████████░░░░ 118ms                   │ │
│  │  P99: ██████████████████████████████ 245ms             │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                             │
│  ┌─ Latency Over Time ───────────────────────────────────┐  │
│  │  300ms ┤                                              │  │
│  │  200ms ┤          ╱╲  P99                             │  │
│  │  100ms ┤  ────────    ──── P95                        │  │
│  │   50ms ┤  ════════════════ P50                        │  │
│  │     0  ┼────────────────────────────────> time         │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  VUsers: 100/100         [Stop Test]  [Download Report]     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### 화면 3: 에이전트 상태 모니터링

```
┌─────────────────────────────────────────────────────────────┐
│  OmniTest > Agents                                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Connected Agents: 3/3 online                               │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ Name           Status   VUsers   CPU    Mem   RPS   │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ agent-seoul-01 ● Active  334/500  45%   120MB 428   │    │
│  │ agent-seoul-02 ● Active  333/500  42%   115MB 421   │    │
│  │ agent-seoul-03 ● Active  333/500  48%   125MB 435   │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ Total                   1000     --    360MB  1,284 │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  Agent Health Timeline (last 1h)                            │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ seoul-01: ■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■ 100%      │    │
│  │ seoul-02: ■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■ 100%      │    │
│  │ seoul-03: ■■■■■■■■■■■■■■■■■■■■□□■■■■■■■■  93%      │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 3.4 REST API 엔드포인트 목록

**Base URL**: `http://controller:8080/api/v1`

| Method | Endpoint | 설명 |
|--------|----------|------|
| **테스트 관리** | | |
| `GET` | `/tests` | 테스트 목록 조회 (페이징, 필터) |
| `POST` | `/tests` | 테스트 생성 (YAML 시나리오 업로드) |
| `GET` | `/tests/{id}` | 테스트 상세 조회 |
| `PUT` | `/tests/{id}` | 테스트 수정 |
| `DELETE` | `/tests/{id}` | 테스트 삭제 |
| **테스트 실행** | | |
| `POST` | `/tests/{id}/run` | 테스트 실행 시작 |
| `POST` | `/tests/{id}/stop` | 실행 중인 테스트 중지 |
| `GET` | `/tests/{id}/status` | 테스트 실행 상태 조회 |
| **테스트 결과** | | |
| `GET` | `/tests/{id}/results` | 테스트 실행 결과 목록 |
| `GET` | `/results/{id}` | 특정 실행 결과 상세 |
| `GET` | `/results/{id}/metrics` | 실행 결과 메트릭 데이터 |
| `GET` | `/results/{id}/report` | HTML/JSON 리포트 다운로드 |
| **에이전트** | | |
| `GET` | `/agents` | 연결된 에이전트 목록 |
| `GET` | `/agents/{id}` | 에이전트 상세 정보 |
| `DELETE` | `/agents/{id}` | 에이전트 연결 해제 |
| **시스템** | | |
| `GET` | `/health` | 헬스체크 |
| `GET` | `/version` | 버전 정보 |

**WebSocket 엔드포인트**:

| Endpoint | 설명 |
|----------|------|
| `ws://controller:8080/ws/metrics/{test_run_id}` | 실행 중 실시간 메트릭 스트리밍 |
| `ws://controller:8080/ws/agents` | 에이전트 상태 실시간 업데이트 |
| `ws://controller:8080/ws/events` | 시스템 이벤트 (테스트 시작/완료/실패 등) |

### 3.5 Docker Compose 구성

```yaml
# docker-compose.yml
version: "3.8"

services:
  controller:
    image: omnitest/controller:latest
    ports:
      - "8080:8080"   # REST API + WebSocket
      - "9090:9090"   # gRPC (Agent 연결)
    environment:
      - DATABASE_URL=postgres://omnitest:omnitest@postgres:5432/omnitest?sslmode=disable
      - DASHBOARD_URL=http://dashboard:3000
    depends_on:
      postgres:
        condition: service_healthy

  agent-1:
    image: omnitest/agent:latest
    environment:
      - CONTROLLER_ADDR=controller:9090
      - AGENT_NAME=agent-1
    depends_on:
      - controller

  agent-2:
    image: omnitest/agent:latest
    environment:
      - CONTROLLER_ADDR=controller:9090
      - AGENT_NAME=agent-2
    depends_on:
      - controller

  agent-3:
    image: omnitest/agent:latest
    environment:
      - CONTROLLER_ADDR=controller:9090
      - AGENT_NAME=agent-3
    depends_on:
      - controller

  dashboard:
    image: omnitest/dashboard:latest
    ports:
      - "3000:3000"
    environment:
      - API_URL=http://controller:8080

  postgres:
    image: postgres:16-alpine
    environment:
      - POSTGRES_USER=omnitest
      - POSTGRES_PASSWORD=omnitest
      - POSTGRES_DB=omnitest
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U omnitest"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

**사용법**:
```bash
# 풀스택 배포
$ docker-compose up -d

# 에이전트 스케일 아웃
$ docker-compose up -d --scale agent=5

# 상태 확인
$ curl http://localhost:8080/api/v1/health
$ curl http://localhost:8080/api/v1/agents

# 웹 대시보드 접속
$ open http://localhost:3000
```

---

## 4. Rabbit Holes

| 위험 요소 | 사전 결정 | 근거 |
|-----------|-----------|------|
| **gRPC 스트리밍 백프레셔** | Agent 측 로컬 버퍼(in-memory ring buffer, 최근 60초분) + Controller 수신 불가 시 최신 데이터만 유지, 오래된 데이터 드롭 | 메트릭 유실보다 Agent OOM이 더 치명적. Locust도 유사 전략 사용 |
| **DB 마이그레이션 도구** | `golang-migrate/migrate` 사용. 마이그레이션 파일은 `migrations/` 디렉토리에 SQL로 관리 | ORM 대신 `sqlc`로 타입 안전 쿼리 생성. 마이그레이션과 쿼리 분리 |
| **WebSocket 연결 관리** | `gorilla/websocket` 사용. 클라이언트당 goroutine 1개. 하트비트 30초 간격. 재연결은 프론트엔드 책임 | MVP에서는 단순 구조 우선. 대규모 연결(1000+)은 Cycle 2 범위 외 |
| **gRPC proto 스키마 버전 관리** | `buf` 린터 + 브레이킹 체인지 감지. proto 파일은 `proto/` 디렉토리에 관리 | proto 호환성 깨지면 Agent 롤링 업데이트 불가. CI에서 자동 검증 |
| **메트릭 집계 정확성** | Controller에서 Agent별 HDR Histogram 병합 시 `Merge()` 사용. T-Digest는 과도 | HDR Histogram의 `Merge()`가 백분위 정확도를 보장. 분산 환경에서 검증된 접근 |
| **Agent 재연결/복구** | Agent가 Controller 연결 끊김 감지 시 exponential backoff(1s→2s→4s...→30s max)로 재연결. 테스트 실행 중 연결 끊기면 로컬 메트릭 버퍼링 후 재연결 시 전송 | Agent가 Controller보다 먼저 시작되는 경우 대응. Docker Compose `depends_on`만으로 부족 |
| **Controller 단일 장애점** | Cycle 2에서는 단일 Controller만 지원. HA는 Cycle 3+에서 다룸 | 복잡도 통제. MVP 분산 테스트에서 Controller HA는 과도한 범위 |
| **프론트엔드 빌드 전략** | React SPA를 Controller 바이너리에 `embed.FS`로 임베딩하는 방안 검토, 단 MVP에서는 별도 Docker 이미지로 분리 | 단일 바이너리 목표와 개발 속도 간 트레이드오프. Docker Compose 배포이므로 분리가 개발 편의성 높음 |
| **Agent → Controller 인증** | Cycle 2에서는 인증 없음. 같은 Docker 네트워크 내 통신 전제 | RBAC/인증은 Cycle 4 스코프. 내부 네트워크에서 우선 동작하는 것이 목표 |

---

## 5. No-Gos

이번 Cycle 2에서 **명시적으로 하지 않을 것**:

| 항목 | 이유 |
|------|------|
| **Kubernetes 네이티브 에이전트 관리** | Cycle 3 스코프. K8s client-go, CRD, Operator는 복잡도가 높아 Docker Compose 기반 검증 후 진행 |
| **CI/CD 통합 (GitHub Actions 등)** | Cycle 3 스코프. REST API가 먼저 안정화되어야 CI 연동 가능 |
| **인증/권한 (RBAC)** | Cycle 4 스코프. 내부 네트워크 전제로 우선 동작. 보안은 별도 사이클에서 체계적으로 |
| **Go 스크립트 지원 (Yaegi)** | Cycle 3 스코프. YAML 시나리오만으로 Cycle 2 기능 검증 가능 |
| **프로토콜 확장 (gRPC/WebSocket 부하 테스트)** | HTTP/HTTPS 부하 테스트만 지원. 프로토콜 확장은 추후 |
| **시계열 DB (VictoriaMetrics)** | PostgreSQL + JSON 컬럼으로 메트릭 요약 저장. 원시 시계열 데이터는 Cycle 3에서 도입 |
| **Controller HA / 수평 확장** | 단일 Controller로 충분. HA는 K8s 배포 시(Cycle 3+) 검토 |
| **테스트 스케줄링 (Cron)** | Cycle 3 스코프. 수동 실행으로 충분히 검증 가능 |
| **PDF 리포트** | HTML/JSON 리포트로 충분. PDF 변환은 headless Chrome 의존성 추가로 복잡도 증가 |
| **에이전트 오토스케일링** | Docker Compose `--scale`로 수동 스케일. 자동 스케일은 K8s 전제 |

---

## 6. UX

### 6.1 웹 대시보드 톤앤매너

**디자인 원칙**:
- **데이터 중심**: 장식적 요소 최소화, 메트릭과 차트가 주인공
- **어두운 테마 기본**: 모니터링 도구의 관례. 눈 피로도 감소, 차트 가독성 향상
- **상태 즉시 인식**: 색상 코드로 상태를 즉시 전달
  - 초록(`#22c55e`): 정상/통과
  - 빨강(`#ef4444`): 오류/실패
  - 주황(`#f59e0b`): 경고/주의
  - 파랑(`#3b82f6`): 실행 중/활성
  - 회색(`#6b7280`): 비활성/대기
- **정보 밀도**: 한 화면에 핵심 메트릭 한눈에 파악. 스크롤 최소화
- **반응형**: 1280px+ 데스크톱 우선. 태블릿/모바일은 범위 외

**기술 스택**: React 18 + TypeScript + Recharts (차트) + shadcn/ui (컴포넌트) + TanStack Query (데이터 페칭) + Zustand (상태 관리)

**참고 모델**: Grafana의 대시보드 레이아웃 + Vercel의 깔끔한 UI + k6 Cloud의 메트릭 표현

### 6.2 API 응답 형식

모든 REST API는 일관된 envelope 형식을 사용한다:

**성공 응답**:
```json
{
  "data": {
    "id": "test-abc123",
    "name": "user-api-load",
    "status": "running",
    "created_at": "2026-03-17T14:30:00Z"
  },
  "meta": {
    "request_id": "req-xyz789"
  }
}
```

**목록 응답** (페이징):
```json
{
  "data": [
    { "id": "test-abc123", "name": "user-api-load" },
    { "id": "test-def456", "name": "payment-stress" }
  ],
  "meta": {
    "total": 42,
    "page": 1,
    "per_page": 20,
    "request_id": "req-xyz789"
  }
}
```

**에러 응답**:
```json
{
  "error": {
    "code": "TEST_NOT_FOUND",
    "message": "Test with ID 'test-abc123' not found",
    "details": "Verify the test ID and try again"
  },
  "meta": {
    "request_id": "req-xyz789"
  }
}
```

**원칙**:
- 모든 응답에 `request_id` 포함 (디버깅용)
- 날짜/시간은 RFC3339 (UTC)
- ID는 prefix + nanoid (`test-`, `agent-`, `result-`)
- 빈 목록은 `[]` (null 아님)
- HTTP 상태 코드: 200(성공), 201(생성), 400(잘못된 요청), 404(없음), 409(충돌), 500(내부 오류)

### 6.3 에러 핸들링 패턴 (REST)

**에러 코드 체계**:

| HTTP | 에러 코드 | 설명 |
|------|-----------|------|
| 400 | `INVALID_SCENARIO` | YAML 시나리오 파싱 실패 |
| 400 | `INVALID_PARAMETER` | 요청 파라미터 유효성 검증 실패 |
| 404 | `TEST_NOT_FOUND` | 테스트 ID 없음 |
| 404 | `RESULT_NOT_FOUND` | 결과 ID 없음 |
| 404 | `AGENT_NOT_FOUND` | 에이전트 ID 없음 |
| 409 | `TEST_ALREADY_RUNNING` | 이미 실행 중인 테스트 |
| 409 | `NO_AGENTS_AVAILABLE` | 연결된 에이전트 없음 |
| 500 | `INTERNAL_ERROR` | 서버 내부 오류 |
| 503 | `AGENT_UNAVAILABLE` | 에이전트 통신 실패 |

**에러 응답 예시**:
```json
{
  "error": {
    "code": "NO_AGENTS_AVAILABLE",
    "message": "Cannot start test: no agents are connected",
    "details": "Start at least one agent with 'omnitest agent --controller=host:9090' or add agent services to docker-compose.yml"
  },
  "meta": {
    "request_id": "req-xyz789"
  }
}
```

### 6.4 CLI agent 모드 사용법

```bash
# 기본 Agent 모드 실행
$ omnitest agent --controller=controller-host:9090
→ Connecting to controller at controller-host:9090...
→ Agent registered: agent-a1b2c3 (name: hostname-default)
→ Waiting for test commands...

# 이름 지정
$ omnitest agent --controller=controller-host:9090 --name="seoul-agent-01"
→ Connecting to controller at controller-host:9090...
→ Agent registered: agent-d4e5f6 (name: seoul-agent-01)
→ Waiting for test commands...

# 테스트 실행 시 (Controller가 명령 전송 시 자동 출력)
→ [14:30:22] Received test: user-api-load (334 VUsers assigned)
→ [14:30:22] Starting workers...
→ [14:30:23] Running: 334/334 VUsers active
→ [14:35:22] Test completed. Sent final metrics to controller.
→ Waiting for test commands...

# 연결 실패 시 자동 재연결
→ Connection lost. Reconnecting in 1s...
→ Reconnecting in 2s...
→ Reconnected to controller.

# Agent 모드 종료
Ctrl+C
→ Graceful shutdown: finishing active workers...
→ Sent final metrics. Agent disconnected.
```

**Agent CLI 플래그**:

| 플래그 | 기본값 | 설명 |
|--------|--------|------|
| `--controller` | (필수) | Controller gRPC 주소 (host:port) |
| `--name` | hostname | 에이전트 식별 이름 |
| `--max-vusers` | 1000 | 이 에이전트가 수용할 최대 VUser 수 |
| `--labels` | - | 에이전트 라벨 (key=value, 반복 가능) |
| `--log-level` | info | 로그 레벨 (debug, info, warn, error) |

---

## 7. Acceptance Criteria (수용 기준)

1. **AC-1**: `docker-compose up`으로 Controller, Agent 3대, PostgreSQL, 웹 대시보드가 모두 기동되고, `curl /api/v1/health`가 200을 반환한다.

2. **AC-2**: REST API(`POST /tests`, `POST /tests/{id}/run`)로 테스트를 생성하고 실행하면, 3대의 Agent에 부하가 분배되어 동시에 실행되고, Controller가 집계된 메트릭(RPS, P50/P95/P99, Error Rate)을 반환한다.

3. **AC-3**: 웹 대시보드에서 실행 중인 테스트의 실시간 메트릭 차트(RPS, Latency, Error Rate)가 WebSocket을 통해 1초 간격으로 갱신된다.

4. **AC-4**: 웹 대시보드의 에이전트 모니터링 화면에서 연결된 에이전트 목록, 상태(online/offline), 할당된 VUser 수가 실시간으로 표시된다.

5. **AC-5**: `omnitest agent --controller=host:9090`으로 Agent를 추가하면 Controller가 자동으로 인식하고, 이후 테스트 실행 시 새 Agent에도 부하가 분배된다.

---

## 8. Quick Gate 1줄 요약

> `docker-compose up`으로 Controller + Agent 3대 + 웹 대시보드를 배포하고, REST API로 분산 부하 테스트를 실행하며, WebSocket 실시간 차트로 에이전트별 메트릭을 모니터링하는 분산 성능 테스트 플랫폼
