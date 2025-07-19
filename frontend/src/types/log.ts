export interface Log {
  id: string;
  timestamp: string;
  level: string;
  message: string;
  service: string;
  trace_id?: string;
  span_id?: string;
  attributes?: Record<string, any>;
}

export interface LogFilter {
  field: string;
  operator: string;
  value: string;
}

export interface WebSocketMessage {
  type: string;
  action?: string;
  data?: any;
  filters?: LogFilter[];
}

export type LogLevel = 'debug' | 'info' | 'warn' | 'error' | 'fatal';

export const LOG_LEVELS: LogLevel[] = ['debug', 'info', 'warn', 'error', 'fatal'];

export const LOG_LEVEL_COLORS: Record<LogLevel, string> = {
  debug: '#9e9e9e',
  info: '#2196f3',
  warn: '#ff9800',
  error: '#f44336',
  fatal: '#d32f2f',
};