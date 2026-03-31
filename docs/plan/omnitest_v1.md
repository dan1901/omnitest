---
title: OmniTest
version: v2
date: 2026-03-17
goal_achievement: 91%
status: finalized
methodology: Shape Up
team: Comandos (AI Agent)
---

# OmniTest 기획서 v2 (Shape Up + AI Agent)

## 1. 서비스 개요

**서비스명**: OmniTest

**한 줄 정의**: Go 기반의 클라우드 네이티브 분산 성능 테스트 플랫폼 -- nGrinder의 엔터프라이즈 기능에 현대적 DX(Developer Experience)를 결합한 차세대 부하 테스트 도구

**핵심 가치 제안**:
- 단일 바이너리 배포로 5분 내 설치 완료 (nGrinder 대비 설치 복잡도 90% 감소)
- YAML/코드 기반 테스트 시나리오로 Git 연동 및 CI/CD 파이프라인 통합
- Kubernetes 네이티브 에이전트 오케스트레이션으로 탄력적 부하 생성
- 실시간 스트리밍 대시보드와 AI 기반 성능 이상 감지

## 2. 문제 정의 & 기회

### 2.1 nGrinder의 주요 한계점

| 영역 | 문제 | 심각도 |
|------|------|--------|
| **설치/운영** | Java/Tomcat 기반으로 JVM 설정, WAR 배포, DB 설정 등 복잡한 설치 과정 | 높음 |
| **리소스 효율** | JVM 기반 에이전트의 높은 메모리 사용량 (에이전트당 1-2GB), GC pause로 인한 부하 생성 불안정 | 높음 |
| **스크립트 작성** | Groovy/Jython 스크립트 학습 곡선, IDE 지원 부족, 디버깅 어려움 | 중간 |
| **클라우드 지원** | 컨테이너/K8s 네이티브 지원 부재, 클라우드 환경에서의 오토스케일링 미지원 | 높음 |
| **CI/CD 통합** | REST API 존재하나 CLI 도구 부재, 파이프라인 통합이 번거로움 | 중간 |
| **모니터링** | 폴링 기반 모니터링, 실시간성 부족, 커스텀 메트릭 제한적 | 중간 |
| **유지보수** | 최근 업데이트 빈도 감소, 커뮤니티 활동 둔화 | 중간 |

### 2.2 시장 기회

| 도구 | 장점 | 약점 | 포지션 |
|------|------|------|--------|
| **k6 (Grafana)** | Go 기반, 우수한 DX, CLI 중심 | 분산 테스트 유료(k6 Cloud), 웹 UI 없음 | 개발자 중심 |
| **Locust** | Python 기반, 쉬운 스크립트 | 성능 한계(Python GIL), 엔터프라이즈 기능 부족 | 프로토타이핑 |
| **Gatling** | Scala DSL, 상세 리포트 | 학습 곡선, 엔터프라이즈 유료 | 엔터프라이즈 |
| **JMeter** | 풍부한 플러그인 | 무거움, UI 구식, 스크립트 XML | 레거시 |
| **nGrinder** | 분산 테스트, 웹 UI | 위에서 언급한 한계점들 | 엔터프라이즈/한국 중심 |

**OmniTest의 포지셔닝**: k6의 우수한 DX + nGrinder의 엔터프라이즈 분산 테스트 역량 + 클라우드 네이티브 아키텍처를 결합한 포지션. 특히 "무료 오픈소스에서 분산 테스트까지 가능한 도구"라는 차별점을 확보.

## 3. 타겟 사용자

### 페르소나 1: 백엔드 개발자 (Primary)
- **역할**: 스타트업/중견기업 백엔드 개발자 (3-7년차)
- **환경**: Kubernetes 기반 MSA 환경
- **니즈**: PR 머지 전 성능 회귀 테스트를 CI/CD에서 자동 실행
- **현재 페인**: k6는 분산 테스트가 유료, nGrinder는 설치가 너무 복잡, JMeter는 스크립트가 XML이라 Git 관리 어려움
- **성공 기준**: 10분 내 첫 테스트 실행, CI 파이프라인 통합 30분 내 완료

### 페르소나 2: QA/성능 엔지니어 (Secondary)
- **역할**: 대기업 성능 테스트 전담 엔지니어 (5-10년차)
- **환경**: 온프레미스 + 클라우드 하이브리드
- **니즈**: 수백 대 에이전트를 조율하여 대규모 부하 테스트 수행, 상세 리포트 자동 생성
- **현재 페인**: nGrinder 에이전트 관리 수작업 많음, 리포트 커스터마이징 한계
- **성공 기준**: 1000+ 에이전트 안정적 운영, 경영진 보고용 리포트 자동 생성

