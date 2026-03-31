import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { fetchTestRun, stopTest } from '../api/client.ts';
import { useWebSocket } from '../hooks/useWebSocket.ts';
import type { TestRun, AggregatedMetrics } from '../api/types.ts';
import MetricsChart from '../components/MetricsChart.tsx';
import StatusBadge from '../components/StatusBadge.tsx';

export default function TestRunPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [run, setRun] = useState<TestRun | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { data: metrics, connected } = useWebSocket<AggregatedMetrics>(id);

  useEffect(() => {
    if (!id) return;
    fetchTestRun(id)
      .then(setRun)
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load run'));
  }, [id]);

  // Poll run status while running
  useEffect(() => {
    if (!id || !run || (run.status !== 'running' && run.status !== 'pending')) return;
    const interval = setInterval(() => {
      fetchTestRun(id).then(setRun).catch(() => { /* ignore polling errors */ });
    }, 5000);
    return () => clearInterval(interval);
  }, [id, run?.status]);

  async function handleStop() {
    if (!id) return;
    try {
      await stopTest(id);
      if (run) setRun({ ...run, status: 'stopped' });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to stop');
    }
  }

  const latestMetrics = metrics.length > 0 ? metrics[metrics.length - 1] : null;

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <button className="btn btn-sm btn-secondary" onClick={() => navigate('/')}>
            &larr; Back
          </button>
          <h1>Test Run {id ? id.substring(0, 8) : ''}</h1>
        </div>
        <div className="header-actions">
          <span className={`ws-indicator ${connected ? 'ws-connected' : 'ws-disconnected'}`}>
            {connected ? 'Live' : 'Disconnected'}
          </span>
          {run?.status === 'running' && (
            <button className="btn btn-danger" onClick={handleStop}>
              Stop Test
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="alert alert-error">
          {error}
          <button className="alert-dismiss" onClick={() => setError(null)}>x</button>
        </div>
      )}

      {run && (
        <div className="stats-grid">
          <div className="stat-card">
            <span className="stat-label">Status</span>
            <StatusBadge status={run.status} />
          </div>
          <div className="stat-card">
            <span className="stat-label">VUsers</span>
            <span className="stat-value">{latestMetrics?.active_vusers ?? run.total_vusers}</span>
          </div>
          <div className="stat-card">
            <span className="stat-label">Duration</span>
            <span className="stat-value">{run.duration_seconds}s</span>
          </div>
          <div className="stat-card">
            <span className="stat-label">RPS</span>
            <span className="stat-value">{latestMetrics?.total_rps.toFixed(1) ?? '-'}</span>
          </div>
          <div className="stat-card">
            <span className="stat-label">Avg Latency</span>
            <span className="stat-value">{latestMetrics?.avg_latency_ms.toFixed(1) ?? '-'} ms</span>
          </div>
          <div className="stat-card">
            <span className="stat-label">Total Requests</span>
            <span className="stat-value">{latestMetrics?.total_requests ?? '-'}</span>
          </div>
          <div className="stat-card">
            <span className="stat-label">Total Errors</span>
            <span className="stat-value stat-error">{latestMetrics?.total_errors ?? '-'}</span>
          </div>
          <div className="stat-card">
            <span className="stat-label">P99 Latency</span>
            <span className="stat-value">{latestMetrics?.p99_latency_ms.toFixed(1) ?? '-'} ms</span>
          </div>
        </div>
      )}

      {metrics.length > 0 ? (
        <MetricsChart data={metrics} />
      ) : (
        <div className="card empty-state">
          <p>Waiting for metrics data...</p>
        </div>
      )}
    </div>
  );
}
