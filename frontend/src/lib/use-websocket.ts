'use client';

import { useEffect, useRef, useState } from 'react';

const WS_BASE = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080';

/**
 * useWebSocket — se connecte au canal temps réel d'une commande
 * et appelle onUpdate à chaque événement reçu.
 */
export function useWebSocket<T = any>(path: string, onUpdate?: (data: T) => void) {
  const [connected, setConnected] = useState(false);
  const socketRef = useRef<WebSocket | null>(null);
  const onUpdateRef = useRef(onUpdate);
  onUpdateRef.current = onUpdate;

  useEffect(() => {
    let retryTimer: ReturnType<typeof setTimeout>;

    const connect = () => {
      const token = localStorage.getItem('access_token');
      if (!token) return;

      const base = WS_BASE.replace(/\/$/, '');
      const separator = path.includes('?') ? '&' : '?';
      const ws = new WebSocket(`${base}${path}${separator}token=${encodeURIComponent(token)}`);
      socketRef.current = ws;

      ws.onopen = () => setConnected(true);

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          onUpdateRef.current?.(data);
        } catch {
          // payload non JSON ignoré
        }
      };

      ws.onclose = () => {
        setConnected(false);
        retryTimer = setTimeout(connect, 3000);
      };

      ws.onerror = () => ws.close();
    };

    connect();

    return () => {
      clearTimeout(retryTimer);
      socketRef.current?.close();
    };
  }, [path]);

  return { connected, socket: socketRef.current };
}
