# OmniTest Examples

## 빠른 시작

### 1. 데모 서버 실행

```bash
go run examples/demo-server/main.go
```

서버가 `http://localhost:8888` 에서 시작됩니다.

### 2. 빌드

```bash
make build
# 또는
go build -o bin/omnitest ./cmd/omnitest
```

### 3. 테스트 실행

```bash
# 가장 기본적인 테스트
bin/omnitest run examples/scenarios/01-quick-start.yaml
```

---

## 시나리오 목록

| # | 파일 | 데모 기능 | 난이도 |
|---|------|----------|--------|
| 01 | `01-quick-start.yaml` | 기본 GET 부하 테스트 | 입문 |
| 02 | `02-ramp-up.yaml` | 점진적 부하 증가 (Ramp-Up) | 입문 |
| 03 | `03-multi-endpoint.yaml` | 다중 타겟 + 다중 시나리오 | 초급 |
| 04 | `04-post-with-body.yaml` | POST + JSON Body | 초급 |
| 05 | `05-thresholds.yaml` | 임계값 Pass/Fail (CI/CD용) | 중급 |
| 06 | `06-headers-auth.yaml` | 커스텀 헤더 + 환경변수 토큰 | 중급 |
| 07 | `07-stress-test.yaml` | 스트레스 테스트 (고부하) | 중급 |
| 08 | `08-error-rate.yaml` | 에러율 모니터링 + Threshold | 중급 |
| 09 | `09-slow-endpoint.yaml` | 느린 응답 분석 | 초급 |
| 10 | `10-report-demo.yaml` | JSON + HTML 리포트 생성 | 초급 |
| 11 | `11-crud-workflow.yaml` | CRUD 전체 메서드 (GET/POST/PUT/DELETE) | 초급 |

---

## 실행 예시

### 기본 실행
```bash
bin/omnitest run examples/scenarios/01-quick-start.yaml
```

### Ramp-Up 테스트
```bash
bin/omnitest run examples/scenarios/02-ramp-up.yaml
```

### 파라미터 오버라이드
```bash
bin/omnitest run examples/scenarios/01-quick-start.yaml \
  --vusers 50 \
  --duration 1m \
  --ramp-up 10s
```

### 리포트 생성
```bash
bin/omnitest run examples/scenarios/10-report-demo.yaml \
  --out json,html \
  --out-dir ./reports

# HTML 리포트 열기
open ./reports/report-*.html
```

### 환경변수 토큰 주입
```bash
API_TOKEN=my-secret-token \
  bin/omnitest run examples/scenarios/06-headers-auth.yaml
```

### CI/CD 파이프라인 (exit code 활용)
```bash
bin/omnitest run examples/scenarios/05-thresholds.yaml
if [ $? -ne 0 ]; then
  echo "Performance test FAILED!"
  exit 1
fi
```

### YAML 검증만 (실행 없이)
```bash
bin/omnitest validate examples/scenarios/07-stress-test.yaml
```

### Quiet 모드 (CI용, 최소 출력)
```bash
bin/omnitest run examples/scenarios/01-quick-start.yaml --quiet
```

---

## 데모 서버 엔드포인트

| 엔드포인트 | 응답 시간 | 에러율 | 용도 |
|-----------|----------|--------|------|
| `GET /health` | 즉시 | 0% | 헬스체크 |
| `GET /api/users` | 20-50ms | 0% | 기본 GET |
| `POST /api/users` | 30-80ms | 0% | POST + Body |
| `GET /api/products` | 50-150ms | 0% | 중간 지연 |
| `GET /api/search?q=` | 100-300ms | 0% | 느린 응답 |
| `POST /api/orders` | 50-100ms | **5%** | 에러 발생 |
| `GET /api/slow` | 500-2000ms | 0% | 매우 느린 응답 |
| `GET /api/flaky` | 20-50ms | **30%** | 불안정 API |
| `GET /api/echo` | 10ms | 인증 필요 | 헤더 검증 |
| `GET /api/variable-load` | 가변 | 0% | 부하 비례 지연 |
| `PUT /api/users/{id}` | 30-70ms | 0% | PUT 메서드 |
| `DELETE /api/users/{id}` | 20ms | 조건부 | DELETE 메서드 |
| `GET /api/stats` | 즉시 | 0% | 서버 통계 |
