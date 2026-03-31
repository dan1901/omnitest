import { useEffect, useState } from 'react';
import { fetchAgents } from '../api/client.ts';
import type { AgentInfo } from '../api/types.ts';
import StatusBadge from '../components/StatusBadge.tsx';

export default function AgentsPage() {
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAgents();
    const interval = setInterval(loadAgents, 10000);
    return () => clearInterval(interval);
  }, []);

  async function loadAgents() {
    try {
      const data = await fetchAgents();
      setAgents(data || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load agents');
    } finally {
      setLoading(false);
    }
  }

  function timeSince(heartbeat: string): string {
    try {
      const diff = Date.now() - new Date(heartbeat).getTime();
      const secs = Math.floor(diff / 1000);
      if (secs < 60) return `${secs}s ago`;
      const mins = Math.floor(secs / 60);
      if (mins < 60) return `${mins}m ago`;
      return `${Math.floor(mins / 60)}h ago`;
    } catch {
      return heartbeat;
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <h1>Agents</h1>
        <button className="btn btn-secondary" onClick={loadAgents}>
          Refresh
        </button>
      </div>

      {error && (
        <div className="alert alert-error">
          {error}
          <button className="alert-dismiss" onClick={() => setError(null)}>x</button>
        </div>
      )}

      {loading ? (
        <div className="loading">Loading agents...</div>
      ) : agents.length === 0 ? (
        <div className="empty-state">
          <p>No agents connected.</p>
        </div>
      ) : (
        <div className="agents-grid">
          {agents.map((agent) => (
            <div className="card agent-card" key={agent.agent_id}>
              <div className="agent-header">
                <h3>{agent.hostname}</h3>
                <StatusBadge status={agent.status} />
              </div>
              <div className="agent-id">{agent.agent_id.substring(0, 12)}</div>

              <div className="agent-stats">
                <div className="agent-stat">
                  <span className="stat-label">VUsers</span>
                  <span className="stat-value">{agent.active_vusers} / {agent.max_vusers}</span>
                </div>
                <div className="agent-stat">
                  <span className="stat-label">CPU</span>
                  <div className="progress-bar">
                    <div
                      className="progress-fill"
                      style={{
                        width: `${Math.min(agent.cpu_usage, 100)}%`,
                        backgroundColor: agent.cpu_usage > 80 ? '#ef4444' : '#3b82f6',
                      }}
                    />
                  </div>
                  <span className="stat-pct">{agent.cpu_usage.toFixed(1)}%</span>
                </div>
                <div className="agent-stat">
                  <span className="stat-label">Memory</span>
                  <div className="progress-bar">
                    <div
                      className="progress-fill"
                      style={{
                        width: `${Math.min(agent.memory_usage, 100)}%`,
                        backgroundColor: agent.memory_usage > 80 ? '#ef4444' : '#22c55e',
                      }}
                    />
                  </div>
                  <span className="stat-pct">{agent.memory_usage.toFixed(1)}%</span>
                </div>
                <div className="agent-stat">
                  <span className="stat-label">Last Heartbeat</span>
                  <span className="stat-value">{timeSince(agent.last_heartbeat)}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
