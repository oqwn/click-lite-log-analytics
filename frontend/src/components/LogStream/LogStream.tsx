import React, { useState, useRef, useEffect, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  IconButton,
  Chip,
  TextField,
  InputAdornment,
  Tooltip,
  CircularProgress,
  Alert,
} from '@mui/material';
import {
  PlayArrow,
  Pause,
  Clear,
  FilterList,
  Search,
  Refresh,
  FiberManualRecord,
} from '@mui/icons-material';
import { format } from 'date-fns';
import { Log, LogFilter, LOG_LEVEL_COLORS } from '../../types/log';
import { useWebSocket } from '../../hooks/useWebSocket';
import LogEntry from './LogEntry';
import FilterDialog from './FilterDialog';

const MAX_LOGS = 1000; // Maximum number of logs to keep in memory

interface LogStreamProps {
  wsUrl: string;
}

const LogStream: React.FC<LogStreamProps> = ({ wsUrl }) => {
  const [logs, setLogs] = useState<Log[]>([]);
  const [filters, setFilters] = useState<LogFilter[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [showFilterDialog, setShowFilterDialog] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);
  const logsEndRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleNewLog = useCallback((log: Log) => {
    setLogs((prevLogs) => {
      const newLogs = [...prevLogs, log];
      // Keep only the last MAX_LOGS entries
      if (newLogs.length > MAX_LOGS) {
        return newLogs.slice(-MAX_LOGS);
      }
      return newLogs;
    });
  }, []);

  const {
    isConnected,
    isPaused,
    pause,
    resume,
    setFilters: applyFilters,
    reconnect,
  } = useWebSocket({
    url: wsUrl,
    onMessage: handleNewLog,
  });

  // Auto-scroll to bottom when new logs arrive
  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  // Detect manual scroll
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = container;
      const isAtBottom = scrollHeight - scrollTop - clientHeight < 10;
      setAutoScroll(isAtBottom);
    };

    container.addEventListener('scroll', handleScroll);
    return () => container.removeEventListener('scroll', handleScroll);
  }, []);

  const handleClearLogs = () => {
    setLogs([]);
  };

  const handleTogglePause = () => {
    if (isPaused) {
      resume();
    } else {
      pause();
    }
  };

  const handleApplyFilters = (newFilters: LogFilter[]) => {
    setFilters(newFilters);
    applyFilters(newFilters);
    setShowFilterDialog(false);
  };

  const filteredLogs = logs.filter((log) => {
    if (!searchTerm) return true;
    const searchLower = searchTerm.toLowerCase();
    return (
      log.message.toLowerCase().includes(searchLower) ||
      log.service.toLowerCase().includes(searchLower) ||
      log.level.toLowerCase().includes(searchLower)
    );
  });

  return (
    <Paper elevation={2} sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box
        sx={{
          p: 2,
          borderBottom: 1,
          borderColor: 'divider',
          display: 'flex',
          alignItems: 'center',
          gap: 2,
        }}
      >
        <Typography variant="h6" sx={{ flexGrow: 0 }}>
          Real-time Logs
        </Typography>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <FiberManualRecord
            sx={{
              fontSize: 12,
              color: isConnected ? 'success.main' : 'error.main',
            }}
          />
          <Typography variant="body2" color="text.secondary">
            {isConnected ? 'Connected' : 'Disconnected'}
          </Typography>
        </Box>

        <TextField
          size="small"
          placeholder="Search logs..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          sx={{ flexGrow: 1, maxWidth: 400 }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <Search />
              </InputAdornment>
            ),
          }}
        />

        <Box sx={{ display: 'flex', gap: 1 }}>
          <Tooltip title={isPaused ? 'Resume' : 'Pause'}>
            <IconButton onClick={handleTogglePause} color="primary">
              {isPaused ? <PlayArrow /> : <Pause />}
            </IconButton>
          </Tooltip>

          <Tooltip title="Clear logs">
            <IconButton onClick={handleClearLogs}>
              <Clear />
            </IconButton>
          </Tooltip>

          <Tooltip title="Filters">
            <IconButton onClick={() => setShowFilterDialog(true)}>
              <FilterList />
              {filters.length > 0 && (
                <Chip
                  label={filters.length}
                  size="small"
                  color="primary"
                  sx={{
                    position: 'absolute',
                    top: -4,
                    right: -4,
                    height: 20,
                    fontSize: 10,
                  }}
                />
              )}
            </IconButton>
          </Tooltip>

          <Tooltip title="Reconnect">
            <span>
              <IconButton onClick={reconnect} disabled={isConnected}>
                <Refresh />
              </IconButton>
            </span>
          </Tooltip>
        </Box>
      </Box>

      {/* Status Bar */}
      {(isPaused || !isConnected) && (
        <Alert
          severity={isPaused ? 'warning' : 'error'}
          sx={{ borderRadius: 0 }}
        >
          {isPaused
            ? 'Log streaming is paused. Click play to resume.'
            : 'Connection lost. Click refresh to reconnect.'}
        </Alert>
      )}

      {/* Log Container */}
      <Box
        ref={containerRef}
        sx={{
          flexGrow: 1,
          overflowY: 'auto',
          p: 2,
          bgcolor: 'grey.50',
        }}
      >
        {filteredLogs.length === 0 ? (
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              color: 'text.secondary',
            }}
          >
            {isConnected ? (
              <>
                <Typography variant="h6">No logs yet</Typography>
                <Typography variant="body2">
                  Waiting for incoming logs...
                </Typography>
              </>
            ) : (
              <>
                <Typography variant="h6">Not connected</Typography>
                <Typography variant="body2">
                  Click refresh to connect to the log stream
                </Typography>
              </>
            )}
          </Box>
        ) : (
          <>
            {filteredLogs.map((log, index) => (
              <LogEntry key={`${log.id}-${index}`} log={log} />
            ))}
            <div ref={logsEndRef} />
          </>
        )}
      </Box>

      {/* Footer */}
      <Box
        sx={{
          p: 1,
          borderTop: 1,
          borderColor: 'divider',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <Typography variant="caption" color="text.secondary">
          Showing {filteredLogs.length} of {logs.length} logs
          {logs.length >= MAX_LOGS && ` (limited to last ${MAX_LOGS})`}
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Auto-scroll: {autoScroll ? 'ON' : 'OFF'}
        </Typography>
      </Box>

      {/* Filter Dialog */}
      <FilterDialog
        open={showFilterDialog}
        filters={filters}
        onClose={() => setShowFilterDialog(false)}
        onApply={handleApplyFilters}
      />
    </Paper>
  );
};

export default LogStream;