### 페르소나 3: DevOps/SRE (Tertiary)
- **역할**: 플랫폼 엔지니어 (3-5년차)
- **환경**: 클라우드 네이티브 (K8s, Terraform)
- **니즈**: 성능 테스트 인프라를 코드로 관리(IaC), 필요 시 자동 스케일
- **현재 페인**: 기존 도구들의 인프라 관리가 선언적이지 않음
- **성공 기준**: Helm chart로 5분 내 배포, HPA로 에이전트 오토스케일

## 4. 핵심 기능

### 4.1 언어 선택: Go (Golang)

**후보군 비교 분석**:

| 기준 | Go | Rust | Python+C | Java (현행) |
|------|-----|------|----------|-------------|
| **동시성 모델** | goroutine (경량 스레드, ~2KB) | async/await + OS 스레드 | asyncio + C 확장 | 스레드 (~1MB) |
| **메모리 효율** | 좋음 (GC 있으나 경량) | 최고 (Zero-cost abstraction) | 보통 (GIL 제약) | 나쁨 (JVM 오버헤드) |
| **빌드/배포** | 단일 바이너리, 크로스 컴파일 | 단일 바이너리, 크로스 컴파일 | 의존성 복잡 | JAR/WAR, JVM 필요 |
| **HTTP/네트워크** | net/http 표준 라이브러리 우수 | hyper/reqwest 우수 | requests/aiohttp 우수 | Netty/OkHttp |
| **학습 곡선** | 낮음 (심플한 문법) | 높음 (Ownership, Lifetime) | 낮음 | 중간 |
| **에코시스템 (인프라 도구)** | 최고 (K8s, Docker, Terraform 등) | 성장 중 | 풍부 (ML/데이터) | 성숙 |
| **기여자 확보 용이성** | 높음 | 중간 | 높음 | 높음 |
| **에이전트당 VUser 용량** | ~10,000+ (goroutine) | ~10,000+ | ~1,000 (GIL) | ~3,000-5,000 |

**Go 선택 근거**:
1. **goroutine 기반 동시성**: 에이전트당 10,000+ 가상 사용자를 2KB/goroutine으로 생성 가능
2. **단일 바이너리 배포**: `curl | sh` 한 줄로 설치 가능. JVM 의존성 완전 제거
3. **클라우드 네이티브 에코시스템**: K8s, Prometheus, gRPC 등 인프라 도구의 de facto 언어
4. **커뮤니티 확장성**: Rust 대비 기여자 진입 장벽이 낮아 오픈소스 성장에 유리
5. **검증된 사례**: k6(Grafana), Vegeta, Hey 등 성능 테스트 도구들이 Go로 성공적으로 구현됨

**Rust를 선택하지 않은 이유**: 극한의 성능이 필요한 에이전트 코어에는 Rust가 유리하나, 전체 시스템을 고려하면 Go의 생산성과 에코시스템이 더 큰 이점을 제공. 향후 에이전트의 핵심 부하 생성 엔진만 Rust로 작성하는 하이브리드 접근도 고려 가능.

### 4.2 기능 목록 (MoSCoW 우선순위)

**Must Have (MVP)**:

| ID | 기능 | 설명 |
|----|------|------|
| M1 | CLI 기반 테스트 실행 | `omnitest run test.yaml` 명령으로 즉시 테스트 실행 |
| M2 | YAML 기반 테스트 시나리오 | 선언적 테스트 정의 (URL, 메서드, 헤더, 바디, 동시성, 지속시간) |
| M3 | Go 스크립트 테스트 | 복잡한 시나리오를 Go 코드로 작성 (플러그인 형태) |
| M4 | Controller-Agent 아키텍처 | 중앙 컨트롤러가 다수 에이전트를 조율하여 분산 부하 생성 |
| M5 | 실시간 메트릭 수집 | RPS, 응답시간(p50/p95/p99), 에러율, 동시접속 수 실시간 수집 |
| M6 | 웹 대시보드 | 테스트 생성/실행/모니터링/결과 조회용 웹 UI |
| M7 | REST API | 모든 기능의 API 제공 (CLI와 웹 UI 모두 API 사용) |
| M8 | HTML/JSON 리포트 | 테스트 완료 후 자동 리포트 생성 |

**Should Have (v1.0)**:

