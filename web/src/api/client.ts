import type { ApiEnvelope, TestDefinition, TestRun, AgentInfo } from './types.ts';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API error ${res.status}: ${text}`);
  }
  const envelope: ApiEnvelope<T> = await res.json();
  if (envelope.error) {
    throw new Error(envelope.error);
  }
  return envelope.data;
}

export async function fetchTests(): Promise<TestDefinition[]> {
  return request<TestDefinition[]>('/api/v1/tests');
}

export async function createTest(name: string, scenarioYaml: string): Promise<TestDefinition> {
  return request<TestDefinition>('/api/v1/tests', {
    method: 'POST',
    body: JSON.stringify({ name, scenario_yaml: scenarioYaml }),
  });
}

export async function runTest(testId: string, totalVusers: number, durationSeconds: number): Promise<TestRun> {
  return request<TestRun>(`/api/v1/tests/${testId}/run`, {
    method: 'POST',
    body: JSON.stringify({ total_vusers: totalVusers, duration_seconds: durationSeconds }),
  });
}

export async function stopTest(runId: string): Promise<void> {
  return request<void>(`/api/v1/runs/${runId}/stop`, {
    method: 'POST',
  });
}

export async function fetchAgents(): Promise<AgentInfo[]> {
  return request<AgentInfo[]>('/api/v1/agents');
}

export async function fetchTestRun(runId: string): Promise<TestRun> {
  return request<TestRun>(`/api/v1/runs/${runId}`);
}

export async function healthCheck(): Promise<{ status: string }> {
  return request<{ status: string }>('/api/v1/health');
}
