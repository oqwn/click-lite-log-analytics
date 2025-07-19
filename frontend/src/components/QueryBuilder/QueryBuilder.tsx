import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  TextField,
  Grid,
  Tabs,
  Tab,
  Alert,
  CircularProgress,
} from '@mui/material';
import { PlayArrow, Save, Download } from '@mui/icons-material';
import { FieldSelector } from './FieldSelector';
import { FilterBuilder } from './FilterBuilder';
import { AggregationBuilder } from './AggregationBuilder';
import { OrderByBuilder } from './OrderByBuilder';
import { TimeRangeSelector } from './TimeRangeSelector';
import { QueryPreview } from './QueryPreview';
import { ResultsTable } from './ResultsTable';
import { queryBuilderApi, handleApiError } from '../../services/api';
import {
  QueryBuilder as QueryBuilderType,
  QueryBuilderResponse,
  QueryField,
  QueryBuilderFilter,
  QueryAggregation,
  QueryOrderBy,
  QueryTimeRange,
} from '../../types/api';

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
      id={`simple-tabpanel-${index}`}
      aria-labelledby={`simple-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

export const QueryBuilder: React.FC = () => {
  const [tabValue, setTabValue] = useState(0);
  const [queryBuilder, setQueryBuilder] = useState<QueryBuilderType>({
    name: '',
    description: '',
    fields: [],
    filters: [],
    aggregations: [],
    group_by: [],
    order_by: [],
    limit: 100,
  });
  const [availableFields, setAvailableFields] = useState<QueryField[]>([]);
  const [queryResult, setQueryResult] = useState<QueryBuilderResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [generatedSQL, setGeneratedSQL] = useState<string>('');

  // Load available fields on component mount
  useEffect(() => {
    loadAvailableFields();
  }, []);

  const loadAvailableFields = async () => {
    try {
      const fieldsData = await queryBuilderApi.getFields();
      setAvailableFields(fieldsData.fields);
    } catch (err) {
      setError(handleApiError(err));
    }
  };

  const handleFieldSelection = (fields: QueryField[]) => {
    setQueryBuilder(prev => ({ ...prev, fields }));
  };

  const handleFiltersChange = (filters: QueryBuilderFilter[]) => {
    setQueryBuilder(prev => ({ ...prev, filters }));
  };

  const handleAggregationsChange = (aggregations: QueryAggregation[]) => {
    setQueryBuilder(prev => ({ ...prev, aggregations }));
  };

  const handleGroupByChange = (groupBy: string[]) => {
    setQueryBuilder(prev => ({ ...prev, group_by: groupBy }));
  };

  const handleOrderByChange = (orderBy: QueryOrderBy[]) => {
    setQueryBuilder(prev => ({ ...prev, order_by: orderBy }));
  };

  const handleTimeRangeChange = (timeRange: QueryTimeRange) => {
    setQueryBuilder(prev => ({ ...prev, time_range: timeRange }));
  };

  const generateSQL = async () => {
    try {
      setLoading(true);
      const sql = await queryBuilderApi.generateSQL(queryBuilder);
      setGeneratedSQL(sql);
      setError(null);
    } catch (err) {
      setError(handleApiError(err));
    } finally {
      setLoading(false);
    }
  };

  const executeQuery = async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await queryBuilderApi.execute(queryBuilder);
      setQueryResult(result);
      setGeneratedSQL(result.sql);
      setTabValue(2); // Switch to results tab
    } catch (err) {
      setError(handleApiError(err));
    } finally {
      setLoading(false);
    }
  };

  const saveQuery = async () => {
    if (!queryBuilder.name.trim()) {
      setError('Please provide a name for the query');
      return;
    }

    try {
      setLoading(true);
      await queryBuilderApi.save(queryBuilder);
      setError(null);
      alert('Query saved successfully!');
    } catch (err) {
      setError(handleApiError(err));
    } finally {
      setLoading(false);
    }
  };

  const exportResults = () => {
    if (!queryResult?.rows?.length) return;

    const csv = [
      // Headers
      queryResult.columns.map(col => col.name).join(','),
      // Data rows
      ...queryResult.rows.map(row =>
        queryResult.columns.map(col => {
          const value = row[col.name];
          return typeof value === 'string' && value.includes(',') 
            ? `"${value}"` 
            : value;
        }).join(',')
      )
    ].join('\n');

    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `query-results-${new Date().toISOString().split('T')[0]}.csv`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  return (
    <Box sx={{ width: '100%' }}>
      <Paper sx={{ p: 3, mb: 3 }}>
        <Typography variant="h4" gutterBottom>
          Query Builder
        </Typography>
        
        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        <Grid container spacing={2} sx={{ mb: 3 }}>
          <Grid item xs={12} md={6}>
            <TextField
              fullWidth
              label="Query Name"
              value={queryBuilder.name}
              onChange={(e) => setQueryBuilder(prev => ({ ...prev, name: e.target.value }))}
              variant="outlined"
            />
          </Grid>
          <Grid item xs={12} md={6}>
            <TextField
              fullWidth
              label="Description"
              value={queryBuilder.description || ''}
              onChange={(e) => setQueryBuilder(prev => ({ ...prev, description: e.target.value }))}
              variant="outlined"
            />
          </Grid>
        </Grid>

        <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
          <Tabs value={tabValue} onChange={(_, newValue) => setTabValue(newValue)}>
            <Tab label="Build Query" />
            <Tab label="SQL Preview" />
            <Tab label="Results" />
          </Tabs>
        </Box>

        <TabPanel value={tabValue} index={0}>
          <Grid container spacing={3}>
            <Grid item xs={12}>
              <FieldSelector
                availableFields={availableFields}
                selectedFields={queryBuilder.fields}
                onChange={handleFieldSelection}
              />
            </Grid>
            
            <Grid item xs={12}>
              <FilterBuilder
                availableFields={availableFields}
                filters={queryBuilder.filters}
                onChange={handleFiltersChange}
              />
            </Grid>
            
            <Grid item xs={12}>
              <AggregationBuilder
                availableFields={availableFields}
                aggregations={queryBuilder.aggregations}
                groupBy={queryBuilder.group_by}
                onAggregationsChange={handleAggregationsChange}
                onGroupByChange={handleGroupByChange}
              />
            </Grid>
            
            <Grid item xs={12}>
              <OrderByBuilder
                availableFields={availableFields}
                orderBy={queryBuilder.order_by}
                onChange={handleOrderByChange}
              />
            </Grid>
            
            <Grid item xs={12}>
              <TimeRangeSelector
                timeRange={queryBuilder.time_range}
                onChange={handleTimeRangeChange}
              />
            </Grid>
            
            <Grid item xs={12}>
              <TextField
                fullWidth
                type="number"
                label="Limit"
                value={queryBuilder.limit || 100}
                onChange={(e) => setQueryBuilder(prev => ({ 
                  ...prev, 
                  limit: parseInt(e.target.value) || 100 
                }))}
                variant="outlined"
                inputProps={{ min: 1, max: 10000 }}
              />
            </Grid>
          </Grid>
        </TabPanel>

        <TabPanel value={tabValue} index={1}>
          <QueryPreview
            queryBuilder={queryBuilder}
            generatedSQL={generatedSQL}
            onGenerate={generateSQL}
            loading={loading}
          />
        </TabPanel>

        <TabPanel value={tabValue} index={2}>
          <ResultsTable
            result={queryResult}
            loading={loading}
            onExport={exportResults}
          />
        </TabPanel>

        <Box sx={{ display: 'flex', gap: 2, mt: 3 }}>
          <Button
            variant="contained"
            startIcon={loading ? <CircularProgress size={20} /> : <PlayArrow />}
            onClick={executeQuery}
            disabled={loading || queryBuilder.fields.length === 0}
          >
            Execute Query
          </Button>
          
          <Button
            variant="outlined"
            startIcon={<Save />}
            onClick={saveQuery}
            disabled={loading || !queryBuilder.name.trim()}
          >
            Save Query
          </Button>
          
          {queryResult?.rows?.length > 0 && (
            <Button
              variant="outlined"
              startIcon={<Download />}
              onClick={exportResults}
            >
              Export CSV
            </Button>
          )}
        </Box>
      </Paper>
    </Box>
  );
};