import React from 'react';
import {
  Box,
  Typography,
  Paper,
  Button,
  IconButton,
  Grid,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  TextField,
  Chip,
} from '@mui/material';
import { Add, Delete } from '@mui/icons-material';
import { QueryField, QueryBuilderFilter, FILTER_OPERATORS } from '../../types/api';

interface FilterBuilderProps {
  availableFields: QueryField[];
  filters: QueryBuilderFilter[];
  onChange: (filters: QueryBuilderFilter[]) => void;
}

export const FilterBuilder: React.FC<FilterBuilderProps> = ({
  availableFields,
  filters,
  onChange,
}) => {
  const addFilter = () => {
    const newFilter: QueryBuilderFilter = {
      id: `filter-${Date.now()}`,
      field: '',
      operator: 'equals',
      value: '',
      logical_op: filters.length > 0 ? 'AND' : undefined,
    };
    onChange([...filters, newFilter]);
  };

  const updateFilter = (index: number, updates: Partial<QueryBuilderFilter>) => {
    const updatedFilters = filters.map((filter, i) => 
      i === index ? { ...filter, ...updates } : filter
    );
    onChange(updatedFilters);
  };

  const removeFilter = (index: number) => {
    const updatedFilters = filters.filter((_, i) => i !== index);
    // Remove logical operator from first filter if it exists
    if (updatedFilters.length > 0 && updatedFilters[0].logical_op) {
      updatedFilters[0] = { ...updatedFilters[0], logical_op: undefined };
    }
    onChange(updatedFilters);
  };

  const getFieldType = (fieldName: string): string => {
    const field = availableFields.find(f => f.name === fieldName);
    return field?.type || 'string';
  };

  const renderValueInput = (filter: QueryBuilderFilter, index: number) => {
    const fieldType = getFieldType(filter.field);
    const needsMultipleValues = ['in', 'not_in', 'between'].includes(filter.operator);
    const needsNoValue = ['is_null', 'is_not_null'].includes(filter.operator);

    if (needsNoValue) {
      return null;
    }

    if (needsMultipleValues) {
      return (
        <TextField
          fullWidth
          label={filter.operator === 'between' ? 'Values (min, max)' : 'Values (comma-separated)'}
          value={Array.isArray(filter.values) ? filter.values.join(', ') : filter.value}
          onChange={(e) => {
            const values = e.target.value.split(',').map(v => v.trim());
            updateFilter(index, { values, value: e.target.value });
          }}
          placeholder={filter.operator === 'between' ? '10, 100' : 'value1, value2, value3'}
        />
      );
    }

    if (fieldType === 'boolean') {
      return (
        <FormControl fullWidth>
          <InputLabel>Value</InputLabel>
          <Select
            value={filter.value}
            onChange={(e) => updateFilter(index, { value: e.target.value })}
            label="Value"
          >
            <MenuItem value="true">True</MenuItem>
            <MenuItem value="false">False</MenuItem>
          </Select>
        </FormControl>
      );
    }

    if (fieldType === 'datetime') {
      return (
        <TextField
          fullWidth
          type="datetime-local"
          label="Value"
          value={filter.value}
          onChange={(e) => updateFilter(index, { value: e.target.value })}
          InputLabelProps={{ shrink: true }}
        />
      );
    }

    return (
      <TextField
        fullWidth
        label="Value"
        type={fieldType === 'number' ? 'number' : 'text'}
        value={filter.value}
        onChange={(e) => updateFilter(index, { value: e.target.value })}
      />
    );
  };

  return (
    <Paper sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">Filters</Typography>
        <Button
          variant="outlined"
          startIcon={<Add />}
          onClick={addFilter}
          size="small"
        >
          Add Filter
        </Button>
      </Box>

      {filters.length === 0 ? (
        <Typography color="text.secondary" align="center">
          No filters applied. Click "Add Filter" to create your first filter.
        </Typography>
      ) : (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {filters.map((filter, index) => (
            <Paper key={filter.id} variant="outlined" sx={{ p: 2 }}>
              <Grid container spacing={2} alignItems="center">
                {index > 0 && (
                  <Grid item xs={12} sm={2}>
                    <FormControl fullWidth size="small">
                      <InputLabel>Logic</InputLabel>
                      <Select
                        value={filter.logical_op || 'AND'}
                        onChange={(e) => updateFilter(index, { logical_op: e.target.value })}
                        label="Logic"
                      >
                        <MenuItem value="AND">AND</MenuItem>
                        <MenuItem value="OR">OR</MenuItem>
                      </Select>
                    </FormControl>
                  </Grid>
                )}
                
                <Grid item xs={12} sm={index > 0 ? 3 : 4}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Field</InputLabel>
                    <Select
                      value={filter.field}
                      onChange={(e) => updateFilter(index, { field: e.target.value })}
                      label="Field"
                    >
                      {availableFields.map((field) => (
                        <MenuItem key={field.name} value={field.name}>
                          {field.label || field.name}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                
                <Grid item xs={12} sm={3}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Operator</InputLabel>
                    <Select
                      value={filter.operator}
                      onChange={(e) => updateFilter(index, { operator: e.target.value })}
                      label="Operator"
                    >
                      {FILTER_OPERATORS.map((op) => (
                        <MenuItem key={op.value} value={op.value}>
                          {op.label}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                
                <Grid item xs={12} sm={index > 0 ? 3 : 4}>
                  {renderValueInput(filter, index)}
                </Grid>
                
                <Grid item xs={12} sm={1}>
                  <IconButton
                    color="error"
                    onClick={() => removeFilter(index)}
                    size="small"
                  >
                    <Delete />
                  </IconButton>
                </Grid>
              </Grid>
            </Paper>
          ))}
        </Box>
      )}

      {filters.length > 0 && (
        <Box sx={{ mt: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            Filter Summary:
          </Typography>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
            {filters.map((filter, index) => (
              <Chip
                key={filter.id}
                label={`${index > 0 ? `${filter.logical_op} ` : ''}${filter.field} ${filter.operator} ${
                  ['is_null', 'is_not_null'].includes(filter.operator) ? '' : filter.value
                }`}
                variant="outlined"
                size="small"
              />
            ))}
          </Box>
        </Box>
      )}
    </Paper>
  );
};