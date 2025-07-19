import React from 'react';
import {
  Box,
  Typography,
  Chip,
} from '@mui/material';
import { TrendingUp, TrendingDown, TrendingFlat } from '@mui/icons-material';

interface MetricWidgetProps {
  data: unknown;
  config: Record<string, unknown>;
}

export const MetricWidget: React.FC<MetricWidgetProps> = ({ data }) => {
  if (!data) {
    return (
      <Box sx={{ p: 3, textAlign: 'center' }}>
        <Typography color="text.secondary">No data available</Typography>
      </Box>
    );
  }

  // Handle different data formats
  let value: number;
  let label: string;
  let unit: string = '';
  let change: number | undefined;
  let changeLabel: string = '';

  if (typeof data === 'object' && data.value !== undefined) {
    // MetricData format
    value = data.value;
    label = data.label || 'Metric';
    unit = data.unit || '';
    change = data.change;
    changeLabel = data.change_label || '';
  } else if (Array.isArray(data) && data.length > 0) {
    // Array of data - take the first numeric value
    const row = data[0];
    const numericKeys = Object.keys(row).filter(key => typeof row[key] === 'number');
    if (numericKeys.length > 0) {
      value = row[numericKeys[0]];
      label = numericKeys[0];
    } else {
      value = 0;
      label = 'Count';
    }
  } else if (typeof data === 'number') {
    // Simple number
    value = data;
    label = 'Value';
  } else {
    value = 0;
    label = 'No Data';
  }

  const formatValue = (val: number): string => {
    if (val >= 1000000) {
      return `${(val / 1000000).toFixed(1)}M`;
    } else if (val >= 1000) {
      return `${(val / 1000).toFixed(1)}K`;
    }
    return val.toLocaleString();
  };

  const getTrendIcon = (changeValue?: number) => {
    if (changeValue === undefined || changeValue === 0) return <TrendingFlat />;
    if (changeValue > 0) return <TrendingUp />;
    return <TrendingDown />;
  };

  const getTrendColor = (changeValue?: number): string => {
    if (changeValue === undefined || changeValue === 0) return 'default';
    if (changeValue > 0) return 'success';
    return 'error';
  };

  return (
    <Box
      sx={{
        p: 3,
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        alignItems: 'center',
        textAlign: 'center',
      }}
    >
      <Typography variant="h3" component="div" sx={{ fontWeight: 'bold', mb: 1 }}>
        {formatValue(value)}{unit}
      </Typography>
      
      <Typography variant="h6" color="text.secondary" gutterBottom>
        {label}
      </Typography>

      {change !== undefined && (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 2 }}>
          <Chip
            icon={getTrendIcon(change)}
            label={`${change > 0 ? '+' : ''}${change.toFixed(1)}%`}
            color={getTrendColor(change) as any}
            variant="outlined"
            size="small"
          />
          {changeLabel && (
            <Typography variant="caption" color="text.secondary">
              {changeLabel}
            </Typography>
          )}
        </Box>
      )}
    </Box>
  );
};