| ID | 기능 | 설명 |
|----|------|------|
| S1 | Kubernetes 네이티브 에이전트 | K8s Job/Pod으로 에이전트 자동 생성/삭제 |
| S2 | CI/CD 통합 | GitHub Actions, GitLab CI, Jenkins 플러그인/액션 |
| S3 | 테스트 스케줄링 | Cron 기반 반복 테스트 실행 |
| S4 | 성능 임계값 검증 | p99 < 200ms 같은 조건 기반 pass/fail 판정 |
| S5 | 사용자 인증/권한 | RBAC 기반 멀티 테넌시 |
| S6 | Prometheus 메트릭 익스포터 | Grafana 연동을 위한 메트릭 노출 |

**Could Have (v1.x)**:

| ID | 기능 | 설명 |
|----|------|------|
| C1 | AI 성능 분석 | 병목 지점 자동 감지, 성능 이상 알림 |
| C2 | 시나리오 레코더 | 브라우저 트래픽 녹화 -> 테스트 시나리오 자동 생성 |
| C3 | 프로토콜 확장 | gRPC, WebSocket, GraphQL, MQTT 지원 |
| C4 | 비교 리포트 | 이전 테스트 대비 성능 변화 자동 비교 |
| C5 | Terraform Provider | IaC로 테스트 인프라 관리 |

**Won't Have (현재 범위 외)**:

| ID | 기능 | 이유 |
|----|------|------|
| W1 | 브라우저 기반 부하 테스트 | 리소스 효율성 낮음, 프로토콜 레벨 집중 |
| W2 | 모바일 앱 테스트 | 범위 초과, 별도 도구 영역 |
| W3 | Chaos Engineering | 별도 도메인 (LitmusChaos 등 연동으로 대체) |

## 5. 사용자 플로우

### 5.1 시나리오 A: 개발자 첫 사용

```
[설치] curl -sSL install.omnitest.io | sh
  |
  v
[테스트 작성] vi load-test.yaml
  |   targets:
  |     - url: https://api.example.com/users
  |       method: GET
  |       headers:
  |         Authorization: "Bearer ${TOKEN}"
  |   scenarios:
  |     - name: "기본 부하 테스트"
  |       vusers: 100
  |       duration: 5m
  |       ramp_up: 30s
  |   thresholds:
  |     - metric: http_req_duration_p99
  |       condition: "< 200ms"
  |
  v
[로컬 실행] omnitest run load-test.yaml
  |
  v
[실시간 출력] 터미널에 RPS, 응답시간, 에러율 실시간 표시
  |
  v
[결과 확인] 자동 생성된 report.html 확인
  |
  v
[CI 통합] GitHub Actions에 omnitest-action 추가
```

### 5.2 시나리오 B: 분산 테스트

```
[Controller 배포] helm install omnitest-controller omnitest/controller
  |
  v
[웹 UI 접속] https://omnitest.company.com
  |
  v
[테스트 생성] 웹 UI에서 테스트 시나리오 구성 또는 Git 리포에서 import
  |
  v
[에이전트 설정] Agent Pool 설정 (K8s: 자동 스케일 / Bare Metal: 수동 등록)
  |
  v
[테스트 실행] 10개 에이전트 x 1,000 VUser = 10,000 동시 부하
  |
  v
[실시간 모니터링] 웹 대시보드에서 실시간 차트 확인
  |  - 에이전트별 상태
  |  - 집계 메트릭 (RPS, latency 분포, 에러율)
  |  - 타겟 서버 리소스 (연동 시)
  |
  v
[리포트 생성] PDF/HTML 리포트 자동 생성, Slack/이메일 알림
```

## 6. 아키텍처 설계

### 6.1 전체 아키텍처

```
                         +------------------+
                         |   Web UI (React)  |
                         +--------+---------+
                                  |
                                  | REST/WebSocket
                                  v
+----------+           +-------------------+           +-----------------+
|   CLI    +---------->+   Controller      +<--------->+ Time-Series DB  |
| (omnitest)|   gRPC   |                   |           | (VictoriaMetrics|
+----------+           |  - API Server     |           |  / InfluxDB)    |
                        |  - Scheduler      |           +-----------------+
                        |  - Agent Manager  |
                        |  - Report Engine  |           +-----------------+
                        +---+-----+----+---+           |   PostgreSQL    |
                            |     |    |               | (메타데이터/설정) |
                     gRPC   |     |    |   gRPC        +-----------------+
               +------------+     |    +----------+
               v                  v               v
        +-----------+     +-----------+    +-----------+
        |  Agent 1  |     |  Agent 2  |    |  Agent N  |
        |           |     |           |    |           |
        | [Worker]  |     | [Worker]  |    | [Worker]  |
        | [Worker]  |     | [Worker]  |    | [Worker]  |
        | [Metrics] |     | [Metrics] |    | [Metrics] |
        +-----------+     +-----------+    +-----------+
              |                 |                |
              v                 v                v
        +-------------------------------------------+
        |          Target System Under Test          |
        +-------------------------------------------+
```

