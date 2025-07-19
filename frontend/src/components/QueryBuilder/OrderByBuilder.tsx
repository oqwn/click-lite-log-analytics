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
  Chip,
} from '@mui/material';
import { Add, Delete, ArrowUpward, ArrowDownward } from '@mui/icons-material';
import { QueryField, QueryOrderBy } from '../../types/api';

interface OrderByBuilderProps {
  availableFields: QueryField[];
  orderBy: QueryOrderBy[];
  onChange: (orderBy: QueryOrderBy[]) => void;
}

export const OrderByBuilder: React.FC<OrderByBuilderProps> = ({
  availableFields,
  orderBy,
  onChange,
}) => {
  const addOrderBy = () => {
    const newOrderBy: QueryOrderBy = {
      field: '',
      direction: 'ASC',
    };
    onChange([...orderBy, newOrderBy]);
  };

  const updateOrderBy = (index: number, updates: Partial<QueryOrderBy>) => {
    const updatedOrderBy = orderBy.map((order, i) => 
      i === index ? { ...order, ...updates } : order
    );
    onChange(updatedOrderBy);
  };

  const removeOrderBy = (index: number) => {
    const updatedOrderBy = orderBy.filter((_, i) => i !== index);
    onChange(updatedOrderBy);
  };

  const moveOrderBy = (index: number, direction: 'up' | 'down') => {
    const newIndex = direction === 'up' ? index - 1 : index + 1;
    if (newIndex < 0 || newIndex >= orderBy.length) return;

    const updatedOrderBy = [...orderBy];
    [updatedOrderBy[index], updatedOrderBy[newIndex]] = [updatedOrderBy[newIndex], updatedOrderBy[index]];
    onChange(updatedOrderBy);
  };

  const getFieldLabel = (fieldName: string): string => {
    const field = availableFields.find(f => f.name === fieldName);
    return field?.label || fieldName;
  };

  return (
    <Paper sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">Sort Order</Typography>
        <Button
          variant="outlined"
          startIcon={<Add />}
          onClick={addOrderBy}
          size="small"
        >
          Add Sort Field
        </Button>
      </Box>

      {orderBy.length === 0 ? (
        <Typography color="text.secondary" align="center">
          No sorting applied. Results will be returned in database default order.
        </Typography>
      ) : (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {orderBy.map((order, index) => (
            <Paper key={index} variant="outlined" sx={{ p: 2 }}>
              <Grid container spacing={2} alignItems="center">
                <Grid item xs={1}>
                  <Typography variant="body2" color="text.secondary" align="center">
                    {index + 1}
                  </Typography>
                </Grid>
                
                <Grid item xs={12} sm={5}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Field</InputLabel>
                    <Select
                      value={order.field}
                      onChange={(e) => updateOrderBy(index, { field: e.target.value })}
                      label="Field"
                    >
                      {availableFields.map((field) => (
                        <MenuItem key={field.name} value={field.name}>
                          {field.label || field.name} ({field.type})
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                
                <Grid item xs={12} sm={3}>
                  <FormControl fullWidth size="small">
                    <InputLabel>Direction</InputLabel>
                    <Select
                      value={order.direction}
                      onChange={(e) => updateOrderBy(index, { direction: e.target.value as 'ASC' | 'DESC' })}
                      label="Direction"
                    >
                      <MenuItem value="ASC">
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <ArrowUpward fontSize="small" />
                          Ascending
                        </Box>
                      </MenuItem>
                      <MenuItem value="DESC">
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <ArrowDownward fontSize="small" />
                          Descending
                        </Box>
                      </MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
                
                <Grid item xs={12} sm={2}>
                  <Box sx={{ display: 'flex', gap: 1 }}>
                    <IconButton
                      size="small"
                      onClick={() => moveOrderBy(index, 'up')}
                      disabled={index === 0}
                    >
                      <ArrowUpward fontSize="small" />
                    </IconButton>
                    <IconButton
                      size="small"
                      onClick={() => moveOrderBy(index, 'down')}
                      disabled={index === orderBy.length - 1}
                    >
                      <ArrowDownward fontSize="small" />
                    </IconButton>
                  </Box>
                </Grid>
                
                <Grid item xs={12} sm={1}>
                  <IconButton
                    color="error"
                    onClick={() => removeOrderBy(index)}
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

      {orderBy.length > 0 && (
        <Box sx={{ mt: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            Sort Summary:
          </Typography>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
            {orderBy.map((order, index) => (
              <Chip
                key={index}
                label={`${index + 1}. ${getFieldLabel(order.field)} ${order.direction}`}
                variant="outlined"
                size="small"
                icon={order.direction === 'ASC' ? <ArrowUpward fontSize="small" /> : <ArrowDownward fontSize="small" />}
              />
            ))}
          </Box>
        </Box>
      )}

      {orderBy.length > 1 && (
        <Box sx={{ mt: 2, p: 2, bgcolor: 'grey.50', borderRadius: 1 }}>
          <Typography variant="caption" color="text.secondary">
            <strong>Sort Priority:</strong> Results will be sorted by field 1 first, then by field 2 for rows with the same value in field 1, and so on.
            Use the arrow buttons to reorder the priority.
          </Typography>
        </Box>
      )}
    </Paper>
  );
};