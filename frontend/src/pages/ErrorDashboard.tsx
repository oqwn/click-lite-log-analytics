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
  Alert,
  AlertTitle,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Divider,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material';
import {
  Error as ErrorIcon,
  Warning,
  TrendingUp,
  TrendingDown,
  TrendingFlat,
  Refresh,
  BugReport,
  Timeline,
} from '@mui/icons-material';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  ArcElement,
} from 'chart.js';
import { Line, Bar, Doughnut } from 'react-chartjs-2';
import api from '../services/api';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  Title,
  Tooltip,
  Legend,
  ArcElement
);

interface ErrorStats {
  pattern: string;
  category: string;
  count: number;
  first_seen: string;
  last_seen: string;
  services: Record<string, number>;
  samples: ErrorSample[];
  rate: number;
  trend: string;
}

interface ErrorSample {
  log_id: string;
  timestamp: string;
  service: string;
  message: string;
  trace_id?: string;
}

interface ErrorAnomaly {
  type: string;
  pattern: string;
  category: string;
  current_rate: number;
  threshold: number;
  severity: string;
  message: string;
}

interface ErrorTrends {
  trends: Record<string, {
    total_errors: number;
    avg_rate: number;
    patterns: number;
    increasing: number;
    decreasing: number;
  }>;
  categories: number;
}

