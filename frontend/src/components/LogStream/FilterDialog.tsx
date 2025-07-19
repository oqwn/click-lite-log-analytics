import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  IconButton,
  Box,
  Typography,
  Chip,
} from '@mui/material';
import { Add, Delete } from '@mui/icons-material';
import { LogFilter } from '../../types/log';

interface FilterDialogProps {
  open: boolean;
  filters: LogFilter[];
  onClose: () => void;
  onApply: (filters: LogFilter[]) => void;
}

const FIELDS = [
  { value: 'level', label: 'Level' },
  { value: 'service', label: 'Service' },
  { value: 'message', label: 'Message' },
  { value: 'trace_id', label: 'Trace ID' },
];

const OPERATORS = [
  { value: 'equals', label: 'Equals' },
  { value: 'contains', label: 'Contains' },
  { value: 'starts_with', label: 'Starts with' },
  { value: 'ends_with', label: 'Ends with' },
  { value: 'not_equals', label: 'Not equals' },
  { value: 'not_contains', label: 'Not contains' },
];

const FilterDialog: React.FC<FilterDialogProps> = ({
  open,
  filters,
  onClose,
  onApply,
}) => {
  const [localFilters, setLocalFilters] = useState<LogFilter[]>(filters);

  useEffect(() => {
    setLocalFilters(filters);
  }, [filters]);

  const handleAddFilter = () => {
    setLocalFilters([
      ...localFilters,
      { field: 'level', operator: 'equals', value: '' },
    ]);
  };

  const handleRemoveFilter = (index: number) => {
    setLocalFilters(localFilters.filter((_, i) => i !== index));
  };

  const handleFilterChange = (
    index: number,
    field: keyof LogFilter,
    value: string
  ) => {
    const updated = [...localFilters];
    updated[index] = { ...updated[index], [field]: value };
    setLocalFilters(updated);
  };

  const handleApply = () => {
    // Filter out empty filters
    const validFilters = localFilters.filter((f) => f.value.trim() !== '');
    onApply(validFilters);
  };

  const handleClearAll = () => {
    setLocalFilters([]);
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="md"
      fullWidth
    >
      <DialogTitle>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="h6">Log Filters</Typography>
          {localFilters.length > 0 && (
            <Button
              size="small"
              onClick={handleClearAll}
              color="error"
            >
              Clear All
            </Button>
          )}
        </Box>
      </DialogTitle>

      <DialogContent>
        <Box sx={{ mt: 2 }}>
          {localFilters.length === 0 ? (
            <Typography variant="body2" color="text.secondary" align="center">
              No filters applied. Click "Add Filter" to get started.
            </Typography>
          ) : (
            localFilters.map((filter, index) => (
              <Box
                key={index}
                sx={{
                  display: 'flex',
                  gap: 2,
                  mb: 2,
                  alignItems: 'center',
                }}
              >
                <FormControl size="small" sx={{ minWidth: 120 }}>
                  <InputLabel>Field</InputLabel>
                  <Select
                    value={filter.field}
                    label="Field"
                    onChange={(e) =>
                      handleFilterChange(index, 'field', e.target.value)
                    }
                  >
                    {FIELDS.map((field) => (
                      <MenuItem key={field.value} value={field.value}>
                        {field.label}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>

                <FormControl size="small" sx={{ minWidth: 140 }}>
                  <InputLabel>Operator</InputLabel>
                  <Select
                    value={filter.operator}
                    label="Operator"
                    onChange={(e) =>
                      handleFilterChange(index, 'operator', e.target.value)
                    }
                  >
                    {OPERATORS.map((op) => (
                      <MenuItem key={op.value} value={op.value}>
                        {op.label}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>

                <TextField
                  size="small"
                  label="Value"
                  value={filter.value}
                  onChange={(e) =>
                    handleFilterChange(index, 'value', e.target.value)
                  }
                  sx={{ flexGrow: 1 }}
                  placeholder="Enter filter value..."
                />

                <IconButton
                  size="small"
                  onClick={() => handleRemoveFilter(index)}
                  color="error"
                >
                  <Delete />
                </IconButton>
              </Box>
            ))
          )}

          <Button
            startIcon={<Add />}
            onClick={handleAddFilter}
            variant="outlined"
            size="small"
            sx={{ mt: 1 }}
          >
            Add Filter
          </Button>
        </Box>

        {localFilters.length > 0 && (
          <Box sx={{ mt: 3 }}>
            <Typography variant="body2" color="text.secondary" gutterBottom>
              Preview: Logs matching ALL of these conditions will be shown
            </Typography>
            <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mt: 1 }}>
              {localFilters
                .filter((f) => f.value.trim() !== '')
                .map((filter, index) => (
                  <Chip
                    key={index}
                    label={`${filter.field} ${filter.operator} "${filter.value}"`}
                    size="small"
                    color="primary"
                    variant="outlined"
                  />
                ))}
            </Box>
          </Box>
        )}
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          onClick={handleApply}
          variant="contained"
          color="primary"
        >
          Apply Filters
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default FilterDialog;