### 6.2 핵심 컴포넌트

**Controller (중앙 제어)**:
- **API Server**: REST API + WebSocket (실시간 스트리밍)
- **Scheduler**: 테스트 스케줄링, 에이전트 할당, 부하 분배 로직
- **Agent Manager**: 에이전트 등록/헬스체크/제거. K8s 모드에서는 client-go로 Pod 직접 관리
- **Report Engine**: 메트릭 집계, 리포트 렌더링 (HTML 템플릿 + PDF 변환)

**Agent (부하 생성)**:
- **Worker Pool**: goroutine 기반 가상 사용자 풀
- **Scenario Engine**: YAML 파싱 또는 Go 플러그인 로드하여 테스트 로직 실행
- **Metrics Collector**: 요청별 latency, status code 등 수집. HDR Histogram으로 백분위 계산
- **gRPC Stream**: Controller에 메트릭을 1초 간격으로 스트리밍 전송

**통신 프로토콜**:
- Controller <-> Agent: gRPC (양방향 스트리밍)
- Controller <-> Web UI: REST + WebSocket
- Controller <-> CLI: gRPC

### 6.3 기술 스택

| 계층 | 기술 | 선택 이유 |
|------|------|-----------|
| **에이전트/컨트롤러** | Go 1.22+ | 동시성, 단일 바이너리, K8s 에코시스템 |
| **통신** | gRPC + Protocol Buffers | 효율적 양방향 스트리밍, 타입 안전성 |
| **웹 UI** | React + TypeScript + Recharts | 실시간 차트, 컴포넌트 재사용성 |
| **메타데이터 DB** | PostgreSQL | 안정성, JSON 지원, 널리 사용됨 |
| **시계열 DB** | VictoriaMetrics | Prometheus 호환, 고성능, 리소스 효율적 |
| **메시지 큐 (선택)** | NATS | 경량, Go 네이티브, 에이전트 디스커버리 |
| **컨테이너화** | Docker + Helm Chart | K8s 배포 표준 |
| **빌드** | GoReleaser + GitHub Actions | 크로스 플랫폼 바이너리 자동 빌드/배포 |

## 7. nGrinder 대비 차별점

| 영역 | nGrinder | OmniTest | 개선 효과 |
|------|----------|----------|-----------|
| **설치** | JVM + Tomcat + DB 설정 (30분+) | 단일 바이너리 다운로드 (5분) | 설치 시간 85% 단축 |
| **리소스** | 에이전트당 1-2GB RAM | 에이전트당 100-200MB RAM | 메모리 사용량 90% 감소 |
| **VUser 밀도** | 에이전트당 ~3,000 | 에이전트당 ~10,000+ | 3배 이상 효율 |
| **스크립트** | Groovy/Jython (학습 필요) | YAML (간단) + Go (고급) | 진입 장벽 대폭 감소 |
| **CI/CD** | REST API만 제공 | CLI + GitHub Action + 임계값 검증 | 네이티브 CI/CD 통합 |
| **클라우드** | 수동 에이전트 배포 | K8s HPA 기반 자동 스케일 | 운영 자동화 |
| **모니터링** | 폴링 기반, 새로고침 | WebSocket 실시간 스트리밍 | 실시간성 확보 |
| **확장성** | 모놀리식 Controller | 마이크로서비스 가능, gRPC 기반 | 수평 확장 용이 |

## 8. 비즈니스 모델 (오픈코어)

**Community Edition (무료/오픈소스 - Apache 2.0)**:
- CLI 기반 테스트 실행, YAML/Go 스크립트
- Controller-Agent 분산 테스트 (에이전트 수 제한 없음)
- 웹 대시보드 (기본), REST API
- HTML/JSON 리포트, Prometheus 메트릭 익스포터

**Enterprise Edition (유료 구독)**:
- SSO/LDAP 인증, RBAC 멀티 테넌시, 감사 로그
- 고급 리포트 (PDF, 비교 리포트, 트렌드 분석)
- AI 기반 성능 분석 및 이상 감지
- 클라우드 프로바이더 통합 (AWS/GCP/Azure 오토스케일)
- SLA 기반 기술 지원

## 9. KPI & 성공 지표

| KPI | 목표 (MVP 출시 후 6개월) | 측정 방법 |
|-----|-------------------------|-----------|
| GitHub Stars | 1,000+ | GitHub API |
| 월간 다운로드 | 5,000+ | GoReleaser / Docker Hub |
| 커뮤니티 기여자 | 20+ | GitHub Contributors |
| 첫 테스트 실행 시간 (TTFR) | 10분 이내 | 사용자 피드백 |
| 에이전트당 최대 VUser | 10,000+ | 벤치마크 |

