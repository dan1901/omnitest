# Pitch: Cycle 1 - MVP Core Engine

> **Quick Gate**: `omnitest run test.yaml` 한 줄로 로컬 HTTP 부하 테스트를 실행하고, 터미널에서 실시간 메트릭을 확인하며, JSON/HTML 리포트를 자동 생성하는 단일 바이너리 CLI 도구

---

## 1. Problem

### 기존 도구의 한계

**nGrinder**: Java/Tomcat/DB 설치에 30분 이상 소요. JVM 기반 에이전트는 에이전트당 1-2GB 메모리를 소비하며, GC pause로 부하 생성이 불안정하다. Groovy/Jython 학습 곡선이 높고, CLI 도구가 없어 CI/CD 통합이 번거롭다. 커뮤니티 활동도 둔화되고 있다.

**k6**: DX는 우수하나 분산 테스트가 유료(k6 Cloud). JavaScript 런타임 의존으로 복잡한 시나리오에서 Go 대비 성능 한계가 있다.

**JMeter**: XML 기반 스크립트로 Git 관리가 어렵고, UI가 구식이며 리소스 소비가 크다.

**Gatling**: Scala DSL 학습 곡선이 높고, 엔터프라이즈 기능은 유료다.

**공통 문제**: "5분 안에 첫 테스트를 실행"할 수 있는, YAML 선언적 시나리오 + CLI 네이티브 + 단일 바이너리 도구가 없다.

---

## 2. Appetite

**Big Batch** - 이번 세션에서 MVP 완성

Cycle 1 전체(4주 Build)를 투자하여 로컬 단독 실행이 가능한 핵심 엔진을 완성한다. 분산 테스트, 웹 UI, 데이터베이스 등은 Cycle 2 이후로 미룬다.

---

## 3. Solution (Fat Marker Sketch)

### 3.1 CLI 사용 플로우

```
# 1. 설치 (단일 바이너리)
$ curl -sSL install.omnitest.io | sh
  # 또는: brew install omnitest / go install github.com/omnitest/omnitest@latest

# 2. 테스트 시나리오 작성
$ vi load-test.yaml

# 3. 테스트 실행
$ omnitest run load-test.yaml

# 4. 실시간 터미널 출력 (자동)
  Running "기본 부하 테스트" with 100 VUsers for 5m0s...

  Elapsed   VUsers   RPS      Avg      P50      P95      P99      Errors
  ─────────────────────────────────────────────────────────────────────────
  00:30     100      1,234    45ms     38ms     120ms    250ms    0.1%
  01:00     100      1,312    42ms     35ms     110ms    230ms    0.0%
  ...

  [████████████████████░░░░░░░░░░] 70% | 3m30s / 5m00s

# 5. 완료 시 리포트 자동 생성
  ✓ Test completed.
  ✓ Report saved: ./reports/report-20260317-143022.html
  ✓ JSON data: ./reports/report-20260317-143022.json

# 6. 임계값 기반 Pass/Fail (CI/CD용)
  ✓ PASS: http_req_duration_p99 (180ms) < 200ms
  ✗ FAIL: http_req_failed (2.1%) > 1%
  Exit code: 1
```

### 3.2 핵심 CLI 명령어

| 명령어 | 설명 |
|--------|------|
| `omnitest run <file.yaml>` | 테스트 실행 (핵심 명령) |
| `omnitest run <file.yaml> --vusers 50 --duration 2m` | 인라인 파라미터 오버라이드 |
| `omnitest run <file.yaml> --out json --out html` | 출력 형식 지정 |
| `omnitest version` | 버전 정보 출력 |
| `omnitest validate <file.yaml>` | YAML 시나리오 검증 (실행 없이) |

### 3.3 YAML 스키마 초안

```yaml
# load-test.yaml
version: "1"

targets:
  - name: "user-api"
    base_url: "https://api.example.com"
    headers:
      Authorization: "Bearer ${TOKEN}"
      Content-Type: "application/json"

scenarios:
  - name: "기본 부하 테스트"
    target: "user-api"
    vusers: 100
    duration: "5m"
    ramp_up: "30s"
    requests:
      - method: GET
        path: "/users"
      - method: POST
        path: "/users"
        body:
          name: "test-user-${__counter}"
          email: "test${__counter}@example.com"
      - method: GET
        path: "/users/${__response.id}"
      - method: PUT
        path: "/users/${__response.id}"
        body:
          name: "updated-user"
      - method: DELETE
        path: "/users/${__response.id}"

thresholds:
  - metric: "http_req_duration_p99"
    condition: "< 200ms"
  - metric: "http_req_failed"
    condition: "< 1%"
  - metric: "http_reqs"
    condition: "> 1000"
```

