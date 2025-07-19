import React from 'react';
import {
  Box,
  Typography,
  FormControlLabel,
  Checkbox,
  Paper,
  Grid,
  Chip,
} from '@mui/material';
import { QueryField } from '../../types/api';

interface FieldSelectorProps {
  availableFields: QueryField[];
  selectedFields: QueryField[];
  onChange: (fields: QueryField[]) => void;
}

export const FieldSelector: React.FC<FieldSelectorProps> = ({
  availableFields,
  selectedFields,
  onChange,
}) => {
  const handleFieldToggle = (field: QueryField) => {
    const isSelected = selectedFields.some(f => f.name === field.name);
    
    if (isSelected) {
      onChange(selectedFields.filter(f => f.name !== field.name));
    } else {
      onChange([...selectedFields, { ...field, selected: true }]);
    }
  };

  const handleSelectAll = () => {
    if (selectedFields.length === availableFields.length) {
      onChange([]);
    } else {
      onChange(availableFields.map(field => ({ ...field, selected: true })));
    }
  };

  const getFieldTypeColor = (type: string) => {
    switch (type.toLowerCase()) {
      case 'string': return 'primary';
      case 'number': return 'success';
      case 'datetime': return 'warning';
      case 'boolean': return 'error';
      default: return 'default';
    }
  };

  return (
    <Paper sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">Select Fields</Typography>
        <FormControlLabel
          control={
            <Checkbox
              checked={selectedFields.length === availableFields.length && availableFields.length > 0}
              indeterminate={selectedFields.length > 0 && selectedFields.length < availableFields.length}
              onChange={handleSelectAll}
            />
          }
          label="Select All"
        />
      </Box>

      {selectedFields.length > 0 && (
        <Box sx={{ mb: 3 }}>
          <Typography variant="subtitle2" gutterBottom>
            Selected Fields ({selectedFields.length}):
          </Typography>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
            {selectedFields.map((field) => (
              <Chip
                key={field.name}
                label={field.label || field.name}
                color={getFieldTypeColor(field.type) as 'primary' | 'secondary' | 'default' | 'error' | 'info' | 'success' | 'warning'}
                variant="outlined"
                onDelete={() => handleFieldToggle(field)}
                size="small"
              />
            ))}
          </Box>
        </Box>
      )}

      <Grid container spacing={2}>
        {availableFields.map((field) => (
          <Grid item xs={12} sm={6} md={4} key={field.name}>
            <FormControlLabel
              control={
                <Checkbox
                  checked={selectedFields.some(f => f.name === field.name)}
                  onChange={() => handleFieldToggle(field)}
                />
              }
              label={
                <Box>
                  <Typography variant="body2" sx={{ fontWeight: 'medium' }}>
                    {field.label || field.name}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {field.type} • {field.name}
                  </Typography>
                </Box>
              }
            />
          </Grid>
        ))}
      </Grid>

      {availableFields.length === 0 && (
        <Typography color="text.secondary" align="center">
          No fields available. Please check your data source connection.
        </Typography>
      )}
    </Paper>
  );
};