## 10. 리스크 & 대응 방안

| 리스크 | 확률 | 영향도 | 대응 방안 |
|--------|------|--------|-----------|
| k6이 분산 테스트를 오픈소스화 | 중간 | 높음 | YAML 시나리오 + 웹 UI + 엔터프라이즈 기능으로 차별화 유지 |
| Go 스크립트 플러그인 보안 이슈 | 중간 | 높음 | WASM 샌드박스 또는 제한된 Go 런타임(Yaegi) 사용 |
| 대규모 분산 테스트 시 메트릭 유실 | 중간 | 중간 | 에이전트 로컬 버퍼 + 재전송 로직 |
| 오픈소스 커뮤니티 성장 부진 | 중간 | 중간 | 콘텐츠 마케팅, 마이그레이션 가이드 제공 |
| PostgreSQL 단일 장애점 | 낮음 | 높음 | 임베디드 SQLite 옵션 (단일 노드 모드) |

## 11. 개발 방법론 & 팀 구조

### 11.1 Shape Up 프로세스

Shape Up 방법론을 AI 에이전트 팀에 맞춰 적용합니다.

**사이클 구조 (기존 6주 → 4주+1주로 단축)**:

```
┌──────────────────────────────────┐  ┌──────────────┐
│        Build Phase (4주)          │  │ Cooldown (1주)│
│                                  │  │              │
│  W1: 인터페이스 정의 + 병렬 착수   │  │ 버그 수정     │
│  W2: 핵심 구현 (에이전트 병렬)     │  │ 기술 부채     │
│  W3: 통합 + 테스트               │  │ 테스트 보강   │
│  W4: 안정화 + 문서화             │  │ 리서치/실험   │
│                                  │  │              │
└──────────────────────────────────┘  └──────────────┘
                    = 1 사이클 (5주)
```

**사이클 단축 근거**:
- 에이전트 팀의 코드 생성 속도는 인간 팀의 3-5배 (반복적 구현 작업 기준)
- 4개 에이전트 병렬 작업으로 처리량 극대화
- Shaping과 통합 검증은 인간 리드가 수행하므로 무한히 단축 불가
- 쿨다운도 2주에서 1주로 단축 (에이전트가 기술 부채 정리를 빠르게 처리)

### 11.2 Shape → Bet → Build 프로세스

#### Shape (사이클 시작 1주 전, 인간 리드 주도)

```
인간 리드가 수행:
  1. 다음 사이클에서 해결할 문제 정의
  2. 솔루션의 방향성(Fat Marker Sketch 수준)을 Pitch 문서로 작성
  3. Appetite 결정 (Small Batch: 1-2주 / Big Batch: 4주)
  4. Rabbit Holes (위험 요소) 식별 및 제거

AI 에이전트가 보조:
  - 기술적 실현 가능성 사전 검증 (프로토타입)
  - 유사 오픈소스 구현체 조사 및 요약
  - 의존성/호환성 리스크 분석
```

**Pitch 문서 템플릿**:
```markdown
# Pitch: {기능명}
## Problem (문제)
## Appetite (시간 예산): Small Batch (1-2주) / Big Batch (4주)
## Solution (방향성)
  - Fat Marker Sketch (추상적 설계)
  - 에이전트별 태스크 분해
## Rabbit Holes (위험 요소)
## No-Gos (이번에 하지 않을 것)
```

#### Bet (사이클 시작 직전, 인간 리드 결정)

- 후보 Pitch 목록 검토
- Big Batch 1개 + Small Batch 2-3개 조합이 이상적
- 선택되지 않은 Pitch는 버림 (다음 사이클에 재제안 가능)
- 일단 Bet하면 사이클 중간에 방향 전환 없음

#### Build (4주, 에이전트 팀 주도)

- **Week 1**: 인터페이스 정의 (proto, API spec) + 병렬 착수
- **Week 2-3**: 각 에이전트 독립 브랜치에서 병렬 구현
- **Week 4**: 통합 테스트, 안정화, 문서화

### 11.3 Hill Chart 기반 진행 추적

```
                    ▲
                   /|\
                  / | \
    Figuring Out /  |  \ Execution
    (탐색/설계)  /   |   \ (구현/완료)
               /    |    \
              /     |     \
─────────────/──────|──────\──────────
           시작    정상    완료
```

| 구간 | 주도 | 활동 |
|------|------|------|
| 오르막 (0-50%) | **인간 리드** | 문제 탐색, 아키텍처 결정, 핵심 접근법 확정 |
| 정상 (50%) | 인간 리드 승인 | "더 이상 미지의 영역이 없는가?" 확인 |
| 내리막 (50-100%) | **AI 에이전트 팀** | 핵심 구현, 테스트, 엣지케이스, 문서화 |

