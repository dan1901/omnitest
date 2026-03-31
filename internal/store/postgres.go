// Package store implements PostgreSQL data storage for OmniTest.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omnitest/omnitest/pkg/model"
)

// Store provides database operations.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a new Store with a connection pool from a database URL.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}
	poolConfig.MaxConns = 20
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Store{pool: pool}, nil
}

// Close closes the connection pool.
func (s *Store) Close() {
	s.pool.Close()
}

// --- Agent CRUD ---

// CreateAgent inserts a new agent record.
func (s *Store) CreateAgent(ctx context.Context, agent *model.AgentInfo) error {
	labelsJSON, _ := json.Marshal(agent.Labels)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO agents (agent_id, hostname, max_vusers, status, labels, registered_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)
		 ON CONFLICT (agent_id) DO UPDATE SET
		   hostname = EXCLUDED.hostname,
		   max_vusers = EXCLUDED.max_vusers,
		   status = EXCLUDED.status,
		   labels = EXCLUDED.labels,
		   updated_at = NOW()`,
		agent.AgentID, agent.Hostname, agent.MaxVUsers, string(agent.Status), labelsJSON, agent.RegisteredAt,
	)
	return err
}

// UpdateAgentHeartbeat updates the agent's heartbeat and status info.
func (s *Store) UpdateAgentHeartbeat(ctx context.Context, agentID string, status model.AgentStatus, cpuUsage, memoryUsage float64, activeVUsers int) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE agents SET status = $2, cpu_usage = $3, memory_usage = $4, last_heartbeat = NOW(), updated_at = NOW()
		 WHERE agent_id = $1`,
		agentID, string(status), cpuUsage, memoryUsage,
	)
	return err
}

// UpdateAgentStatus updates only the agent's status.
func (s *Store) UpdateAgentStatus(ctx context.Context, agentID string, status model.AgentStatus) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE agents SET status = $2, updated_at = NOW() WHERE agent_id = $1`,
		agentID, string(status),
	)
	return err
}

// ListAgents returns all agents.
func (s *Store) ListAgents(ctx context.Context) ([]model.AgentInfo, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT agent_id, hostname, max_vusers, status, labels, cpu_usage, memory_usage, last_heartbeat, registered_at
		 FROM agents ORDER BY registered_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []model.AgentInfo
	for rows.Next() {
		var a model.AgentInfo
		var labelsJSON []byte
		var lastHB *time.Time
		if err := rows.Scan(&a.AgentID, &a.Hostname, &a.MaxVUsers, &a.Status, &labelsJSON, &a.CPUUsage, &a.MemoryUsage, &lastHB, &a.RegisteredAt); err != nil {
			return nil, err
		}
		if labelsJSON != nil {
			json.Unmarshal(labelsJSON, &a.Labels)
		}
		if lastHB != nil {
			a.LastHeartbeat = *lastHB
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// GetAgent returns a single agent by agent_id.
func (s *Store) GetAgent(ctx context.Context, agentID string) (*model.AgentInfo, error) {
	var a model.AgentInfo
	var labelsJSON []byte
	var lastHB *time.Time
	err := s.pool.QueryRow(ctx,
		`SELECT agent_id, hostname, max_vusers, status, labels, cpu_usage, memory_usage, last_heartbeat, registered_at
		 FROM agents WHERE agent_id = $1`, agentID,
	).Scan(&a.AgentID, &a.Hostname, &a.MaxVUsers, &a.Status, &labelsJSON, &a.CPUUsage, &a.MemoryUsage, &lastHB, &a.RegisteredAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if labelsJSON != nil {
		json.Unmarshal(labelsJSON, &a.Labels)
	}
	if lastHB != nil {
		a.LastHeartbeat = *lastHB
	}
	return &a, nil
}

// DeleteAgent deletes an agent record.
func (s *Store) DeleteAgent(ctx context.Context, agentID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM agents WHERE agent_id = $1`, agentID)
	return err
}

// --- Test CRUD ---

