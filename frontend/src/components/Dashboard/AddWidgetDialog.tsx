import React, { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Grid,
  Typography,
  Box,
  Stepper,
  Step,
  StepLabel,
  Paper,
} from '@mui/material';
import { CHART_TYPES } from '../../types/api';
import type { DashboardWidget, WidgetDataSource } from '../../types/api';

interface AddWidgetDialogProps {
  open: boolean;
  onClose: () => void;
  onAdd: (widget: Omit<DashboardWidget, 'id'>) => void;
}

const steps = ['Widget Type', 'Data Source', 'Configuration'];

export const AddWidgetDialog: React.FC<AddWidgetDialogProps> = ({
  open,
  onClose,
  onAdd,
}) => {
  const [activeStep, setActiveStep] = useState(0);
  const [widget, setWidget] = useState<Partial<DashboardWidget>>({
    type: 'chart',
    title: '',
    position: { x: 0, y: 0 },
    size: { width: 6, height: 300 },
    config: {
      chart_type: 'line',
      show_legend: true,
      show_grid: true,
    },
    data_source: {
      type: 'custom_sql',
      sql: 'SELECT timestamp, COUNT(*) as count FROM logs WHERE timestamp >= now() - INTERVAL 1 HOUR GROUP BY timestamp ORDER BY timestamp',
    },
    refresh_rate: 30000,
  });

  const handleNext = () => {
    setActiveStep((prevStep) => prevStep + 1);
  };

  const handleBack = () => {
    setActiveStep((prevStep) => prevStep - 1);
  };

  const handleReset = () => {
    setActiveStep(0);
    setWidget({
      type: 'chart',
      title: '',
      position: { x: 0, y: 0 },
      size: { width: 6, height: 300 },
      config: {
        chart_type: 'line',
        show_legend: true,
        show_grid: true,
      },
      data_source: {
        type: 'custom_sql',
        sql: 'SELECT timestamp, COUNT(*) as count FROM logs WHERE timestamp >= now() - INTERVAL 1 HOUR GROUP BY timestamp ORDER BY timestamp',
      },
      refresh_rate: 30000,
    });
  };

  const handleSubmit = () => {
    if (!widget.title?.trim()) {
      alert('Please provide a widget title');
      return;
    }

    onAdd(widget as Omit<DashboardWidget, 'id'>);
    handleReset();
  };

  const handleClose = () => {
    handleReset();
    onClose();
  };

  const updateWidget = (updates: Partial<DashboardWidget>) => {
    setWidget(prev => ({ ...prev, ...updates }));
  };

  const updateConfig = (configUpdates: any) => {
    setWidget(prev => ({
      ...prev,
      config: { ...prev.config, ...configUpdates },
    }));
  };

  const updateDataSource = (dataSourceUpdates: Partial<WidgetDataSource>) => {
    setWidget(prev => ({
      ...prev,
      data_source: { ...prev.data_source, ...dataSourceUpdates },
    }));
  };

  const renderStepContent = (step: number) => {
    switch (step) {
      case 0:
        return (
          <Grid container spacing={3}>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Widget Title"
                value={widget.title || ''}
                onChange={(e) => updateWidget({ title: e.target.value })}
                required
              />
            </Grid>
            <Grid item xs={12}>
              <FormControl fullWidth>
                <InputLabel>Widget Type</InputLabel>
                <Select
                  value={widget.type || 'chart'}
                  onChange={(e) => updateWidget({ type: e.target.value as any })}
                  label="Widget Type"
                >
                  <MenuItem value="chart">Chart</MenuItem>
                  <MenuItem value="table">Table</MenuItem>
                  <MenuItem value="metric">Metric</MenuItem>
                  <MenuItem value="text">Text</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={6}>
              <TextField
                fullWidth
                type="number"
                label="Width (Grid Columns)"
                value={widget.size?.width || 6}
                onChange={(e) => updateWidget({
                  size: { ...widget.size!, width: parseInt(e.target.value) }
                })}
                inputProps={{ min: 1, max: 12 }}
              />
            </Grid>
            <Grid item xs={6}>
              <TextField
                fullWidth
                type="number"
                label="Height (Pixels)"
                value={widget.size?.height || 300}
                onChange={(e) => updateWidget({
                  size: { ...widget.size!, height: parseInt(e.target.value) }
                })}
                inputProps={{ min: 200, max: 800 }}
              />
            </Grid>
          </Grid>
        );

      case 1:
        return (
          <Grid container spacing={3}>
            <Grid item xs={12}>
              <FormControl fullWidth>
                <InputLabel>Data Source Type</InputLabel>
                <Select
                  value={widget.data_source?.type || 'custom_sql'}
                  onChange={(e) => updateDataSource({ type: e.target.value as any })}
                  label="Data Source Type"
                >
                  <MenuItem value="custom_sql">Custom SQL</MenuItem>
                  <MenuItem value="query_builder">Query Builder</MenuItem>
                  <MenuItem value="saved_query">Saved Query</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            
            {widget.data_source?.type === 'custom_sql' && (
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  multiline
                  rows={6}
                  label="SQL Query"
                  value={widget.data_source?.sql || ''}
                  onChange={(e) => updateDataSource({ sql: e.target.value })}
                  placeholder="SELECT timestamp, COUNT(*) as count FROM logs WHERE timestamp >= now() - INTERVAL 1 HOUR GROUP BY timestamp ORDER BY timestamp"
                />
              </Grid>
            )}

            {widget.data_source?.type === 'query_builder' && (
              <Grid item xs={12}>
                <Paper sx={{ p: 2 }}>
                  <Typography color="text.secondary">
                    Query Builder integration will be available after creating the widget.
                    You can edit the widget to configure the query builder.
                  </Typography>
                </Paper>
              </Grid>
            )}

            {widget.data_source?.type === 'saved_query' && (
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Saved Query ID"
                  value={widget.data_source?.query_id || ''}
                  onChange={(e) => updateDataSource({ query_id: e.target.value })}
                  placeholder="Enter the ID of the saved query"
                />
              </Grid>
            )}
          </Grid>
        );

      case 2:
        return (
          <Grid container spacing={3}>
            {widget.type === 'chart' && (
              <>
                <Grid item xs={12}>
                  <FormControl fullWidth>
                    <InputLabel>Chart Type</InputLabel>
                    <Select
                      value={widget.config?.chart_type || 'line'}
                      onChange={(e) => updateConfig({ chart_type: e.target.value })}
                      label="Chart Type"
                    >
                      {CHART_TYPES.map((type) => (
                        <MenuItem key={type.value} value={type.value}>
                          {type.label}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={6}>
                  <FormControl fullWidth>
                    <InputLabel>Show Legend</InputLabel>
                    <Select
                      value={widget.config?.show_legend ? 'true' : 'false'}
                      onChange={(e) => updateConfig({ show_legend: e.target.value === 'true' })}
                      label="Show Legend"
                    >
                      <MenuItem value="true">Yes</MenuItem>
                      <MenuItem value="false">No</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
                <Grid item xs={6}>
                  <FormControl fullWidth>
                    <InputLabel>Show Grid</InputLabel>
                    <Select
                      value={widget.config?.show_grid ? 'true' : 'false'}
                      onChange={(e) => updateConfig({ show_grid: e.target.value === 'true' })}
                      label="Show Grid"
                    >
                      <MenuItem value="true">Yes</MenuItem>
                      <MenuItem value="false">No</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
              </>
            )}
            
            <Grid item xs={12}>
              <FormControl fullWidth>
                <InputLabel>Refresh Rate</InputLabel>
                <Select
                  value={widget.refresh_rate || 30000}
                  onChange={(e) => updateWidget({ refresh_rate: e.target.value as number })}
                  label="Refresh Rate"
                >
                  <MenuItem value={0}>Manual</MenuItem>
                  <MenuItem value={5000}>5 seconds</MenuItem>
                  <MenuItem value={10000}>10 seconds</MenuItem>
                  <MenuItem value={30000}>30 seconds</MenuItem>
                  <MenuItem value={60000}>1 minute</MenuItem>
                  <MenuItem value={300000}>5 minutes</MenuItem>
                </Select>
              </FormControl>
            </Grid>
          </Grid>
        );

      default:
        return null;
    }
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="md" fullWidth>
      <DialogTitle>Add New Widget</DialogTitle>
      
      <DialogContent>
        <Box sx={{ width: '100%', mb: 3 }}>
          <Stepper activeStep={activeStep}>
            {steps.map((label) => (
              <Step key={label}>
                <StepLabel>{label}</StepLabel>
              </Step>
            ))}
          </Stepper>
        </Box>

        <Box sx={{ mt: 2 }}>
          {renderStepContent(activeStep)}
        </Box>
      </DialogContent>

      <DialogActions>
        <Button onClick={handleClose}>Cancel</Button>
        <Box sx={{ flexGrow: 1 }} />
        <Button disabled={activeStep === 0} onClick={handleBack}>
          Back
        </Button>
        {activeStep === steps.length - 1 ? (
          <Button variant="contained" onClick={handleSubmit}>
            Add Widget
          </Button>
        ) : (
          <Button variant="contained" onClick={handleNext}>
            Next
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};