### 11.4 AI 에이전트 팀 구조 (Comandos)

| 역할 | 담당 | 인간/에이전트 |
|------|------|--------------|
| **Tech Lead** | 아키텍처 결정, Shape/Bet 주도, 코드 리뷰 최종 승인 | 인간 |
| **Core Engine Agent** | Controller/Agent 코어, gRPC 통신, goroutine 워커 풀 | AI 에이전트 |
| **API/Integration Agent** | REST API, CLI, CI/CD 연동, Helm Chart | AI 에이전트 |
| **Frontend Agent** | React 웹 대시보드, 실시간 차트, UI 컴포넌트 | AI 에이전트 |
| **Infra/Test Agent** | Docker, K8s, E2E 테스트, 벤치마크, 기술 부채 정리 | AI 에이전트 |
| **Docs/DX Agent** | 문서화, 예제, 마이그레이션 가이드, README | AI 에이전트 |

**협업 패턴**:

```
                    Tech Lead (인간)
                    ┌─────────────┐
                    │  Shape/Bet  │
                    │  코드 리뷰   │
                    │  통합 판단   │
                    └──────┬──────┘
                           │ Pitch 문서 + 태스크 분배
          ┌────────────────┼────────────────┐
          v                v                v
   ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
   │ Core Engine  │ │ API/Integ.   │ │ Frontend     │
   │ Agent        │ │ Agent        │ │ Agent        │
   └──────┬───────┘ └──────┬───────┘ └──────┬───────┘
          └────────────────┼────────────────┘
                           │ PR 생성 (각자 독립 브랜치)
                           v
                    ┌──────────────┐
                    │ Infra/Test   │──> 통합 테스트 실행
                    │ Agent        │
                    └──────┬───────┘
                           v
                    ┌──────────────┐
                    │ Tech Lead    │──> 최종 리뷰 & 머지
                    └──────────────┘
```

**협업 규칙**:
1. **독립 브랜치 원칙**: 각 에이전트는 독립된 feature 브랜치에서 작업
2. **인터페이스 선행 정의**: Core Engine Agent가 proto/API 스키마를 먼저 생성 → 병렬 개발
3. **PR 기반 통합**: 에이전트 PR → Infra/Test 통합 테스트 → Tech Lead 최종 리뷰
4. **비동기 커뮤니케이션**: 공유 문서(ADR, API spec)를 통해 소통

### 11.5 인간 리드 vs 에이전트 역할 분담

| 활동 | 인간 리드 | AI 에이전트 팀 |
|------|-----------|---------------|
| 아키텍처 결정 | **주도** (ADR 작성) | 대안 리서치, 프로토타입 |
| 기능 Shaping | **주도** (방향성 정의) | 기술적 실현 가능성 검증 |
| 코드 구현 | 핵심 알고리즘 리뷰 | **주도** (병렬 구현) |
| 코드 리뷰 | **최종 승인** | 자동 린팅, 커버리지 확인 |
| 테스트 작성 | 통합 테스트 시나리오 설계 | **주도** (단위/E2E 구현) |
| 기술 부채 정리 | 우선순위 결정 | **주도** (쿨다운 기간) |
| 문서화 | 아키텍처 문서 검수 | **주도** (API 문서, 가이드) |

### 11.6 코드 리뷰 & 배포 프로세스

```
에이전트 작업 완료 → PR 생성
       │
       v
  자동 검증 (CI)
  ├─ golangci-lint
  ├─ 단위 테스트
  ├─ 빌드 확인
  └─ 커버리지 체크
       │
       v
  Infra/Test Agent → 통합 테스트 + 벤치마크
       │
       v
  Tech Lead → 최종 리뷰 (아키텍처, 보안, 엣지케이스)
       │
       v
  머지 → 자동 배포
  ├─ GoReleaser → 바이너리
  ├─ Docker 이미지 → Registry
  └─ Helm Chart 업데이트
```

---

## 12. 로드맵 (Shape Up 사이클 기반)

### 전체 타임라인

```
┌─────────┬────────┬─────────┬────────┬─────────┬────────┬─────────┬────────┐
│Cycle 1  │Cooldown│Cycle 2  │Cooldown│Cycle 3  │Cooldown│Cycle 4  │Cooldown│
│MVP Core │  1주   │분산 아키 │  1주   │K8s/CICD │  1주   │Enterprise│ 1주   │
│ (4주)   │        │텍처(4주) │        │ (4주)   │        │ (4주)    │       │
└─────────┴────────┴─────────┴────────┴─────────┴────────┴─────────┴────────┘
 Week 1-4   Week 5   Week 6-9  Week 10  Week11-14  Week 15  Week16-19 Week 20

총 기간: 약 20주 (5개월) — 기존 12개월 대비 약 58% 단축
```

