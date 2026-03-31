# OmniTest

Go-based cloud-native distributed performance testing platform.

A modern alternative to nGrinder — single binary, YAML scenarios, real-time metrics, and distributed load testing with a web dashboard.

## Features

- **Single Binary** — No JVM, no dependencies. Just download and run.
- **YAML Scenarios** — Declarative test definitions, Git-friendly.
- **Real-time Metrics** — RPS, P50/P95/P99 latency, error rate in your terminal.
- **Ramp-up** — Gradual load increase with linear ramp-up.
- **Threshold Pass/Fail** — CI/CD native with exit codes.
- **JSON/HTML Reports** — Auto-generated with interactive charts.
- **Distributed Testing** — Controller-Agent architecture with gRPC streaming.
- **Web Dashboard** — React-based real-time monitoring (3 pages).
- **PostgreSQL** — Persistent test results and agent management.
- **Docker Compose** — One-command full-stack deployment.

## Quick Start

### Install

```bash
go install github.com/dan1901/omnitest/cmd/omnitest@latest
```

### Write a Test

```yaml
# test.yaml
version: "1"

targets:
  - name: "my-api"
    base_url: "http://localhost:8080"

scenarios:
  - name: "API Load Test"
    target: "my-api"
    vusers: 100
    duration: "30s"
    ramp_up: "10s"
    requests:
      - method: GET
        path: "/api/users"

thresholds:
  - metric: "http_req_duration_p99"
    condition: "< 200ms"
  - metric: "http_req_failed"
    condition: "< 1%"
```

### Run

```bash
omnitest run test.yaml
```

Output:

```
→ Loading scenario: API Load Test
→ Target: my-api (http://localhost:8080)
→ VUsers: 100, Duration: 30s, Ramp-up: 10s

  Elapsed   VUsers   RPS      Avg      P50      P95      P99      Errors
  ─────────────────────────────────────────────────────────────────────────
  00:15     100/100  1,234    45ms     38ms     120ms    250ms    0.1%

  [████████████████░░░░░░░░░░░░░░]  50% | 00:15 remaining

✓ Test completed.
✓ PASS: http_req_duration_p99 (180ms) < 200ms
✓ PASS: http_req_failed (0.1%) < 1%
```

## CLI Commands

```bash
omnitest run <file.yaml>                        # Run load test
omnitest run <file.yaml> --vusers 50 --duration 2m  # Override parameters
omnitest run <file.yaml> --out json,html        # Generate reports
omnitest run <file.yaml> --ramp-up 10s          # Gradual ramp-up
omnitest run <file.yaml> --quiet                # Minimal output (CI)
omnitest validate <file.yaml>                   # Validate YAML only
omnitest version                                # Show version
```

### Distributed Mode

```bash
# Start Controller
omnitest controller --grpc-port=9090 --http-port=8080 --db-url=postgres://...

# Connect Agents
omnitest agent --controller=localhost:9090 --name=agent-1 --max-vusers=5000
omnitest agent --controller=localhost:9090 --name=agent-2 --max-vusers=5000
```

### Docker Compose

```bash
docker-compose up
# Controller + 3 Agents + PostgreSQL + Web Dashboard
```

## YAML Schema

```yaml
version: "1"

targets:
  - name: "api-server"
    base_url: "https://api.example.com"
    headers:                              # Optional: default headers
      Authorization: "Bearer ${API_TOKEN}"  # Environment variable substitution
      Content-Type: "application/json"

scenarios:
  - name: "Scenario Name"
    target: "api-server"                  # Reference to target
    vusers: 100                           # Number of virtual users
    duration: "30s"                       # Test duration
    ramp_up: "10s"                        # Optional: gradual ramp-up
    requests:
      - method: GET                       # GET, POST, PUT, DELETE
        path: "/api/endpoint"
        headers:                          # Optional: per-request headers
          X-Custom: "value"
        body:                             # Optional: JSON body (POST/PUT)
          key: "value"

thresholds:                               # Optional: pass/fail criteria
  - metric: "http_req_duration_p99"       # p50, p95, p99, avg
    condition: "< 200ms"
  - metric: "http_req_failed"             # Error rate
    condition: "< 1%"
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All thresholds passed |
| 1 | One or more thresholds failed |
| 2 | Configuration error |
| 3 | Runtime error |
| 99 | Report generation error |

## Architecture

```
                         +------------------+
                         |   Web Dashboard  |
                         |   (React + TS)   |
                         +--------+---------+
                                  |
                                  | REST / WebSocket
                                  v
+----------+           +-------------------+           +-----------------+
|   CLI    +---------->+   Controller      +<--------->+ PostgreSQL      |
| omnitest |   gRPC    |                   |           |                 |
+----------+           |  - API Server     |           +-----------------+
                        |  - Agent Manager  |
                        |  - Scheduler      |
                        |  - Aggregator     |
                        +---+-----+----+---+
                            |     |    |
                     gRPC   |     |    |   gRPC
               +------------+     |    +----------+
               v                  v               v
        +-----------+     +-----------+    +-----------+
        |  Agent 1  |     |  Agent 2  |    |  Agent N  |
        | [Workers] |     | [Workers] |    | [Workers] |
        | [Metrics] |     | [Metrics] |    | [Metrics] |
        +-----------+     +-----------+    +-----------+
```

## Examples

A demo server and 11 scenario files are included for testing all features:

```bash
# Start demo server
go run examples/demo-server/main.go

# Run scenarios
omnitest run examples/scenarios/01-quick-start.yaml
omnitest run examples/scenarios/02-ramp-up.yaml
omnitest run examples/scenarios/05-thresholds.yaml
omnitest run examples/scenarios/10-report-demo.yaml --out json,html
```

See [examples/README.md](examples/README.md) for the full list.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Core Engine | Go 1.22+ |
| Communication | gRPC + Protocol Buffers |
| Web Dashboard | React + TypeScript + Vite + Recharts |
| Database | PostgreSQL (pgx/v5) |
| Metrics | HDR Histogram |
| CLI | Cobra |
| Container | Docker + Docker Compose |

## Key Characteristics

| | OmniTest |
|---|----------|
| **Install** | Single binary, no dependencies (5min) |
| **Memory** | ~100-200MB per agent |
| **VUsers/agent** | 10,000+ (goroutine-based) |
| **Scenarios** | YAML declarative + Go scripting (planned) |
| **CI/CD** | CLI native + threshold exit codes |
| **Monitoring** | WebSocket real-time streaming |
| **Deployment** | Docker Compose / K8s (planned) |
| **Reports** | JSON + HTML with interactive charts |

## Roadmap

- [x] Cycle 1: MVP Core Engine (CLI, YAML, metrics, reports)
- [x] Cycle 2: Distributed Architecture (gRPC, REST API, Web Dashboard, PostgreSQL)
- [ ] Cycle 3: Cloud Native (K8s, Helm, CI/CD, Prometheus)
- [ ] Cycle 4: Enterprise (RBAC, AI analysis, protocol extensions)

## License

This project is licensed under the [GNU Affero General Public License v3.0](LICENSE).
