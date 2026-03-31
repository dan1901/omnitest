import { useEffect, useRef, useState, useCallback } from 'react';

const WS_BASE = import.meta.env.VITE_API_URL
  ? import.meta.env.VITE_API_URL.replace(/^http/, 'ws')
  : 'ws://localhost:8080';

interface WsMessage<T> {
  type: string;
  data: T;
}

export function useWebSocket<T>(runId: string | undefined) {
  const [data, setData] = useState<T[]>([]);
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const connect = useCallback(() => {
    if (!runId) return;

    const ws = new WebSocket(`${WS_BASE}/ws/metrics/${runId}`);
    wsRef.current = ws;

    ws.onopen = () => setConnected(true);

    ws.onclose = () => {
      setConnected(false);
      reconnectTimer.current = setTimeout(connect, 3000);
    };

    ws.onerror = () => {
      ws.close();
    };

    ws.onmessage = (event) => {
      try {
        const msg: WsMessage<T> = JSON.parse(event.data);
        if (msg.type === 'metrics_update') {
          setData((prev) => [...prev.slice(-299), msg.data]);
        }
      } catch {
        // ignore non-JSON messages
      }
    };
  }, [runId]);

  useEffect(() => {
    connect();
    return () => {
      if (reconnectTimer.current) clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
    };
  }, [connect]);

  const clear = useCallback(() => setData([]), []);

  return { data, connected, clear };
}
