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
  FormControlLabel,
  Checkbox,
} from '@mui/material';
import { Add, Delete } from '@mui/icons-material';
import { QueryField, QueryAggregation, AGGREGATION_FUNCTIONS } from '../../types/api';

interface AggregationBuilderProps {
  availableFields: QueryField[];
  aggregations: QueryAggregation[];
  groupBy: string[];
  onAggregationsChange: (aggregations: QueryAggregation[]) => void;
  onGroupByChange: (groupBy: string[]) => void;
}

export const AggregationBuilder: React.FC<AggregationBuilderProps> = ({
  availableFields,
  aggregations,
  groupBy,
  onAggregationsChange,
  onGroupByChange,
}) => {
  const addAggregation = () => {
    const newAggregation: QueryAggregation = {
      id: `agg-${Date.now()}`,
      function: 'COUNT',
      field: undefined,
      alias: '',
    };
    onAggregationsChange([...aggregations, newAggregation]);
  };

  const updateAggregation = (index: number, updates: Partial<QueryAggregation>) => {
    const updatedAggregations = aggregations.map((agg, i) => 
      i === index ? { ...agg, ...updates } : agg
    );
    onAggregationsChange(updatedAggregations);
  };

  const removeAggregation = (index: number) => {
    const updatedAggregations = aggregations.filter((_, i) => i !== index);
    onAggregationsChange(updatedAggregations);
  };

  const handleGroupByToggle = (fieldName: string) => {
    if (groupBy.includes(fieldName)) {
      onGroupByChange(groupBy.filter(field => field !== fieldName));
    } else {
      onGroupByChange([...groupBy, fieldName]);
    }
  };

  const needsField = (functionName: string): boolean => {
    return !['COUNT'].includes(functionName);
  };

  const getNumericFields = (): QueryField[] => {
    return availableFields.filter(field => field.type === 'number');
  };

  const getGroupableFields = (): QueryField[] => {
    return availableFields.filter(field => ['string', 'datetime', 'boolean'].includes(field.type));
  };

  return (
    <Paper sx={{ p: 3 }}>
      <Typography variant="h6" gutterBottom>
        Aggregations & Grouping
      </Typography>

      {/* Group By Section */}
      <Box sx={{ mb: 3 }}>
        <Typography variant="subtitle1" gutterBottom>
          Group By Fields
        </Typography>
        <Grid container spacing={2}>
          {getGroupableFields().map((field) => (
            <Grid item xs={12} sm={6} md={4} key={field.name}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={groupBy.includes(field.name)}
                    onChange={() => handleGroupByToggle(field.name)}
                  />
                }
                label={
                  <Box>
                    <Typography variant="body2">{field.label || field.name}</Typography>
                    <Typography variant="caption" color="text.secondary">
                      {field.type}
                    </Typography>
                  </Box>
                }
              />
            </Grid>
          ))}
        </Grid>

        {groupBy.length > 0 && (
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle2" gutterBottom>
              Grouping by:
            </Typography>
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
              {groupBy.map((fieldName) => {
                const field = availableFields.find(f => f.name === fieldName);
                return (
                  <Chip
                    key={fieldName}
                    label={field?.label || fieldName}
                    variant="outlined"
                    onDelete={() => handleGroupByToggle(fieldName)}
                    size="small"
                  />
                );
              })}
            </Box>
          </Box>
        )}
      </Box>

      {/* Aggregations Section */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="subtitle1">Aggregation Functions</Typography>
        <Button
          variant="outlined"
          startIcon={<Add />}
          onClick={addAggregation}
          size="small"
        >
          Add Aggregation
        </Button>
      </Box>

      {aggregations.length === 0 ? (
        <Typography color="text.secondary" align="center">
          No aggregations defined. Add aggregations to calculate metrics like counts, sums, averages, etc.
        </Typography>
      ) : (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {aggregations.map((aggregation, index) => (
            <Paper key={aggregation.id} variant="outlined" sx={{ p: 2 }}>
              <Grid container spacing={2} alignItems="center">
                <Grid item xs={12} sm={3}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Function</InputLabel>
                    <Select
                      value={aggregation.function}
                      onChange={(e) => {
                        const func = e.target.value;
                        updateAggregation(index, { 
                          function: func,
                          field: needsField(func) ? aggregation.field : undefined 
                        });
                      }}
                      label="Function"
                    >
                      {AGGREGATION_FUNCTIONS.map((func) => (
                        <MenuItem key={func.value} value={func.value}>
                          {func.label}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                
                {needsField(aggregation.function) && (
                  <Grid item xs={12} sm={3}>
                    <FormControl fullWidth size="small">
                      <InputLabel>Field</InputLabel>
                      <Select
                        value={aggregation.field || ''}
                        onChange={(e) => updateAggregation(index, { field: e.target.value })}
                        label="Field"
                      >
                        {(aggregation.function === 'COUNT_DISTINCT' ? availableFields : getNumericFields()).map((field) => (
                          <MenuItem key={field.name} value={field.name}>
                            {field.label || field.name}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                  </Grid>
                )}
                
                <Grid item xs={12} sm={needsField(aggregation.function) ? 4 : 7}>
                  <TextField
                    fullWidth
                    size="small"
                    label="Alias (optional)"
                    value={aggregation.alias || ''}
                    onChange={(e) => updateAggregation(index, { alias: e.target.value })}
                    placeholder={`${aggregation.function.toLowerCase()}${aggregation.field ? `_${aggregation.field}` : ''}`}
                  />
                </Grid>
                
                <Grid item xs={12} sm={1}>
                  <IconButton
                    color="error"
                    onClick={() => removeAggregation(index)}
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

      {aggregations.length > 0 && (
        <Box sx={{ mt: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            Aggregation Summary:
          </Typography>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
            {aggregations.map((agg) => (
              <Chip
                key={agg.id}
                label={`${agg.function}${agg.field ? `(${agg.field})` : '()'} ${agg.alias ? `as ${agg.alias}` : ''}`}
                variant="outlined"
                size="small"
              />
            ))}
          </Box>
        </Box>
      )}

      {(groupBy.length > 0 || aggregations.length > 0) && (
        <Box sx={{ mt: 2, p: 2, bgcolor: 'grey.50', borderRadius: 1 }}>
          <Typography variant="caption" color="text.secondary">
            <strong>Note:</strong> When using aggregations or grouping, only aggregated fields and grouped fields will be included in the results.
            {groupBy.length === 0 && aggregations.length > 0 && ' No grouping means the aggregation will be calculated across all rows.'}
          </Typography>
        </Box>
      )}
    </Paper>
  );
};