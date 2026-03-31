interface Props {
  status: string;
}

const STATUS_COLORS: Record<string, string> = {
  pass: '#22c55e',
  passed: '#22c55e',
  completed: '#22c55e',
  connected: '#22c55e',
  healthy: '#22c55e',
  fail: '#ef4444',
  failed: '#ef4444',
  error: '#ef4444',
  disconnected: '#ef4444',
  running: '#3b82f6',
  active: '#3b82f6',
  idle: '#6b7280',
  pending: '#6b7280',
  stopped: '#6b7280',
};

export default function StatusBadge({ status }: Props) {
  const color = STATUS_COLORS[status.toLowerCase()] || '#6b7280';

  return (
    <span
      className="status-badge"
      style={{
        backgroundColor: `${color}20`,
        color: color,
        border: `1px solid ${color}40`,
      }}
    >
      {status}
    </span>
  );
}
