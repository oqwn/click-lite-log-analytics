import React from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Card,
  CardContent,
  CardActions,
  Button,
  Chip,
} from '@mui/material';
import {
  Dashboard as DashboardIcon,
  Search,
  Timeline,
  TrendingUp,
  Speed,
  Storage,
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';

export const HomePage: React.FC = () => {
  const navigate = useNavigate();

  const features = [
    {
      title: 'Live Log Streaming',
      description: 'Real-time log monitoring with WebSocket connections. Filter and search logs as they come in.',
      icon: Timeline,
      action: () => navigate('/logs'),
      buttonText: 'View Live Logs',
      color: 'primary',
    },
    {
      title: 'Query Builder',
      description: 'Visual query builder for complex log analysis. Create, save, and execute SQL queries with ease.',
      icon: Search,
      action: () => navigate('/query-builder'),
      buttonText: 'Build Queries',
      color: 'secondary',
    },
    {
      title: 'Dashboards',
      description: 'Create customizable dashboards with charts, metrics, and tables. Share insights with your team.',
      icon: DashboardIcon,
      action: () => navigate('/dashboards'),
      buttonText: 'Manage Dashboards',
      color: 'success',
    },
  ];

  const stats = [
    {
      title: 'High Performance',
      description: 'ClickHouse-powered analytics for fast queries on large datasets',
      icon: Speed,
      value: '< 100ms',
      label: 'Query Response Time',
    },
    {
      title: 'Scalable Storage',
      description: 'Columnar storage with compression for efficient log retention',
      icon: Storage,
      value: '90%',
      label: 'Storage Compression',
    },
    {
      title: 'Real-time Insights',
      description: 'Live streaming and instant analytics for operational visibility',
      icon: TrendingUp,
      value: '< 1s',
      label: 'Data Ingestion Latency',
    },
  ];

  return (
    <Box>
      {/* Hero Section */}
      <Paper sx={{ p: 6, mb: 4, textAlign: 'center', bgcolor: 'primary.main', color: 'white' }}>
        <Typography variant="h2" component="h1" gutterBottom>
          Click-Lite Log Analytics
        </Typography>
        <Typography variant="h5" component="h2" sx={{ mb: 3, opacity: 0.9 }}>
          Fast, scalable log analytics platform powered by ClickHouse
        </Typography>
        <Box sx={{ display: 'flex', justifyContent: 'center', gap: 2, flexWrap: 'wrap' }}>
          <Chip label="Real-time Streaming" variant="outlined" sx={{ color: 'white', borderColor: 'white' }} />
          <Chip label="SQL Query Engine" variant="outlined" sx={{ color: 'white', borderColor: 'white' }} />
          <Chip label="Interactive Dashboards" variant="outlined" sx={{ color: 'white', borderColor: 'white' }} />
          <Chip label="High Performance" variant="outlined" sx={{ color: 'white', borderColor: 'white' }} />
        </Box>
      </Paper>

      {/* Features Grid */}
      <Typography variant="h4" component="h2" gutterBottom sx={{ mb: 3 }}>
        Features
      </Typography>
      
      <Grid container spacing={3} sx={{ mb: 6 }}>
        {features.map((feature, index) => {
          const Icon = feature.icon;
          return (
            <Grid item xs={12} md={4} key={index}>
              <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
                <CardContent sx={{ flexGrow: 1 }}>
                  <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                    <Icon color={feature.color as any} sx={{ fontSize: 32, mr: 1 }} />
                    <Typography variant="h6" component="h3">
                      {feature.title}
                    </Typography>
                  </Box>
                  <Typography color="text.secondary">
                    {feature.description}
                  </Typography>
                </CardContent>
                <CardActions>
                  <Button
                    size="small"
                    color={feature.color as any}
                    onClick={feature.action}
                  >
                    {feature.buttonText}
                  </Button>
                </CardActions>
              </Card>
            </Grid>
          );
        })}
      </Grid>

      {/* Stats Section */}
      <Typography variant="h4" component="h2" gutterBottom sx={{ mb: 3 }}>
        Platform Capabilities
      </Typography>
      
      <Grid container spacing={3}>
        {stats.map((stat, index) => {
          const Icon = stat.icon;
          return (
            <Grid item xs={12} md={4} key={index}>
              <Paper sx={{ p: 3, textAlign: 'center' }}>
                <Icon color="primary" sx={{ fontSize: 48, mb: 2 }} />
                <Typography variant="h4" component="div" color="primary" gutterBottom>
                  {stat.value}
                </Typography>
                <Typography variant="h6" component="h3" gutterBottom>
                  {stat.title}
                </Typography>
                <Typography variant="body2" color="text.secondary" gutterBottom>
                  {stat.label}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {stat.description}
                </Typography>
              </Paper>
            </Grid>
          );
        })}
      </Grid>

      {/* Getting Started */}
      <Paper sx={{ p: 4, mt: 6, textAlign: 'center' }}>
        <Typography variant="h5" component="h2" gutterBottom>
          Ready to get started?
        </Typography>
        <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
          Explore your log data with powerful analytics tools
        </Typography>
        <Box sx={{ display: 'flex', justifyContent: 'center', gap: 2, flexWrap: 'wrap' }}>
          <Button
            variant="contained"
            size="large"
            startIcon={<Timeline />}
            onClick={() => navigate('/logs')}
          >
            Start Live Monitoring
          </Button>
          <Button
            variant="outlined"
            size="large"
            startIcon={<Search />}
            onClick={() => navigate('/query-builder')}
          >
            Build Your First Query
          </Button>
        </Box>
      </Paper>
    </Box>
  );
};