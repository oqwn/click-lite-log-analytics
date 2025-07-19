import { useEffect, useRef, useState, useCallback } from 'react';
import { Log, LogFilter, WebSocketMessage } from '../types/log';

interface UseWebSocketOptions {
  url: string;
  onMessage?: (log: Log) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
}

interface UseWebSocketReturn {
  isConnected: boolean;
  isPaused: boolean;
  sendMessage: (message: WebSocketMessage) => void;
  pause: () => void;
  resume: () => void;
  setFilters: (filters: LogFilter[]) => void;
  disconnect: () => void;
  reconnect: () => void;
}

export const useWebSocket = ({
  url,
  onMessage,
  onConnect,
  onDisconnect,
  onError,
}: UseWebSocketOptions): UseWebSocketReturn => {
  const ws = useRef<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isPaused, setIsPaused] = useState(false);
  const reconnectTimeout = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttempts = useRef(0);

  const connect = useCallback(() => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      ws.current = new WebSocket(url);

      ws.current.onopen = () => {
        setIsConnected(true);
        reconnectAttempts.current = 0;
        onConnect?.();
      };

      ws.current.onclose = () => {
        setIsConnected(false);
        onDisconnect?.();

        // Attempt to reconnect with exponential backoff
        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 30000);
        reconnectAttempts.current++;
        
        reconnectTimeout.current = setTimeout(() => {
          connect();
        }, delay);
      };

      ws.current.onerror = (error) => {
        onError?.(error);
      };

      ws.current.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          
          if (message.type === 'log' && message.data) {
            onMessage?.(message.data as Log);
          } else if (message.type === 'status') {
            handleStatusMessage(message);
          }
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };
    } catch (error) {
      console.error('Failed to connect to WebSocket:', error);
    }
  }, [url, onConnect, onDisconnect, onError, onMessage]);

  const handleStatusMessage = (message: WebSocketMessage) => {
    if (message.data?.status === 'paused') {
      setIsPaused(true);
    } else if (message.data?.status === 'resumed') {
      setIsPaused(false);
    }
  };

  const sendMessage = useCallback((message: WebSocketMessage) => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(message));
    }
  }, []);

  const pause = useCallback(() => {
    sendMessage({ type: 'pause' });
    setIsPaused(true);
  }, [sendMessage]);

  const resume = useCallback(() => {
    sendMessage({ type: 'resume' });
    setIsPaused(false);
  }, [sendMessage]);

  const setFilters = useCallback((filters: LogFilter[]) => {
    sendMessage({ type: 'filter', filters });
  }, [sendMessage]);

  const disconnect = useCallback(() => {
    if (reconnectTimeout.current) {
      clearTimeout(reconnectTimeout.current);
    }
    ws.current?.close();
    setIsConnected(false);
  }, []);

  const reconnect = useCallback(() => {
    disconnect();
    reconnectAttempts.current = 0;
    connect();
  }, [connect, disconnect]);

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }
      ws.current?.close();
    };
  }, [connect]);

  return {
    isConnected,
    isPaused,
    sendMessage,
    pause,
    resume,
    setFilters,
    disconnect,
    reconnect,
  };
};