import React, { useEffect, useState } from 'react';
import {
  Box,
  Container,
  Typography,
  Card,
  CardContent,
  Grid,
  Chip,
  Paper,
  IconButton,
  TextField,
  InputAdornment,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  Divider,
  LinearProgress,
  Tooltip,
} from '@mui/material';
import {
  Search,
  Refresh,
  Timeline,
  Error as ErrorIcon,
  CheckCircle,
  Warning,
  AccessTime,
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import api from '../services/api';

interface Trace {
  trace_id: string;
  start_time: string;
  end_time: string;
  duration: number;
  service_count: number;
  span_count: number;
  error_count: number;
  services: Record<string, boolean>;
}

interface TraceTimeline {
  trace_id: string;
  start_time: number;
  end_time: number;
  duration: number;
  events: TimelineEvent[];
}

interface TimelineEvent {
  id: string;
  parent_id?: string;
  service: string;
  operation: string;
  start: number;
  end: number;
  duration: number;
  status: string;
  logs: number;
}

export const TracePage: React.FC = () => {
  const navigate = useNavigate();
  const [traces, setTraces] = useState<Trace[]>([]);
  const [selectedTrace, setSelectedTrace] = useState<TraceTimeline | null>(null);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');

  const fetchTraces = async () => {
    try {
      setLoading(true);
      const response = await api.get<{ traces: Trace[] }>('/traces?limit=100');
      setTraces(response.data.traces || []);
    } catch (error) {
      console.error('Failed to fetch traces:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchTraceTimeline = async (traceId: string) => {
    try {
      const response = await api.get<TraceTimeline>(`/traces/${traceId}/timeline`);
      setSelectedTrace(response.data);
    } catch (error) {
      console.error('Failed to fetch trace timeline:', error);
    }
  };

  useEffect(() => {
    fetchTraces();
  }, []);

  const filteredTraces = traces.filter(trace => 
    trace.trace_id.includes(searchTerm) ||
    Object.keys(trace.services).some(service => 
      service.toLowerCase().includes(searchTerm.toLowerCase())
    )
  );

  const getStatusColor = (errorCount: number) => {
    if (errorCount === 0) return 'success';
    if (errorCount < 3) return 'warning';
    return 'error';
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'ok':
        return <CheckCircle color="success" fontSize="small" />;
      case 'warning':
        return <Warning color="warning" fontSize="small" />;
      case 'error':
        return <ErrorIcon color="error" fontSize="small" />;
      default:
        return null;
    }
  };

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  const renderTimeline = () => {
    if (!selectedTrace) return null;

    const totalDuration = selectedTrace.duration;
    const startTime = selectedTrace.start_time;

    return (
      <Paper sx={{ p: 3, mt: 3 }}>
        <Typography variant="h6" gutterBottom>
          Trace Timeline: {selectedTrace.trace_id}
        </Typography>
        <Box sx={{ mt: 2 }}>
          {selectedTrace.events.map((event) => {
            const relativeStart = ((event.start - startTime) / totalDuration) * 100;
            const width = (event.duration / totalDuration) * 100;

            return (
              <Box key={event.id} sx={{ mb: 2 }}>
                <Box display="flex" alignItems="center" mb={1}>
                  {getStatusIcon(event.status)}
                  <Typography variant="body2" sx={{ ml: 1 }}>
                    {event.service} - {event.operation}
                  </Typography>
                  <Typography variant="caption" color="text.secondary" sx={{ ml: 2 }}>
                    {formatDuration(event.duration)}
                  </Typography>
                </Box>
                <Box sx={{ position: 'relative', height: 30, bgcolor: 'grey.100', borderRadius: 1 }}>
                  <Tooltip title={`${event.operation} (${formatDuration(event.duration)})`}>
                    <Box
                      sx={{
                        position: 'absolute',
                        left: `${relativeStart}%`,
                        width: `${Math.max(width, 1)}%`,
                        height: '100%',
                        bgcolor: event.status === 'error' ? 'error.main' : 'primary.main',
                        borderRadius: 1,
                        cursor: 'pointer',
                        '&:hover': {
                          opacity: 0.8,
                        },
                      }}
                    />
                  </Tooltip>
                </Box>
              </Box>
            );
          })}
        </Box>
      </Paper>
    );
  };

  if (loading) {
    return (
      <Container maxWidth="xl">
        <LinearProgress />
      </Container>
    );
  }

  return (
    <Container maxWidth="xl">
      <Box py={3}>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
          <Typography variant="h4">
            <Timeline sx={{ mr: 1, verticalAlign: 'middle' }} />
            Distributed Traces
          </Typography>
          <IconButton onClick={fetchTraces}>
            <Refresh />
          </IconButton>
        </Box>

        <TextField
          fullWidth
          variant="outlined"
          placeholder="Search by trace ID or service..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          sx={{ mb: 3 }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <Search />
              </InputAdornment>
            ),
          }}
        />

        <Grid container spacing={3}>
          <Grid item xs={12} md={selectedTrace ? 4 : 12}>
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Recent Traces ({filteredTraces.length})
                </Typography>
                <List>
                  {filteredTraces.map((trace, index) => (
                    <React.Fragment key={trace.trace_id}>
                      <ListItem
                        button
                        onClick={() => fetchTraceTimeline(trace.trace_id)}
                        selected={selectedTrace?.trace_id === trace.trace_id}
                      >
                        <ListItemText
                          primary={
                            <Box display="flex" alignItems="center">
                              <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                                {trace.trace_id.substring(0, 16)}...
                              </Typography>
                              <Chip
                                size="small"
                                label={`${trace.error_count} errors`}
                                color={getStatusColor(trace.error_count)}
                                sx={{ ml: 1 }}
                              />
                            </Box>
                          }
                          secondary={
                            <Box>
                              <Typography variant="caption" display="block">
                                {Object.keys(trace.services).join(', ')}
                              </Typography>
                              <Typography variant="caption" color="text.secondary">
                                {trace.span_count} spans • {formatDuration(trace.duration)}
                              </Typography>
                            </Box>
                          }
                        />
                        <ListItemSecondaryAction>
                          <AccessTime fontSize="small" color="action" />
                        </ListItemSecondaryAction>
                      </ListItem>
                      {index < filteredTraces.length - 1 && <Divider />}
                    </React.Fragment>
                  ))}
                </List>
              </CardContent>
            </Card>
          </Grid>

          {selectedTrace && (
            <Grid item xs={12} md={8}>
              <Card>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    Trace Details
                  </Typography>
                  <Grid container spacing={2}>
                    <Grid item xs={6} sm={3}>
                      <Typography variant="caption" color="text.secondary">
                        Duration
                      </Typography>
                      <Typography variant="body1">
                        {formatDuration(selectedTrace.duration)}
                      </Typography>
                    </Grid>
                    <Grid item xs={6} sm={3}>
                      <Typography variant="caption" color="text.secondary">
                        Services
                      </Typography>
                      <Typography variant="body1">
                        {selectedTrace.events.length}
                      </Typography>
                    </Grid>
                    <Grid item xs={6} sm={3}>
                      <Typography variant="caption" color="text.secondary">
                        Total Spans
                      </Typography>
                      <Typography variant="body1">
                        {selectedTrace.events.length}
                      </Typography>
                    </Grid>
                    <Grid item xs={6} sm={3}>
                      <Typography variant="caption" color="text.secondary">
                        Errors
                      </Typography>
                      <Typography variant="body1">
                        {selectedTrace.events.filter(e => e.status === 'error').length}
                      </Typography>
                    </Grid>
                  </Grid>
                  {renderTimeline()}
                </CardContent>
              </Card>
            </Grid>
          )}
        </Grid>
      </Box>
    </Container>
  );
};