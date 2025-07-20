import React, { useEffect, useState } from 'react';
import {
  Box,
  Container,
  Grid,
  Paper,
  Typography,
  Card,
  CardContent,
  LinearProgress,
  Chip,
  Alert,
  AlertTitle,
  IconButton,
  Tab,
  Tabs,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Divider,
} from '@mui/material';
import {
  Speed,
  Storage,
  Memory,
  Warning,
  Error,
  CheckCircle,
  Refresh,
  TrendingUp,
  TrendingDown,
  QueryStats,
} from '@mui/icons-material';
import api from '../services/api';

interface HealthComponent {
  name: string;
  status: 'ok' | 'degraded' | 'down';
  message?: string;
  last_checked: string;
  response_time_ms: number;
  details?: Record<string, unknown>;
}

interface SystemHealth {
  status: 'ok' | 'degraded' | 'down';
  timestamp: string;
  version: string;
  uptime_seconds: number;
  components: Record<string, HealthComponent>;
  system_info: {
    go_version: string;
    num_goroutines: number;
    memory_alloc_mb: number;
    memory_total_mb: number;
    num_cpu: number;
    storage_used_gb: number;
    storage_total_gb: number;
  };
}

interface Metric {
  name: string;
  type: 'counter' | 'gauge' | 'histogram';
  value: number;
  labels?: Record<string, string>;
  timestamp: string;
  description?: string;
}

interface Alert {
  id: string;
  name: string;
  severity: 'info' | 'warning' | 'critical';
  status: 'active' | 'resolved';
  message: string;
  source: string;
  start_time: string;
  end_time?: string;
  last_updated: string;
  count: number;
  details?: unknown;
}

