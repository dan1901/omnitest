import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
} from 'recharts';
import type { AggregatedMetrics } from '../api/types.ts';

interface Props {
  data: AggregatedMetrics[];
}

function formatTime(ts: string): string {
  try {
    return new Date(ts).toLocaleTimeString();
  } catch {
    return ts;
  }
}

export default function MetricsChart({ data }: Props) {
  const chartData = data.map((m) => ({
    time: formatTime(m.timestamp),
    rps: m.total_rps,
    avg_latency: m.avg_latency_ms,
    p95_latency: m.p95_latency_ms,
    p99_latency: m.p99_latency_ms,
    errors: m.total_errors,
    vusers: m.active_vusers,
  }));

  return (
    <div className="charts-grid">
      <div className="card">
        <h3>Requests Per Second</h3>
        <ResponsiveContainer width="100%" height={250}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
            <XAxis dataKey="time" stroke="#94a3b8" fontSize={12} />
            <YAxis stroke="#94a3b8" fontSize={12} />
            <Tooltip
              contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #334155', borderRadius: '6px' }}
              labelStyle={{ color: '#e2e8f0' }}
            />
            <Legend />
            <Line type="monotone" dataKey="rps" stroke="#3b82f6" name="RPS" strokeWidth={2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div className="card">
        <h3>Latency (ms)</h3>
        <ResponsiveContainer width="100%" height={250}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
            <XAxis dataKey="time" stroke="#94a3b8" fontSize={12} />
            <YAxis stroke="#94a3b8" fontSize={12} />
            <Tooltip
              contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #334155', borderRadius: '6px' }}
              labelStyle={{ color: '#e2e8f0' }}
            />
            <Legend />
            <Line type="monotone" dataKey="avg_latency" stroke="#22c55e" name="Avg" strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="p95_latency" stroke="#f59e0b" name="P95" strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="p99_latency" stroke="#ef4444" name="P99" strokeWidth={2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div className="card">
        <h3>Active VUsers & Errors</h3>
        <ResponsiveContainer width="100%" height={250}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
            <XAxis dataKey="time" stroke="#94a3b8" fontSize={12} />
            <YAxis yAxisId="left" stroke="#94a3b8" fontSize={12} />
            <YAxis yAxisId="right" orientation="right" stroke="#94a3b8" fontSize={12} />
            <Tooltip
              contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #334155', borderRadius: '6px' }}
              labelStyle={{ color: '#e2e8f0' }}
            />
            <Legend />
            <Line yAxisId="left" type="monotone" dataKey="vusers" stroke="#8b5cf6" name="VUsers" strokeWidth={2} dot={false} />
            <Line yAxisId="right" type="monotone" dataKey="errors" stroke="#ef4444" name="Errors" strokeWidth={2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