**YAML 스키마 핵심 필드**:
- `version`: 스키마 버전 (하위 호환성 관리)
- `targets`: 테스트 대상 서버 정의 (base_url, 공통 헤더)
- `scenarios`: 부하 시나리오 (VUser 수, 지속시간, Ramp-up, 요청 목록)
- `thresholds`: Pass/Fail 판정 기준 (메트릭 + 조건)
- 환경변수 참조: `${ENV_VAR}` 문법 지원

### 3.4 메트릭 출력 형태

**실시간 터미널 출력** (1초 간격 갱신):
```
omnitest v0.1.0 | 기본 부하 테스트

  Status:    running
  VUsers:    100 / 100
  Duration:  02:30 / 05:00

  ┌─────────────────────────────────────────────────┐
  │ Metric          Current    Avg       Max        │
  ├─────────────────────────────────────────────────┤
  │ RPS             1,284      1,256     1,412      │
  │ Latency Avg     42ms       45ms      --         │
  │ Latency P50     35ms       38ms      --         │
  │ Latency P95     110ms      118ms     --         │
  │ Latency P99     230ms      245ms     --         │
  │ Error Rate      0.1%       0.08%     --         │
  │ Bytes In        12.4 MB/s  --        --         │
  └─────────────────────────────────────────────────┘

  [████████████████░░░░░░░░░░░░░░] 50% | 02:30 remaining
```

**최종 요약** (테스트 완료 시):
```
──────────────────── Test Summary ────────────────────

  Total Requests:   375,840
  Total Duration:   5m0s
  Success Rate:     99.92%

  Latency Distribution:
    P50:   38ms
    P95:   118ms
    P99:   245ms
    Max:   1,230ms

  RPS:
    Avg:   1,253
    Max:   1,412

  Thresholds:
    ✓ http_req_duration_p99 (245ms) ... WARN (< 200ms FAILED)
    ✓ http_req_failed (0.08%) ........ PASS (< 1%)

──────────────────────────────────────────────────────
```

---

## 4. Rabbit Holes

| 위험 요소 | 사전 결정 | 근거 |
|-----------|-----------|------|
| HDR Histogram 라이브러리 선택 | `github.com/HdrHistogram/hdrhistogram-go` 확정 | k6, Vegeta에서 검증됨. 정확한 백분위 계산 보장 |
| HTTP 클라이언트 (`net/http` vs `fasthttp`) | `net/http` 표준 라이브러리 사용 | HTTP/2 지원, 안정성 우선. fasthttp는 HTTP/2 미지원. MVP에서 충분한 성능 |
| YAML 파서 | `gopkg.in/yaml.v3` 사용 | Go 표준에 가까운 안정된 라이브러리 |
| CLI 프레임워크 | `spf13/cobra` 확정 | Go CLI의 de facto 표준. 자동 완성, 서브커맨드 지원 |
| TUI 실시간 출력 방식 | 텍스트 기반 프로그레스 (Bubble Tea 또는 단순 ANSI) | MVP에서는 과도한 TUI 복잡도 지양. 최소한 ANSI 제어로 터미널 갱신 |
| 환경변수 치환 | `${VAR}` 문법, `os.Getenv` 기반 단순 구현 | 템플릿 엔진(Go template 등) 도입은 과도. 단순 문자열 치환으로 시작 |
| Ramp-up 알고리즘 | 선형 Ramp-up만 지원 | Step, Exponential 등은 Cycle 2 이후 |
| HTML 리포트 템플릿 | Go `html/template` + 인라인 CSS/JS | 외부 의존성 없이 단일 HTML 파일 생성. 차트는 인라인 SVG 또는 go-echarts |
| Coordinated Omission | MVP에서는 미대응 | Gil Tene의 CO 문제는 인지하되, MVP 단계에서는 단순 측정. 향후 보정 옵션 추가 |

---

## 5. No-Gos

이번 Cycle 1에서 **명시적으로 하지 않을 것**:

| 항목 | 이유 |
|------|------|
| **분산 테스트 (Controller-Agent)** | Cycle 2 스코프. MVP는 단일 프로세스 로컬 실행에 집중 |
| **웹 UI / 대시보드** | Cycle 2 스코프. 터미널 출력으로 충분 |
| **Go 스크립트 (Yaegi)** | Cycle 3 스코프. YAML만으로 MVP 시나리오 충분히 커버 |
| **데이터베이스 연동** | 분산 테스트 시 필요. MVP는 파일 기반 리포트로 충분 |
| **gRPC/WebSocket/GraphQL 프로토콜** | HTTP/HTTPS만 1차 지원. 프로토콜 확장은 이후 |
| **인증/RBAC** | 로컬 CLI 도구에 불필요 |
| **테스트 스케줄링 (Cron)** | 서버 컴포넌트 없이 불가능 |
| **Prometheus 메트릭 익스포터** | 서버 모드 없이 불필요 |
| **과도한 TUI (full Bubble Tea)** | 단순 텍스트 갱신으로 시작. 복잡한 TUI는 사용자 피드백 후 결정 |