export const MonitoringPage: React.FC = () => {
  const [health, setHealth] = useState<SystemHealth | null>(null);
  const [metrics, setMetrics] = useState<Metric[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [activeTab, setActiveTab] = useState(0);
  const [loading, setLoading] = useState(true);

  const fetchMonitoringData = async () => {
    try {
      // Handle health endpoint (may return 503 when degraded, but still has data)
      const healthPromise = api.get<SystemHealth>('/monitoring/health').catch(error => {
        if (error.response?.status === 503) {
          // 503 still contains valid health data
          return error.response;
        }
        throw error;
      });
      
      const [healthRes, metricsRes, alertsRes] = await Promise.all([
        healthPromise,
        api.get<{ metrics: Metric[] }>('/monitoring/metrics'),
        api.get<{ alerts: Alert[] }>('/monitoring/alerts/active'),
      ]);

      setHealth(healthRes.data);
      setMetrics(metricsRes.data.metrics);
      setAlerts(alertsRes.data.alerts);
    } catch (error) {
      // Silent error handling - monitoring will retry automatically
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMonitoringData();
    const interval = setInterval(fetchMonitoringData, 5000); // Refresh every 5 seconds

    return () => {
      if (interval) clearInterval(interval);
    };
  }, []);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'ok':
        return 'success';
      case 'degraded':
        return 'warning';
      case 'down':
        return 'error';
      default:
        return 'default';
    }
  };

  const getSeverityColor = (severity: string): 'error' | 'warning' | 'info' | 'success' => {
    switch (severity) {
      case 'critical':
        return 'error';
      case 'warning':
        return 'warning';
      case 'info':
        return 'info';
      default:
        return 'success';
    }
  };

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${days}d ${hours}h ${minutes}m`;
  };

  const getMetricValue = (name: string): number => {
    const metric = metrics.find(m => m.name === name);
    return metric?.value || 0;
  };

  const renderMetricCard = (
    title: string,
    value: string | number,
    icon: React.ReactNode,
    trend?: 'up' | 'down' | 'neutral',
    color: 'primary' | 'secondary' | 'error' | 'warning' | 'info' | 'success' = 'primary'
  ) => (
    <Card>
      <CardContent>
        <Box display="flex" alignItems="center" justifyContent="space-between">
          <Box>
            <Typography color="textSecondary" gutterBottom variant="body2">
              {title}
            </Typography>
            <Typography variant="h4">{value}</Typography>
            {trend && (
              <Box display="flex" alignItems="center" mt={1}>
                {trend === 'up' ? (
                  <TrendingUp color="success" fontSize="small" />
                ) : trend === 'down' ? (
                  <TrendingDown color="error" fontSize="small" />
                ) : null}
              </Box>
            )}
          </Box>
          <Box color={`${color}.main`}>{icon}</Box>
        </Box>
      </CardContent>
    </Card>
  );

  const renderHealthStatus = () => (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <Paper sx={{ p: 2 }}>
          <Box display="flex" alignItems="center" justifyContent="space-between" mb={2}>
            <Typography variant="h6">System Health</Typography>
            <Box display="flex" alignItems="center" gap={1}>
              <Chip
                label={health?.status.toUpperCase()}
                color={getStatusColor(health?.status || 'unknown')}
                size="small"
              />
              <IconButton size="small" onClick={fetchMonitoringData}>
                <Refresh />
              </IconButton>
            </Box>
          </Box>

          <Grid container spacing={2}>
            <Grid item xs={12} md={3}>
              <Typography variant="body2" color="textSecondary">
                Version
              </Typography>
              <Typography variant="body1">{health?.version}</Typography>
            </Grid>
            <Grid item xs={12} md={3}>
              <Typography variant="body2" color="textSecondary">
                Uptime
              </Typography>
              <Typography variant="body1">
                {health ? formatUptime(health.uptime_seconds) : '-'}
              </Typography>
            </Grid>
            <Grid item xs={12} md={3}>
              <Typography variant="body2" color="textSecondary">
                Go Version
              </Typography>
              <Typography variant="body1">{health?.system_info.go_version}</Typography>
            </Grid>
            <Grid item xs={12} md={3}>
              <Typography variant="body2" color="textSecondary">
                Goroutines
              </Typography>
              <Typography variant="body1">{health?.system_info.num_goroutines}</Typography>
            </Grid>
          </Grid>

          <Divider sx={{ my: 2 }} />

          <Typography variant="subtitle2" gutterBottom>
            Components
          </Typography>
          <List>
            {health &&
              Object.entries(health.components).map(([name, component]) => (
                <ListItem key={name}>
                  <ListItemIcon>
                    {component.status === 'ok' ? (
                      <CheckCircle color="success" />
                    ) : component.status === 'degraded' ? (
                      <Warning color="warning" />
                    ) : (
                      <Error color="error" />
                    )}
                  </ListItemIcon>
                  <ListItemText
                    primary={name}
                    secondary={
                      <>
                        {component.message && (
                          <Typography variant="caption" display="block">
                            {component.message}
                          </Typography>
                        )}
                        <Typography variant="caption" color="textSecondary">
                          Response time: {component.response_time_ms}ms
                        </Typography>
                      </>
                    }
                  />
                </ListItem>
              ))}
          </List>
        </Paper>
      </Grid>
    </Grid>
  );

  const renderMetrics = () => (
    <Grid container spacing={3}>
      {/* Key Metrics Cards */}
      <Grid item xs={12} sm={6} md={3}>
        {renderMetricCard(
          'Ingestion Rate',
          `${getMetricValue('ingestion_rate_per_second').toFixed(0)}/s`,
          <Speed fontSize="large" />,
          'neutral',
          'primary'
        )}
      </Grid>
      <Grid item xs={12} sm={6} md={3}>
        {renderMetricCard(
          'Query Rate',
          `${getMetricValue('query_rate_per_second').toFixed(0)}/s`,
          <QueryStats fontSize="large" />,
          'neutral',
          'secondary'
        )}
      </Grid>
      <Grid item xs={12} sm={6} md={3}>
        {renderMetricCard(
          'Storage Used',
          `${(getMetricValue('storage_size_mb') / 1024).toFixed(1)} GB`,
          <Storage fontSize="large" />,
          'up',
          'warning'
        )}
      </Grid>
      <Grid item xs={12} sm={6} md={3}>
        {renderMetricCard(
          'Memory Usage',
          `${health?.system_info.memory_alloc_mb.toFixed(0) || 0} MB`,
          <Memory fontSize="large" />,
          'neutral',
          'info'
        )}
      </Grid>

      {/* Performance Metrics */}
      <Grid item xs={12} md={6}>
        <Paper sx={{ p: 2 }}>
          <Typography variant="h6" gutterBottom>
            Query Performance
          </Typography>
          <Grid container spacing={2}>
            <Grid item xs={4}>
              <Typography variant="body2" color="textSecondary">
                Avg Duration
              </Typography>
              <Typography variant="h6">
                {getMetricValue('query_duration_ms_avg').toFixed(0)} ms
              </Typography>
            </Grid>
            <Grid item xs={4}>
              <Typography variant="body2" color="textSecondary">
                P90 Duration
              </Typography>
              <Typography variant="h6">
                {getMetricValue('query_duration_ms_p90').toFixed(0)} ms
              </Typography>
            </Grid>
            <Grid item xs={4}>
              <Typography variant="body2" color="textSecondary">
                P99 Duration
              </Typography>
              <Typography variant="h6">
                {getMetricValue('query_duration_ms_p99').toFixed(0)} ms
              </Typography>
            </Grid>
          </Grid>
        </Paper>
      </Grid>

      <Grid item xs={12} md={6}>
        <Paper sx={{ p: 2 }}>
          <Typography variant="h6" gutterBottom>
            Ingestion Statistics
          </Typography>
          <Grid container spacing={2}>
            <Grid item xs={6}>
              <Typography variant="body2" color="textSecondary">
                Total Logs Ingested
              </Typography>
              <Typography variant="h6">
                {getMetricValue('total_logs_ingested').toLocaleString()}
              </Typography>
            </Grid>
            <Grid item xs={6}>
              <Typography variant="body2" color="textSecondary">
                Total Queries Executed
              </Typography>
              <Typography variant="h6">
                {getMetricValue('total_queries_executed').toLocaleString()}
              </Typography>
            </Grid>
          </Grid>
        </Paper>
      </Grid>
    </Grid>
  );

  const renderAlerts = () => (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        {alerts.length === 0 ? (
          <Alert severity="success">
            <AlertTitle>All Systems Operational</AlertTitle>
            No active alerts at this time.
          </Alert>
        ) : (
          alerts.map((alert) => (
            <Alert
              key={alert.id}
              severity={getSeverityColor(alert.severity)}
              sx={{ mb: 2 }}
            >
              <AlertTitle>
                {alert.name} ({alert.count} occurrences)
              </AlertTitle>
              <Typography variant="body2">{alert.message}</Typography>
              <Typography variant="caption" color="textSecondary">
                Started: {new Date(alert.start_time).toLocaleString()}
              </Typography>
            </Alert>
          ))
        )}
      </Grid>
    </Grid>
  );

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
        <Typography variant="h4" gutterBottom>
          System Monitoring
        </Typography>

        <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
          <Tabs value={activeTab} onChange={(_, value) => setActiveTab(value)}>
            <Tab label="Health Status" />
            <Tab label="Metrics" />
            <Tab label="Alerts" />
          </Tabs>
        </Box>

        {activeTab === 0 && renderHealthStatus()}
        {activeTab === 1 && renderMetrics()}
        {activeTab === 2 && renderAlerts()}
      </Box>
    </Container>
  );
};