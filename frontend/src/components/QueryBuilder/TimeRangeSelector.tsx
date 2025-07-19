import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  TextField,
  Grid,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material';
import { QueryTimeRange, TIME_RANGES } from '../../types/api';

interface TimeRangeSelectorProps {
  timeRange?: QueryTimeRange;
  onChange: (timeRange: QueryTimeRange) => void;
}

export const TimeRangeSelector: React.FC<TimeRangeSelectorProps> = ({
  timeRange,
  onChange,
}) => {
  const [mode, setMode] = useState<'relative' | 'absolute'>(
    timeRange?.relative ? 'relative' : 'absolute'
  );

  const handleModeChange = (newMode: 'relative' | 'absolute') => {
    setMode(newMode);
    if (newMode === 'relative') {
      onChange({
        relative: 'last_1h',
        start: undefined,
        end: undefined,
      });
    } else {
      onChange({
        relative: undefined,
        start: '',
        end: '',
      });
    }
  };

  const handleRelativeChange = (relative: string) => {
    onChange({
      ...timeRange,
      relative,
      start: undefined,
      end: undefined,
    });
  };

  const handleAbsoluteChange = (field: 'start' | 'end', value: string) => {
    onChange({
      ...timeRange,
      relative: undefined,
      [field]: value,
    });
  };

  const formatDateTimeLocal = (date: Date): string => {
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${year}-${month}-${day}T${hours}:${minutes}`;
  };

  const getDefaultEndTime = (): string => {
    return formatDateTimeLocal(new Date());
  };

  const getDefaultStartTime = (): string => {
    const date = new Date();
    date.setHours(date.getHours() - 1); // 1 hour ago
    return formatDateTimeLocal(date);
  };

  return (
    <Paper sx={{ p: 3 }}>
      <Typography variant="h6" gutterBottom>
        Time Range
      </Typography>

      <Box sx={{ mb: 3 }}>
        <ToggleButtonGroup
          value={mode}
          exclusive
          onChange={(_, newMode) => newMode && handleModeChange(newMode)}
          aria-label="time range mode"
        >
          <ToggleButton value="relative" aria-label="relative time">
            Relative Time
          </ToggleButton>
          <ToggleButton value="absolute" aria-label="absolute time">
            Absolute Time
          </ToggleButton>
        </ToggleButtonGroup>
      </Box>

      {mode === 'relative' ? (
        <FormControl fullWidth>
          <InputLabel>Time Range</InputLabel>
          <Select
            value={timeRange?.relative || 'last_1h'}
            onChange={(e) => handleRelativeChange(e.target.value)}
            label="Time Range"
          >
            {TIME_RANGES.map((range) => (
              <MenuItem key={range.value} value={range.value}>
                {range.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      ) : (
        <Grid container spacing={2}>
          <Grid item xs={12} sm={6}>
            <TextField
              fullWidth
              type="datetime-local"
              label="Start Time"
              value={timeRange?.start || getDefaultStartTime()}
              onChange={(e) => handleAbsoluteChange('start', e.target.value)}
              InputLabelProps={{ shrink: true }}
            />
          </Grid>
          <Grid item xs={12} sm={6}>
            <TextField
              fullWidth
              type="datetime-local"
              label="End Time"
              value={timeRange?.end || getDefaultEndTime()}
              onChange={(e) => handleAbsoluteChange('end', e.target.value)}
              InputLabelProps={{ shrink: true }}
            />
          </Grid>
        </Grid>
      )}

      <Box sx={{ mt: 2, p: 2, bgcolor: 'grey.50', borderRadius: 1 }}>
        <Typography variant="caption" color="text.secondary">
          <strong>Time Filter:</strong> {
            mode === 'relative' 
              ? `Showing data from ${TIME_RANGES.find(r => r.value === timeRange?.relative)?.label || 'last 1 hour'}`
              : `Showing data from ${timeRange?.start || 'start time'} to ${timeRange?.end || 'end time'}`
          }
        </Typography>
      </Box>
    </Paper>
  );
};