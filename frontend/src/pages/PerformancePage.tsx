import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Card,
  CardContent,
  Button,
  TextField,
  Chip,
  Alert,
  CircularProgress,
  Tabs,
  Tab,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  IconButton,
} from '@mui/material';
import {
  Speed as SpeedIcon,
  Cached as CacheIcon,
  Storage as StorageIcon,
  GroupWork as ClusterIcon,
  Refresh as RefreshIcon,
  Delete as DeleteIcon,
  TrendingUp as TrendingUpIcon,
  Assessment as AssessmentIcon,
} from '@mui/icons-material';
import { ResponsiveContainer, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip } from 'recharts';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`performance-tabpanel-${index}`}
      aria-labelledby={`performance-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

const PerformancePage: React.FC = () => {
  const [tabValue, setTabValue] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Query Optimization State
  const [queryText, setQueryText] = useState('SELECT * FROM logs WHERE timestamp > now() - INTERVAL 1 DAY');
  const [optimizationResult, setOptimizationResult] = useState<any>(null);
  
  // Cache State
  const [cacheStats, setCacheStats] = useState<any>(null);
  
  // Storage State
  const [storageAnalysis, setStorageAnalysis] = useState<any>(null);
  const [storageTableName, setStorageTableName] = useState('logs');
  
  // Cluster State
  const [clusterStatus, setClusterStatus] = useState<any>(null);
  
  // Performance Metrics
  const [performanceMetrics, setPerformanceMetrics] = useState<any>(null);
  
  // Benchmark State
  const [benchmarkDialog, setBenchmarkDialog] = useState(false);
  const [benchmarkQuery, setBenchmarkQuery] = useState('');
  const [benchmarkResult, setBenchmarkResult] = useState<any>(null);

  const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:20002/api/v1';

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  const fetchCacheStats = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/performance/cache/stats`);
      if (!response.ok) throw new Error('Failed to fetch cache stats');
      const data = await response.json();
      setCacheStats(data);
    } catch (err) {
      setError('Failed to fetch cache statistics');
      console.error(err);
    }
  };

  const fetchPerformanceMetrics = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/performance/metrics`);
      if (!response.ok) throw new Error('Failed to fetch performance metrics');
      const data = await response.json();
      setPerformanceMetrics(data);
    } catch (err) {
      setError('Failed to fetch performance metrics');
      console.error(err);
    }
  };

  const fetchClusterStatus = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/performance/cluster/status`);
      if (!response.ok) throw new Error('Failed to fetch cluster status');
      const data = await response.json();
      setClusterStatus(data);
    } catch (err) {
      setError('Failed to fetch cluster status');
      console.error(err);
    }
  };

  const optimizeQuery = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE_URL}/performance/optimize-query`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: queryText }),
      });
      if (!response.ok) throw new Error('Failed to optimize query');
      const data = await response.json();
      setOptimizationResult(data);
    } catch (err) {
      setError('Failed to optimize query');
      console.error(err);
    }
    setLoading(false);
  };

  const analyzeStorage = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE_URL}/performance/storage/analyze`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ table_name: storageTableName }),
      });
      if (!response.ok) throw new Error('Failed to analyze storage');
      const data = await response.json();
      setStorageAnalysis(data);
    } catch (err) {
      setError('Failed to analyze storage');
      console.error(err);
    }
    setLoading(false);
  };

  const clearCache = async () => {
    setLoading(true);
    try {
      const response = await fetch(`${API_BASE_URL}/performance/cache`, {
        method: 'DELETE',
      });
      if (!response.ok) throw new Error('Failed to clear cache');
      await fetchCacheStats();
    } catch (err) {
      setError('Failed to clear cache');
      console.error(err);
    }
    setLoading(false);
  };

  const runBenchmark = async () => {
    setLoading(true);
    try {
      const response = await fetch(`${API_BASE_URL}/performance/benchmark-query`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          query: benchmarkQuery,
          iterations: 10,
          use_cache: true
        }),
      });
      if (!response.ok) throw new Error('Failed to run benchmark');
      const data = await response.json();
      setBenchmarkResult(data);
    } catch (err) {
      setError('Failed to run benchmark');
      console.error(err);
    }
    setLoading(false);
  };

  useEffect(() => {
    fetchCacheStats();
    fetchPerformanceMetrics();
    fetchClusterStatus();
  }, []);

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  return (
    <Box sx={{ padding: 3 }}>
      <Typography variant="h4" gutterBottom sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <SpeedIcon /> Performance Optimization
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
        <Tabs value={tabValue} onChange={handleTabChange}>
          <Tab label="Query Optimization" icon={<AssessmentIcon />} />
          <Tab label="Cache Management" icon={<CacheIcon />} />
          <Tab label="Storage Analysis" icon={<StorageIcon />} />
          <Tab label="Cluster Status" icon={<ClusterIcon />} />
          <Tab label="Performance Metrics" icon={<TrendingUpIcon />} />
        </Tabs>
      </Box>

      {/* Query Optimization Tab */}
      <TabPanel value={tabValue} index={0}>
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                SQL Query Optimizer
              </Typography>
              <TextField
                fullWidth
                multiline
                rows={6}
                value={queryText}
                onChange={(e) => setQueryText(e.target.value)}
                placeholder="Enter your SQL query here..."
                sx={{ mb: 2 }}
              />
              <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
                <Button
                  variant="contained"
                  onClick={optimizeQuery}
                  disabled={loading}
                  startIcon={loading ? <CircularProgress size={20} /> : <SpeedIcon />}
                >
                  Optimize Query
                </Button>
                <Button
                  variant="outlined"
                  onClick={() => setBenchmarkDialog(true)}
                  startIcon={<AssessmentIcon />}
                >
                  Benchmark Query
                </Button>
              </Box>
              
              {optimizationResult && (
                <Card sx={{ mt: 2 }}>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>
                      Optimization Results
                    </Typography>
                    <Grid container spacing={2}>
                      <Grid item xs={12} md={6}>
                        <Typography variant="subtitle2">Original Query:</Typography>
                        <Box sx={{ bgcolor: 'grey.100', p: 2, borderRadius: 1, mb: 2, fontFamily: 'monospace' }}>
                          {optimizationResult.original_query}
                        </Box>
                      </Grid>
                      <Grid item xs={12} md={6}>
                        <Typography variant="subtitle2">Optimized Query:</Typography>
                        <Box sx={{ bgcolor: 'success.50', p: 2, borderRadius: 1, mb: 2, fontFamily: 'monospace' }}>
                          {optimizationResult.optimized_query}
                        </Box>
                      </Grid>
                    </Grid>
                    
                    <Typography variant="subtitle2" gutterBottom>
                      Applied Optimizations:
                    </Typography>
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 2 }}>
                      {optimizationResult.optimizations?.map((opt: string, index: number) => (
                        <Chip key={index} label={opt} size="small" color="primary" />
                      ))}
                    </Box>
                    
                    <Grid container spacing={2}>
                      <Grid item xs={6} md={3}>
                        <Typography variant="body2" color="text.secondary">
                          Estimated Cost: {optimizationResult.estimated_cost?.toFixed(2)}
                        </Typography>
                      </Grid>
                      <Grid item xs={6} md={3}>
                        <Typography variant="body2" color="text.secondary">
                          Parallelism: {optimizationResult.parallelism}
                        </Typography>
                      </Grid>
                      <Grid item xs={6} md={3}>
                        <Typography variant="body2" color="text.secondary">
                          Partition Pruning: {optimizationResult.partition_pruning ? 'Yes' : 'No'}
                        </Typography>
                      </Grid>
                      <Grid item xs={6} md={3}>
                        <Typography variant="body2" color="text.secondary">
                          Indexes: {optimizationResult.indexes_used?.length || 0}
                        </Typography>
                      </Grid>
                    </Grid>
                  </CardContent>
                </Card>
              )}
            </Paper>
          </Grid>
        </Grid>
      </TabPanel>

      {/* Cache Management Tab */}
      <TabPanel value={tabValue} index={1}>
        <Grid container spacing={3}>
          <Grid item xs={12} md={6}>
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Typography variant="h6">Cache Statistics</Typography>
                  <Box>
                    <IconButton onClick={fetchCacheStats} size="small">
                      <RefreshIcon />
                    </IconButton>
                    <IconButton onClick={clearCache} size="small" color="error">
                      <DeleteIcon />
                    </IconButton>
                  </Box>
                </Box>
                
                {cacheStats ? (
                  <Grid container spacing={2}>
                    <Grid item xs={6}>
                      <Typography variant="body2" color="text.secondary">
                        Cache Hits: {cacheStats.hits?.toLocaleString() || 0}
                      </Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Typography variant="body2" color="text.secondary">
                        Cache Misses: {cacheStats.misses?.toLocaleString() || 0}
                      </Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Typography variant="body2" color="text.secondary">
                        Hit Rate: {((cacheStats.hit_rate || 0) * 100).toFixed(1)}%
                      </Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Typography variant="body2" color="text.secondary">
                        Cache Size: {cacheStats.size || 0} / {cacheStats.max_size || 0}
                      </Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Typography variant="body2" color="text.secondary">
                        Evictions: {cacheStats.evictions?.toLocaleString() || 0}
                      </Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Typography variant="body2" color="text.secondary">
                        Last Reset: {cacheStats.last_reset ? new Date(cacheStats.last_reset).toLocaleString() : 'N/A'}
                      </Typography>
                    </Grid>
                  </Grid>
                ) : (
                  <CircularProgress />
                )}
              </CardContent>
            </Card>
          </Grid>
          
          <Grid item xs={12} md={6}>
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Cache Performance
                </Typography>
                {cacheStats && (
                  <Box sx={{ width: '100%', height: 200 }}>
                    <ResponsiveContainer>
                      <BarChart data={[
                        { name: 'Hits', value: cacheStats.hits || 0 },
                        { name: 'Misses', value: cacheStats.misses || 0 },
                      ]}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="name" />
                        <YAxis />
                        <Tooltip />
                        <Bar dataKey="value" fill="#8884d8" />
                      </BarChart>
                    </ResponsiveContainer>
                  </Box>
                )}
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      </TabPanel>

      {/* Storage Analysis Tab */}
      <TabPanel value={tabValue} index={2}>
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Paper sx={{ p: 3 }}>
              <Typography variant="h6" gutterBottom>
                Storage Analysis
              </Typography>
              <Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
                <TextField
                  label="Table Name"
                  value={storageTableName}
                  onChange={(e) => setStorageTableName(e.target.value)}
                  size="small"
                />
                <Button
                  variant="contained"
                  onClick={analyzeStorage}
                  disabled={loading}
                  startIcon={loading ? <CircularProgress size={20} /> : <StorageIcon />}
                >
                  Analyze Storage
                </Button>
              </Box>
              
              {storageAnalysis && (
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>
                      Analysis Results for {storageAnalysis.table_name}
                    </Typography>
                    
                    <Grid container spacing={2}>
                      <Grid item xs={6} md={3}>
                        <Typography variant="body2" color="text.secondary">
                          Total Rows: {storageAnalysis.total_rows?.toLocaleString() || 0}
                        </Typography>
                      </Grid>
                      <Grid item xs={6} md={3}>
                        <Typography variant="body2" color="text.secondary">
                          Total Size: {formatBytes(storageAnalysis.total_bytes || 0)}
                        </Typography>
                      </Grid>
                      <Grid item xs={6} md={3}>
                        <Typography variant="body2" color="text.secondary">
                          Compressed: {formatBytes(storageAnalysis.compressed_bytes || 0)}
                        </Typography>
                      </Grid>
                      <Grid item xs={6} md={3}>
                        <Typography variant="body2" color="text.secondary">
                          Compression Ratio: {(storageAnalysis.compression_ratio || 0).toFixed(2)}x
                        </Typography>
                      </Grid>
                      <Grid item xs={6} md={3}>
                        <Typography variant="body2" color="text.secondary">
                          Total Parts: {storageAnalysis.total_parts || 0}
                        </Typography>
                      </Grid>
                      <Grid item xs={6} md={3}>
                        <Typography variant="body2" color="text.secondary">
                          Analysis Time: {storageAnalysis.analysis_time ? new Date(storageAnalysis.analysis_time).toLocaleString() : 'N/A'}
                        </Typography>
                      </Grid>
                    </Grid>
                    
                    {storageAnalysis.recommendations && storageAnalysis.recommendations.length > 0 && (
                      <Box sx={{ mt: 2 }}>
                        <Typography variant="subtitle2" gutterBottom>
                          Recommendations:
                        </Typography>
                        {storageAnalysis.recommendations.map((rec: string, index: number) => (
                          <Alert key={index} severity="info" sx={{ mb: 1 }}>
                            {rec}
                          </Alert>
                        ))}
                      </Box>
                    )}
                  </CardContent>
                </Card>
              )}
            </Paper>
          </Grid>
        </Grid>
      </TabPanel>

      {/* Cluster Status Tab */}
      <TabPanel value={tabValue} index={3}>
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Typography variant="h6">Cluster Status</Typography>
                  <IconButton onClick={fetchClusterStatus} size="small">
                    <RefreshIcon />
                  </IconButton>
                </Box>
                
                {clusterStatus ? (
                  <Grid container spacing={2}>
                    <Grid item xs={12} md={4}>
                      <Typography variant="body2" color="text.secondary">
                        Status: 
                        <Chip 
                          label={clusterStatus.status} 
                          color={clusterStatus.status === 'healthy' ? 'success' : 'error'} 
                          size="small" 
                          sx={{ ml: 1 }}
                        />
                      </Typography>
                    </Grid>
                    <Grid item xs={12} md={4}>
                      <Typography variant="body2" color="text.secondary">
                        Active Nodes: {clusterStatus.nodes || 0}
                      </Typography>
                    </Grid>
                    <Grid item xs={12} md={4}>
                      <Typography variant="body2" color="text.secondary">
                        Load Balancing: Active
                      </Typography>
                    </Grid>
                  </Grid>
                ) : (
                  <CircularProgress />
                )}
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      </TabPanel>

      {/* Performance Metrics Tab */}
      <TabPanel value={tabValue} index={4}>
        <Grid container spacing={3}>
          <Grid item xs={12}>
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Typography variant="h6">Performance Overview</Typography>
                  <IconButton onClick={fetchPerformanceMetrics} size="small">
                    <RefreshIcon />
                  </IconButton>
                </Box>
                
                {performanceMetrics ? (
                  <Grid container spacing={2}>
                    <Grid item xs={12} md={3}>
                      <Paper sx={{ p: 2, textAlign: 'center' }}>
                        <CacheIcon fontSize="large" color="primary" />
                        <Typography variant="h6">
                          {((performanceMetrics.cache?.hit_rate || 0) * 100).toFixed(1)}%
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                          Cache Hit Rate
                        </Typography>
                      </Paper>
                    </Grid>
                    
                    <Grid item xs={12} md={3}>
                      <Paper sx={{ p: 2, textAlign: 'center' }}>
                        <SpeedIcon fontSize="large" color="success" />
                        <Typography variant="h6">
                          {performanceMetrics.query_optimization?.enabled ? 'Active' : 'Inactive'}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                          Query Optimization
                        </Typography>
                      </Paper>
                    </Grid>
                    
                    <Grid item xs={12} md={3}>
                      <Paper sx={{ p: 2, textAlign: 'center' }}>
                        <StorageIcon fontSize="large" color="warning" />
                        <Typography variant="h6">
                          {performanceMetrics.storage?.optimization_enabled ? 'Enabled' : 'Disabled'}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                          Storage Optimization
                        </Typography>
                      </Paper>
                    </Grid>
                    
                    <Grid item xs={12} md={3}>
                      <Paper sx={{ p: 2, textAlign: 'center' }}>
                        <ClusterIcon fontSize="large" color="info" />
                        <Typography variant="h6">
                          {performanceMetrics.cluster?.nodes_count || 0}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                          Cluster Nodes
                        </Typography>
                      </Paper>
                    </Grid>
                  </Grid>
                ) : (
                  <CircularProgress />
                )}
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      </TabPanel>

      {/* Benchmark Dialog */}
      <Dialog open={benchmarkDialog} onClose={() => setBenchmarkDialog(false)} maxWidth="md" fullWidth>
        <DialogTitle>Query Benchmark</DialogTitle>
        <DialogContent>
          <TextField
            fullWidth
            multiline
            rows={4}
            value={benchmarkQuery}
            onChange={(e) => setBenchmarkQuery(e.target.value)}
            placeholder="Enter query to benchmark..."
            sx={{ mt: 1, mb: 2 }}
          />
          
          {benchmarkResult && (
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Benchmark Results
                </Typography>
                <Grid container spacing={2}>
                  <Grid item xs={6}>
                    <Typography variant="body2">
                      Average Time: {benchmarkResult.average_execution_time_ms?.toFixed(2)}ms
                    </Typography>
                  </Grid>
                  <Grid item xs={6}>
                    <Typography variant="body2">
                      Min Time: {benchmarkResult.min_execution_time_ms?.toFixed(2)}ms
                    </Typography>
                  </Grid>
                  <Grid item xs={6}>
                    <Typography variant="body2">
                      Max Time: {benchmarkResult.max_execution_time_ms?.toFixed(2)}ms
                    </Typography>
                  </Grid>
                  <Grid item xs={6}>
                    <Typography variant="body2">
                      Iterations: {benchmarkResult.iterations}
                    </Typography>
                  </Grid>
                </Grid>
              </CardContent>
            </Card>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setBenchmarkDialog(false)}>Cancel</Button>
          <Button onClick={runBenchmark} variant="contained" disabled={loading || !benchmarkQuery}>
            Run Benchmark
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default PerformancePage;