### Cycle 1: MVP Core Engine (4주 Build + 1주 Cooldown)

**Pitch**: "개발자가 `omnitest run test.yaml` 한 줄로 로컬 부하 테스트를 실행하고, 터미널에서 실시간 결과를 확인할 수 있는 최소 핵심 엔진"

**Appetite**: Big Batch (4주)

**Scope**:
- CLI 프레임워크 (cobra 기반 `omnitest run`, `omnitest version`)
- YAML 시나리오 파서 (targets, scenarios, thresholds)
- goroutine 기반 가상 사용자 워커 풀
- HTTP 요청 엔진 (GET/POST/PUT/DELETE, 헤더, 바디)
- 메트릭 수집기 (RPS, latency p50/p95/p99, error rate, HDR Histogram)
- 터미널 실시간 출력 (TUI 기반 진행률 + 메트릭)
- JSON/HTML 리포트 생성
- 단일 바이너리 빌드 (GoReleaser)

**No-Gos**: 분산 테스트, 웹 UI, Go 스크립트, 데이터베이스

**Rabbit Holes**:
- HDR Histogram → `github.com/HdrHistogram/hdrhistogram-go` 사전 확정
- YAML 스키마 → MVP 최소 필드만 정의

**에이전트 태스크 분배**:

| 에이전트 | Week 1 | Week 2-3 | Week 4 |
|---------|--------|----------|--------|
| Core Engine | CLI 프레임워크, YAML 파서 | 워커 풀, HTTP 엔진, 메트릭 수집 | 안정화, 엣지케이스 |
| API/Integration | 설치 스크립트, GoReleaser | JSON 리포트 엔진 | CI 파이프라인 |
| Frontend | (불참) | - | - |
| Infra/Test | 프로젝트 구조, Makefile, CI | 단위 테스트, 벤치마크 | 통합 테스트, 성능 검증 |
| Docs/DX | README, 기여 가이드 | YAML 스키마 문서 | Quick Start 가이드 |

**Hill Chart 체크포인트**:
- Week 1 종료: 정상 도달 (CLI로 YAML 파싱 → HTTP 요청 1건 전송 성공)
- Week 3 종료: 내리막 80% (100 VUser 동시 부하 + 메트릭 수집)
- Week 4 종료: 완료 (리포트 생성 + 바이너리 배포)

**산출물**: `curl -sSL install.omnitest.io | sh && omnitest run test.yaml`

**Cooldown**: 린팅 정리, 테스트 커버리지 80%+, Cycle 2 Shaping

---

### Cycle 2: 분산 아키텍처 (4주 Build + 1주 Cooldown)

**Pitch**: "Controller가 다수 Agent를 gRPC로 조율하여 분산 부하를 생성하고, 웹 대시보드에서 실시간 모니터링"

**Appetite**: Big Batch (4주)

**Scope**:
- Controller 서버 (API Server, Agent Manager, Scheduler)
- Agent 모드 분리 (standalone / agent 모드)
- gRPC proto 정의 및 양방향 스트리밍
- Agent 등록/디스커버리/헬스체크
- 기본 웹 대시보드 (React): 테스트 생성/실행, 실시간 차트, 에이전트 상태
- REST API (테스트 CRUD, 실행, 결과 조회)
- PostgreSQL 메타데이터 저장
- Docker Compose 배포

**No-Gos**: K8s 네이티브, CI/CD 통합, 인증/권한, Go 스크립트

**에이전트 태스크 분배**:

| 에이전트 | Week 1 | Week 2-3 | Week 4 |
|---------|--------|----------|--------|
| Core Engine | gRPC proto, Controller 코어 | Agent Manager, Scheduler, 메트릭 집계 | 안정화 |
| API/Integration | REST API 스키마 (OpenAPI) | REST API 구현, Docker Compose | API 테스트 |
| Frontend | React 셋업, 디자인 시스템 | 대시보드 3개 화면, WebSocket 연동 | UI 안정화 |
| Infra/Test | DB 스키마, 마이그레이션 | Agent 3대 통합 테스트 | 부하 검증 |
| Docs/DX | API 문서 (자동 생성) | 분산 테스트 가이드 | 배포 가이드 |

**Hill Chart 체크포인트**:
- Week 1: 정상 도달 (Controller-Agent gRPC 핸드셰이크 성공)
- Week 3: 내리막 80% (Agent 3대 분산 부하 + 웹 대시보드)
- Week 4: 완료 (Docker Compose 풀스택 배포)

