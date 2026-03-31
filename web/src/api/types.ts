export interface TestDefinition {
  id: string;
  name: string;
  scenario_yaml: string;
  created_at: string;
  updated_at: string;
}

export interface TestRun {
  id: string;
  test_id: string;
  status: string;
  total_vusers: number;
  duration_seconds: number;
  started_at?: string;
  finished_at?: string;
  created_at: string;
}

export interface AgentInfo {
  agent_id: string;
  hostname: string;
  max_vusers: number;
  status: string;
  active_vusers: number;
  cpu_usage: number;
  memory_usage: number;
  last_heartbeat: string;
}

export interface AggregatedMetrics {
  test_run_id: string;
  timestamp: string;
  total_rps: number;
  avg_latency_ms: number;
  p50_latency_ms: number;
  p95_latency_ms: number;
  p99_latency_ms: number;
  total_requests: number;
  total_errors: number;
  active_vusers: number;
}

export interface ApiEnvelope<T> {
  data: T;
  error: string | null;
  meta?: Record<string, unknown>;
}
