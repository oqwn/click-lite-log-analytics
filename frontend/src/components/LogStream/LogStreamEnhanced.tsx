import React, { useState, useEffect, useRef } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  IconButton,
  Chip,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Grid,
  Switch,
  FormControlLabel,
  Tooltip,
} from '@mui/material';
import {
  PlayArrow,
  Pause,
  Clear,
  FilterList,
  Download,
  Refresh,
  Settings,
} from '@mui/icons-material';
import { LogFilter, WebSocketMessage } from '../../types/api';

interface LogEntry {
  id: string;
  timestamp: string;
  level: string;
  service: string;
  message: string;
  trace_id?: string;
}

interface LogStreamEnhancedProps {
  wsUrl: string;
}

export const LogStreamEnhanced: React.FC<LogStreamEnhancedProps> = ({ wsUrl }) => {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<LogFilter[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedLevel, setSelectedLevel] = useState<string>('');
  const [selectedService, setSelectedService] = useState<string>('');
  const [autoScroll, setAutoScroll] = useState(true);
  const [maxLogs, setMaxLogs] = useState(1000);
  
  const wsRef = useRef<WebSocket | null>(null);
  const tableRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    connectWebSocket();
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [wsUrl]);

  useEffect(() => {
    if (autoScroll && tableRef.current) {
      tableRef.current.scrollTop = tableRef.current.scrollHeight;
    }
  }, [logs, autoScroll]);

  const connectWebSocket = () => {
    try {
      if (wsRef.current) {
        wsRef.current.close();
      }

      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        setIsConnected(true);
        setError(null);
        // WebSocket connected
      };

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          
          if (message.type === 'log_entry' && message.data) {
            const newLog: LogEntry = {
              id: message.data.id || `${Date.now()}-${Math.random()}`,
              timestamp: message.data.timestamp,
              level: message.data.level,
              service: message.data.service,
              message: message.data.message,
              trace_id: message.data.trace_id,
            };
            
            setLogs(prev => {
              const updated = [...prev, newLog];
              return updated.slice(-maxLogs); // Keep only the last N logs
            });
          }
        } catch (err) {
          // Error parsing WebSocket message
        }
      };

      ws.onclose = () => {
        setIsConnected(false);
        setIsStreaming(false);
        // WebSocket disconnected
      };

      ws.onerror = (err) => {
        setError('WebSocket connection error');
        // WebSocket error
      };
    } catch (err) {
      setError('Failed to connect to WebSocket');
      // WebSocket connection error
    }
  };

  const startStreaming = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      const message: WebSocketMessage = {
        type: 'tail',
        action: 'start',
        filters: getCurrentFilters(),
      };
      wsRef.current.send(JSON.stringify(message));
      setIsStreaming(true);
    }
  };

  const stopStreaming = () => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      const message: WebSocketMessage = {
        type: 'tail',
        action: 'stop',
      };
      wsRef.current.send(JSON.stringify(message));
    }
    setIsStreaming(false);
  };

  const getCurrentFilters = (): LogFilter[] => {
    const currentFilters: LogFilter[] = [];
    
    if (selectedLevel) {
      currentFilters.push({
        field: 'level',
        operator: 'equals',
        value: selectedLevel,
      });
    }
    
    if (selectedService) {
      currentFilters.push({
        field: 'service',
        operator: 'equals',
        value: selectedService,
      });
    }
    
    if (searchTerm) {
      currentFilters.push({
        field: 'message',
        operator: 'contains',
        value: searchTerm,
      });
    }
    
    return [...currentFilters, ...filters];
  };

  const applyFilters = () => {
    if (isStreaming) {
      stopStreaming();
      setTimeout(startStreaming, 100);
    }
  };

  const clearLogs = () => {
    setLogs([]);
  };

  const exportLogs = () => {
    const csv = [
      'timestamp,level,service,message,trace_id',
      ...filteredLogs.map(log => 
        `"${log.timestamp}","${log.level}","${log.service}","${log.message.replace(/"/g, '""')}","${log.trace_id || ''}"`
      )
    ].join('\n');

    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `logs-${new Date().toISOString().split('T')[0]}.csv`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const filteredLogs = logs.filter(log => {
    if (searchTerm && !log.message.toLowerCase().includes(searchTerm.toLowerCase())) {
      return false;
    }
    if (selectedLevel && log.level !== selectedLevel) {
      return false;
    }
    if (selectedService && log.service !== selectedService) {
      return false;
    }
    return true;
  });

  const uniqueLevels = [...new Set(logs.map(log => log.level))];
  const uniqueServices = [...new Set(logs.map(log => log.service))];

  const getLevelColor = (level: string) => {
    switch (level.toLowerCase()) {
      case 'error': return 'error';
      case 'warn': 
      case 'warning': return 'warning';
      case 'info': return 'info';
      case 'debug': return 'default';
      default: return 'default';
    }
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleTimeString();
  };

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Controls */}
      <Paper sx={{ p: 2, mb: 2 }}>
        <Grid container spacing={2} alignItems="center">
          <Grid item xs={12} md={3}>
            <TextField
              fullWidth
              size="small"
              label="Search messages"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              variant="outlined"
            />
          </Grid>
          
          <Grid item xs={6} md={2}>
            <FormControl fullWidth size="small">
              <InputLabel>Level</InputLabel>
              <Select
                value={selectedLevel}
                onChange={(e) => setSelectedLevel(e.target.value)}
                label="Level"
              >
                <MenuItem value="">All</MenuItem>
                {uniqueLevels.map(level => (
                  <MenuItem key={level} value={level}>{level}</MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          
          <Grid item xs={6} md={2}>
            <FormControl fullWidth size="small">
              <InputLabel>Service</InputLabel>
              <Select
                value={selectedService}
                onChange={(e) => setSelectedService(e.target.value)}
                label="Service"
              >
                <MenuItem value="">All</MenuItem>
                {uniqueServices.map(service => (
                  <MenuItem key={service} value={service}>{service}</MenuItem>
                ))}
              </Select>
            </FormControl>
          </Grid>
          
          <Grid item xs={12} md={5}>
            <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
              <Button
                variant={isStreaming ? "outlined" : "contained"}
                startIcon={isStreaming ? <Pause /> : <PlayArrow />}
                onClick={isStreaming ? stopStreaming : startStreaming}
                disabled={!isConnected}
                color={isStreaming ? "secondary" : "primary"}
              >
                {isStreaming ? 'Pause' : 'Start'} Stream
              </Button>
              
              <IconButton onClick={applyFilters} disabled={!isConnected}>
                <FilterList />
              </IconButton>
              
              <IconButton onClick={clearLogs}>
                <Clear />
              </IconButton>
              
              <IconButton onClick={exportLogs} disabled={filteredLogs.length === 0}>
                <Download />
              </IconButton>
              
              <IconButton onClick={connectWebSocket}>
                <Refresh />
              </IconButton>
              
              <FormControlLabel
                control={
                  <Switch
                    checked={autoScroll}
                    onChange={(e) => setAutoScroll(e.target.checked)}
                  />
                }
                label="Auto-scroll"
              />
            </Box>
          </Grid>
        </Grid>

        {/* Status indicators */}
        <Box sx={{ display: 'flex', gap: 1, mt: 2, alignItems: 'center' }}>
          <Chip
            label={isConnected ? 'Connected' : 'Disconnected'}
            color={isConnected ? 'success' : 'error'}
            size="small"
          />
          <Chip
            label={isStreaming ? 'Streaming' : 'Paused'}
            color={isStreaming ? 'primary' : 'default'}
            size="small"
          />
          <Chip
            label={`${filteredLogs.length} / ${logs.length} logs`}
            variant="outlined"
            size="small"
          />
        </Box>
      </Paper>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      {/* Log table */}
      <Paper sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
        <TableContainer 
          ref={tableRef}
          sx={{ flexGrow: 1, maxHeight: 'calc(100vh - 300px)' }}
        >
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow>
                <TableCell>Time</TableCell>
                <TableCell>Level</TableCell>
                <TableCell>Service</TableCell>
                <TableCell>Message</TableCell>
                <TableCell>Trace ID</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredLogs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} align="center">
                    <Typography color="text.secondary">
                      {logs.length === 0 
                        ? 'No logs received yet. Start streaming to see live logs.'
                        : 'No logs match your current filters.'
                      }
                    </Typography>
                  </TableCell>
                </TableRow>
              ) : (
                filteredLogs.map((log) => (
                  <TableRow key={log.id} hover>
                    <TableCell>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                        {formatTimestamp(log.timestamp)}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={log.level}
                        size="small"
                        color={getLevelColor(log.level) as any}
                      />
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {log.service}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography
                        variant="body2"
                        sx={{
                          maxWidth: 400,
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          whiteSpace: 'nowrap',
                        }}
                        title={log.message}
                      >
                        {log.message}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      {log.trace_id && (
                        <Typography
                          variant="body2"
                          sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}
                        >
                          {log.trace_id}
                        </Typography>
                      )}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>
    </Box>
  );
};