**산출물**: `docker-compose up`으로 분산 부하 테스트 + 웹 모니터링

---

### Cycle 3: 클라우드 네이티브 + CI/CD (4주 Build + 1주 Cooldown)

**Pitch**: "K8s에서 Helm으로 5분 내 배포, GitHub Actions에서 PR마다 자동 성능 테스트 + 임계값 pass/fail"

**Appetite**: Big Batch (4주) — Small Batch 2개로 분해

**Small Batch A (Week 1-2)**: K8s 네이티브 + Helm
- K8s Job/Pod 기반 에이전트 자동 생성/삭제 (client-go)
- HPA 기반 에이전트 오토스케일
- Helm Chart (Controller + Agent + 의존성)
- Go 스크립트 지원 (Yaegi 인터프리터)

**Small Batch B (Week 3-4)**: CI/CD + 모니터링
- GitHub Actions Action (`omnitest-action`)
- CLI 확장 (`omnitest ci` 명령)
- 성능 임계값 검증 (thresholds → exit code)
- Prometheus 메트릭 익스포터
- 테스트 스케줄링 (Cron)

**산출물**: K8s 프로덕션 배포 + GitHub Actions 자동 성능 테스트

---

### Cycle 4: 엔터프라이즈 (4주 Build + 1주 Cooldown)

**Pitch**: "인증/RBAC, 고급 리포트, AI 성능 분석 기초를 갖추고 Community/Enterprise 에디션 분리"

**Appetite**: Big Batch (4주) — Small Batch 3개로 분해

**Small Batch A (Week 1-2)**: 인증/RBAC + 멀티 테넌시
**Small Batch B (Week 2-3)**: 고급 리포트 (비교 리포트, PDF)
**Small Batch C (Week 3-4)**: AI 성능 분석 기초 + gRPC 프로토콜 확장

**산출물**: Community + Enterprise Edition 분리 배포

---

### 로드맵 비교

| 항목 | 기존 (Phase 기반) | Shape Up + AI Agent |
|------|-------------------|---------------------|
| **총 기간** | 12개월 | **5개월** |
| 방법론 | Waterfall 유사 | Shape Up (Pitch/Bet/Build) |
| 팀 구성 | 미정 | Comandos (인간 1 + AI 에이전트 5) |
| 사이클 | 3개월/Phase | 4주 Build + 1주 Cooldown |
| 진행 추적 | 마일스톤 | Hill Chart |
| 스코프 관리 | 고정 | Appetite 기반 (시간 고정, 스코프 유동) |
| 병렬성 | 순차적 | 에이전트 4-5개 병렬 |
| 기술 부채 | 별도 계획 없음 | 쿨다운 기간에 체계적 정리 |

### 타임라인 단축 전제 조건

1. **인간 리드의 Shaping 품질**: 각 사이클 전 충분한 Shaping 필수
2. **인터페이스 선행 정의**: Core Engine Agent가 Week 1에 proto/API 스키마 확정
3. **자동화된 통합 테스트**: CI 파이프라인이 견고해야 병렬 머지 시 충돌 조기 발견
4. **에이전트 품질 검증**: Tech Lead 리뷰 시간 충분히 확보 (주당 1일 이상)

---

## 13. 제약 조건 & 가정

### 가정
1. 타겟 사용자는 전체 대상 (사내 QA팀, 개발자, SaaS, 오픈소스+엔터프라이즈)
2. 내부에 Java에 익숙한 엔지니어가 없음
3. 배포 환경은 클라우드 또는 하이브리드
4. 사내 도구로 시작 후 오픈소스로 전향 예정
5. HTTP/HTTPS가 1차 지원 프로토콜
6. AI 에이전트(comandos 팀)를 활용한 개발 진행
7. Shape Up 방법론 적용 (4주 Build + 1주 Cooldown)

### 제약 조건
1. 오픈소스 라이선스: Apache 2.0
2. 외부 의존성 최소화 (단일 바이너리 목표)
3. Go 1.22+ 필수
4. UI는 SPA로 구현 (API 우선 설계)

---

## 14. 레퍼런스

> 별도 문서 참조: [docs/plan/references.md](./references.md)

---

## 변경 이력

| 버전 | 날짜 | 변경 내용 |
|------|------|-----------|
| v1 | 2026-03-17 | 초안 작성 (사용자 인터뷰 기반) |
| v2 | 2026-03-17 | Shape Up + AI Agent(Comandos) 기반 로드맵 재설계, 팀 구조/개발 프로세스 추가 |