export const ErrorDashboard: React.FC = () => {
  const [errorStats, setErrorStats] = useState<ErrorStats[]>([]);
  const [anomalies, setAnomalies] = useState<ErrorAnomaly[]>([]);
  const [trends, setTrends] = useState<ErrorTrends | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchErrorData = async () => {
    try {
      setLoading(true);
      const [statsRes, anomaliesRes, trendsRes] = await Promise.all([
        api.get<{ stats: ErrorStats[] }>('/errors/stats'),
        api.get<{ anomalies: ErrorAnomaly[] }>('/errors/anomalies'),
        api.get<ErrorTrends>('/errors/trends'),
      ]);

      setErrorStats(statsRes.data.stats || []);
      setAnomalies(anomaliesRes.data.anomalies || []);
      setTrends(trendsRes.data);
    } catch (error) {
      console.error('Failed to fetch error data:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchErrorData();
    const interval = setInterval(fetchErrorData, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);

  const getTrendIcon = (trend: string) => {
    switch (trend) {
      case 'increasing':
        return <TrendingUp color="error" />;
      case 'decreasing':
        return <TrendingDown color="success" />;
      default:
        return <TrendingFlat color="action" />;
    }
  };

  const getSeverityColor = (severity: string): 'error' | 'warning' | 'info' => {
    switch (severity) {
      case 'critical':
        return 'error';
      case 'warning':
        return 'warning';
      default:
        return 'info';
    }
  };

  const getCategoryColor = (category: string) => {
    const colors: Record<string, string> = {
      application: '#f44336',
      database: '#9c27b0',
      http: '#3f51b5',
      network: '#009688',
      system: '#ff9800',
      security: '#f50057',
      generic: '#607d8b',
    };
    return colors[category] || '#757575';
  };

  const renderErrorRateChart = () => {
    if (!errorStats.length) return null;

    const topErrors = errorStats
      .sort((a, b) => b.rate - a.rate)
      .slice(0, 10);

    const data = {
      labels: topErrors.map(e => e.pattern),
      datasets: [
        {
          label: 'Errors per Minute',
          data: topErrors.map(e => e.rate),
          backgroundColor: topErrors.map(e => getCategoryColor(e.category)),
        },
      ],
    };

    const options = {
      responsive: true,
      plugins: {
        legend: {
          display: false,
        },
        title: {
          display: true,
          text: 'Top Error Patterns by Rate',
        },
      },
      scales: {
        x: {
          ticks: {
            maxRotation: 45,
            minRotation: 45,
          },
        },
      },
    };

    return <Bar data={data} options={options} />;
  };

  const renderCategoryDistribution = () => {
    if (!trends) return null;

    const data = {
      labels: Object.keys(trends.trends),
      datasets: [
        {
          data: Object.values(trends.trends).map(t => t.total_errors),
          backgroundColor: Object.keys(trends.trends).map(c => getCategoryColor(c)),
        },
      ],
    };

    const options = {
      responsive: true,
      plugins: {
        legend: {
          position: 'bottom' as const,
        },
        title: {
          display: true,
          text: 'Error Distribution by Category',
        },
      },
    };

    return <Doughnut data={data} options={options} />;
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
            <BugReport sx={{ mr: 1, verticalAlign: 'middle' }} />
            Error Dashboard
          </Typography>
          <IconButton onClick={fetchErrorData}>
            <Refresh />
          </IconButton>
        </Box>

        {/* Anomaly Alerts */}
        {anomalies.length > 0 && (
          <Box mb={3}>
            {anomalies.map((anomaly, index) => (
              <Alert
                key={index}
                severity={getSeverityColor(anomaly.severity)}
                sx={{ mb: 1 }}
              >
                <AlertTitle>
                  {anomaly.type === 'high_error_rate' ? 'High Error Rate' : 'Anomaly Detected'}
                </AlertTitle>
                {anomaly.message}
              </Alert>
            ))}
          </Box>
        )}

        {/* Metrics Cards */}
        <Grid container spacing={3} mb={3}>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Typography color="textSecondary" gutterBottom>
                  Total Errors
                </Typography>
                <Typography variant="h4">
                  {errorStats.reduce((sum, e) => sum + e.count, 0).toLocaleString()}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Typography color="textSecondary" gutterBottom>
                  Error Patterns
                </Typography>
                <Typography variant="h4">{errorStats.length}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Typography color="textSecondary" gutterBottom>
                  Active Anomalies
                </Typography>
                <Typography variant="h4">{anomalies.length}</Typography>
              </CardContent>
            </Card>
          </Grid>
          <Grid item xs={12} sm={6} md={3}>
            <Card>
              <CardContent>
                <Typography color="textSecondary" gutterBottom>
                  Error Categories
                </Typography>
                <Typography variant="h4">{trends?.categories || 0}</Typography>
              </CardContent>
            </Card>
          </Grid>
        </Grid>

        {/* Charts */}
        <Grid container spacing={3}>
          <Grid item xs={12} md={8}>
            <Paper sx={{ p: 2 }}>
              {renderErrorRateChart()}
            </Paper>
          </Grid>
          <Grid item xs={12} md={4}>
            <Paper sx={{ p: 2 }}>
              {renderCategoryDistribution()}
            </Paper>
          </Grid>
        </Grid>

        {/* Error Details Table */}
        <Paper sx={{ mt: 3 }}>
          <Box p={2}>
            <Typography variant="h6" gutterBottom>
              Error Pattern Details
            </Typography>
          </Box>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Pattern</TableCell>
                  <TableCell>Category</TableCell>
                  <TableCell align="right">Count</TableCell>
                  <TableCell align="right">Rate/min</TableCell>
                  <TableCell>Services</TableCell>
                  <TableCell>Trend</TableCell>
                  <TableCell>Last Seen</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {errorStats.map((stat) => (
                  <TableRow key={`${stat.category}:${stat.pattern}`}>
                    <TableCell>{stat.pattern}</TableCell>
                    <TableCell>
                      <Chip
                        label={stat.category}
                        size="small"
                        style={{
                          backgroundColor: getCategoryColor(stat.category),
                          color: 'white',
                        }}
                      />
                    </TableCell>
                    <TableCell align="right">{stat.count.toLocaleString()}</TableCell>
                    <TableCell align="right">{stat.rate.toFixed(2)}</TableCell>
                    <TableCell>
                      {Object.keys(stat.services).slice(0, 3).join(', ')}
                      {Object.keys(stat.services).length > 3 && '...'}
                    </TableCell>
                    <TableCell>{getTrendIcon(stat.trend)}</TableCell>
                    <TableCell>
                      {new Date(stat.last_seen).toLocaleString()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      </Box>
    </Container>
  );
};