---

## 6. UX

### 6.1 CLI 톤앤매너

**원칙**: 간결하고 전문적인 개발자 도구. 불필요한 장식 없이 핵심 정보만 전달.

- **진행 상태**: 깔끔한 테이블 형태 + 프로그레스 바
- **성공**: `✓` 접두사, 간결한 메시지 (`✓ Test completed.`)
- **경고**: `⚠` 접두사, 노란색 (ANSI) (`⚠ WARN: p99 exceeded threshold`)
- **실패**: `✗` 접두사, 빨간색 (ANSI) (`✗ FAIL: error rate 2.1% > 1%`)
- **정보**: 접두사 없음 또는 `→` (`→ Loading scenario: load-test.yaml`)
- **컬러**: `--no-color` 플래그로 비활성화 가능 (CI 환경 호환)
- **Quiet 모드**: `--quiet` 플래그로 최종 요약만 출력
- **Verbose 모드**: `--verbose` 플래그로 디버그 정보 출력

**참고 모델**: k6의 CLI 출력 스타일 (깔끔, 구조화, 색상 절제)

### 6.2 에러 메시지 체계

에러 메시지는 3단 구조를 따른다: **What → Why → How**

```
✗ Error: failed to parse scenario file
  → load-test.yaml:12: unknown field "vuser" in scenarios
  → Did you mean "vusers"? (hint: check YAML field names)
```

```
✗ Error: connection refused
  → GET https://api.example.com/users
  → Target server is not reachable. Check if the server is running
    and the URL is correct.
```

```
✗ Error: invalid threshold condition
  → "http_req_duration_p99 < 200"
  → Duration values must include a unit. Use "< 200ms" instead.
```

| 에러 유형 | Exit Code |
|-----------|-----------|
| 성공 (모든 threshold 통과) | 0 |
| Threshold 실패 | 1 |
| 시나리오 파일 오류 | 2 |
| 연결/네트워크 오류 | 3 |
| 내부 오류 | 99 |

### 6.3 핵심 CLI 명령어/플래그 정의

```
omnitest - Cloud-native performance testing tool

Usage:
  omnitest [command]

Available Commands:
  run         Run a load test from a YAML scenario file
  validate    Validate a YAML scenario file without running
  version     Print version information
  help        Help about any command

Flags:
  -h, --help       Help for omnitest
  --no-color       Disable colored output
  --verbose        Enable verbose/debug output

Run Flags:
  --vusers int       Override virtual users count
  --duration string  Override test duration (e.g., "5m", "1h")
  --ramp-up string   Override ramp-up period
  --out string       Output format: json, html (repeatable)
  --out-dir string   Output directory for reports (default: ./reports)
  --quiet            Show only final summary
  --env KEY=VALUE    Set environment variable (repeatable)
```

---

## 7. Acceptance Criteria (수용 기준)

1. **AC-1**: `omnitest run test.yaml` 명령으로 YAML에 정의된 HTTP 요청(GET/POST/PUT/DELETE)을 지정된 VUser 수와 Duration으로 동시 실행할 수 있다.

2. **AC-2**: 테스트 실행 중 터미널에 RPS, Latency (P50/P95/P99), Error Rate 메트릭이 1초 간격으로 실시간 갱신되어 출력된다.

3. **AC-3**: 테스트 완료 시 JSON 리포트와 HTML 리포트가 자동 생성되며, 결과 요약이 터미널에 출력된다.

4. **AC-4**: `thresholds` 설정에 따라 테스트 Pass/Fail이 exit code(0/1)로 반환되어, CI/CD 파이프라인에서 자동 판정에 사용할 수 있다.

5. **AC-5**: GoReleaser로 빌드된 단일 바이너리가 macOS(arm64/amd64), Linux(arm64/amd64)에서 추가 의존성 없이 실행된다.

---

## 8. Quick Gate 1줄 요약

> `omnitest run test.yaml` 한 줄로 YAML 기반 HTTP 부하 테스트를 실행하고, 터미널 실시간 메트릭 + JSON/HTML 리포트 + threshold exit code를 제공하는 Go 단일 바이너리 CLI 도구