// CreateTest inserts a new test definition.
func (s *Store) CreateTest(ctx context.Context, test *model.TestDefinition) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO tests (name, scenario_yaml, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 RETURNING id, created_at, updated_at`,
		test.Name, test.ScenarioYAML, test.CreatedBy,
	).Scan(&test.ID, &test.CreatedAt, &test.UpdatedAt)
	return err
}

// GetTest returns a test definition by ID.
func (s *Store) GetTest(ctx context.Context, id string) (*model.TestDefinition, error) {
	var t model.TestDefinition
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, scenario_yaml, COALESCE(created_by, ''), created_at, updated_at
		 FROM tests WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.ScenarioYAML, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ListTests returns tests with pagination.
func (s *Store) ListTests(ctx context.Context, page, perPage int) ([]model.TestDefinition, int, error) {
	var total int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tests`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, scenario_yaml, COALESCE(created_by, ''), created_at, updated_at
		 FROM tests ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		perPage, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tests []model.TestDefinition
	for rows.Next() {
		var t model.TestDefinition
		if err := rows.Scan(&t.ID, &t.Name, &t.ScenarioYAML, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		tests = append(tests, t)
	}
	return tests, total, nil
}

// UpdateTest updates a test definition.
func (s *Store) UpdateTest(ctx context.Context, id, name, scenarioYAML string) (*model.TestDefinition, error) {
	var t model.TestDefinition
	err := s.pool.QueryRow(ctx,
		`UPDATE tests SET name = $2, scenario_yaml = $3, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, name, scenario_yaml, COALESCE(created_by, ''), created_at, updated_at`,
		id, name, scenarioYAML,
	).Scan(&t.ID, &t.Name, &t.ScenarioYAML, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// DeleteTest deletes a test definition.
func (s *Store) DeleteTest(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM tests WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- TestRun CRUD ---

// CreateTestRun inserts a new test run record.
func (s *Store) CreateTestRun(ctx context.Context, run *model.TestRun) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO test_runs (test_id, status, total_vusers, duration_seconds, created_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 RETURNING id, created_at`,
		run.TestID, string(run.Status), run.TotalVUsers, run.DurationSeconds,
	).Scan(&run.ID, &run.CreatedAt)
	return err
}

// GetTestRun returns a test run by ID.
func (s *Store) GetTestRun(ctx context.Context, id string) (*model.TestRun, error) {
	var r model.TestRun
	var resultJSON, thresholdJSON []byte
	err := s.pool.QueryRow(ctx,
		`SELECT id, test_id, status, total_vusers, duration_seconds, started_at, finished_at,
		        result_summary, threshold_results, created_at
		 FROM test_runs WHERE id = $1`, id,
	).Scan(&r.ID, &r.TestID, &r.Status, &r.TotalVUsers, &r.DurationSeconds,
		&r.StartedAt, &r.FinishedAt, &resultJSON, &thresholdJSON, &r.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if resultJSON != nil {
		var result model.TestResult
		json.Unmarshal(resultJSON, &result)
		r.ResultSummary = &result
	}
	if thresholdJSON != nil {
		json.Unmarshal(thresholdJSON, &r.ThresholdResults)
	}
	return &r, nil
}

// ListTestRuns returns test runs for a given test ID.
func (s *Store) ListTestRuns(ctx context.Context, testID string, page, perPage int) ([]model.TestRun, int, error) {
	var total int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM test_runs WHERE test_id = $1`, testID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows, err := s.pool.Query(ctx,
		`SELECT id, test_id, status, total_vusers, duration_seconds, started_at, finished_at,
		        result_summary, threshold_results, created_at
		 FROM test_runs WHERE test_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		testID, perPage, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var runs []model.TestRun
	for rows.Next() {
		var r model.TestRun
		var resultJSON, thresholdJSON []byte
		if err := rows.Scan(&r.ID, &r.TestID, &r.Status, &r.TotalVUsers, &r.DurationSeconds,
			&r.StartedAt, &r.FinishedAt, &resultJSON, &thresholdJSON, &r.CreatedAt); err != nil {
			return nil, 0, err
		}
		if resultJSON != nil {
			var result model.TestResult
			json.Unmarshal(resultJSON, &result)
			r.ResultSummary = &result
		}
		if thresholdJSON != nil {
			json.Unmarshal(thresholdJSON, &r.ThresholdResults)
		}
		runs = append(runs, r)
	}
	return runs, total, nil
}

// UpdateTestRunStatus updates the status and optionally started_at/finished_at.
func (s *Store) UpdateTestRunStatus(ctx context.Context, id string, status model.TestRunStatus) error {
	switch status {
	case model.TestRunRunning:
		_, err := s.pool.Exec(ctx,
			`UPDATE test_runs SET status = $2, started_at = NOW() WHERE id = $1`,
			id, string(status))
		return err
	case model.TestRunCompleted, model.TestRunFailed, model.TestRunStopped:
		_, err := s.pool.Exec(ctx,
			`UPDATE test_runs SET status = $2, finished_at = NOW() WHERE id = $1`,
			id, string(status))
		return err
	default:
		_, err := s.pool.Exec(ctx,
			`UPDATE test_runs SET status = $2 WHERE id = $1`,
			id, string(status))
		return err
	}
}

// UpdateTestRunResult updates the final result summary.
func (s *Store) UpdateTestRunResult(ctx context.Context, id string, result *model.TestResult, thresholds []model.ThresholdResult) error {
	resultJSON, _ := json.Marshal(result)
	thresholdJSON, _ := json.Marshal(thresholds)
	_, err := s.pool.Exec(ctx,
		`UPDATE test_runs SET result_summary = $2, threshold_results = $3, finished_at = NOW(), status = 'completed'
		 WHERE id = $1`,
		id, resultJSON, thresholdJSON)
	return err
}

// GetRunningTestRun returns the running test run for a given test ID, if any.
func (s *Store) GetRunningTestRun(ctx context.Context, testID string) (*model.TestRun, error) {
	var r model.TestRun
	err := s.pool.QueryRow(ctx,
		`SELECT id, test_id, status, total_vusers, duration_seconds, started_at, finished_at, created_at
		 FROM test_runs WHERE test_id = $1 AND status IN ('pending', 'running') LIMIT 1`, testID,
	).Scan(&r.ID, &r.TestID, &r.Status, &r.TotalVUsers, &r.DurationSeconds, &r.StartedAt, &r.FinishedAt, &r.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// --- Metrics ---

// InsertMetric inserts a metric report from an agent.
func (s *Store) InsertMetric(ctx context.Context, testRunID, agentID string, ts time.Time,
	totalReqs, totalErrors int64, rps, avgLatency, p50, p95, p99 float64, activeVUsers int) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO metrics (test_run_id, agent_id, timestamp, total_requests, total_errors,
		  rps, avg_latency_ms, p50_latency_ms, p95_latency_ms, p99_latency_ms, active_vusers)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		testRunID, agentID, ts, totalReqs, totalErrors, rps, avgLatency, p50, p95, p99, activeVUsers,
	)
	return err
}

// ListMetrics returns time-series metrics for a test run.
func (s *Store) ListMetrics(ctx context.Context, testRunID string) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT agent_id, timestamp, total_requests, total_errors, rps,
		        avg_latency_ms, p50_latency_ms, p95_latency_ms, p99_latency_ms, active_vusers
		 FROM metrics WHERE test_run_id = $1 ORDER BY timestamp ASC`, testRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []map[string]interface{}
	for rows.Next() {
		var agentID string
		var ts time.Time
		var totalReqs, totalErrors int64
		var rps, avgLat, p50, p95, p99 float64
		var activeVU int
		if err := rows.Scan(&agentID, &ts, &totalReqs, &totalErrors, &rps, &avgLat, &p50, &p95, &p99, &activeVU); err != nil {
			return nil, err
		}
		metrics = append(metrics, map[string]interface{}{
			"agent_id":       agentID,
			"timestamp":      ts,
			"total_requests": totalReqs,
			"total_errors":   totalErrors,
			"rps":            rps,
			"avg_latency_ms": avgLat,
			"p50_latency_ms": p50,
			"p95_latency_ms": p95,
			"p99_latency_ms": p99,
			"active_vusers":  activeVU,
		})
	}
	return metrics, nil
}
