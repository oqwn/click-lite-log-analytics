import React, { useState, useEffect } from 'react';
import {
  Paper,
  Typography,
  IconButton,
  Box,
  Menu,
  MenuItem,
  CircularProgress,
  Alert,
} from '@mui/material';
import {
  MoreVert,
  Edit,
  Delete,
  Refresh,
  Fullscreen,
} from '@mui/icons-material';
import { ChartWidget } from '../Charts/ChartWidget';
import { TableWidget } from './TableWidget';
import { MetricWidget } from './MetricWidget';
import { TextWidget } from './TextWidget';
import { dashboardApi, handleApiError } from '../../services/api';
import { DashboardWidget as WidgetType } from '../../types/api';

interface DashboardWidgetProps {
  widget: WidgetType;
  dashboardId: string;
  onUpdate?: (updates: Partial<WidgetType>) => void;
  onDelete: () => void;
  isEditing?: boolean;
}

export const DashboardWidget: React.FC<DashboardWidgetProps> = ({
  widget,
  dashboardId,
  onUpdate,
  onDelete,
  isEditing = false,
}) => {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date>(new Date());

  useEffect(() => {
    loadWidgetData();
  }, [widget.id, dashboardId, loadWidgetData]);

  useEffect(() => {
    // Set up auto-refresh if specified
    if (widget.refresh_rate && widget.refresh_rate > 0) {
      const interval = setInterval(() => {
        loadWidgetData();
      }, widget.refresh_rate);

      return () => clearInterval(interval);
    }
  }, [widget.refresh_rate, loadWidgetData]);

  const loadWidgetData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      const response = await dashboardApi.getWidgetData(dashboardId, widget.id);
      setData(response.data);
      setLastRefresh(new Date());
    } catch (err) {
      setError(handleApiError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>) => {
    setMenuAnchor(event.currentTarget);
  };

  const handleMenuClose = () => {
    setMenuAnchor(null);
  };

  const handleRefresh = () => {
    loadWidgetData();
    handleMenuClose();
  };

  const handleEdit = () => {
    // TODO: Open widget edit dialog
    handleMenuClose();
  };

  const handleDelete = () => {
    if (window.confirm(`Are you sure you want to delete "${widget.title}"?`)) {
      onDelete();
    }
    handleMenuClose();
  };

  const renderWidgetContent = () => {
    if (loading) {
      return (
        <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: 200 }}>
          <CircularProgress />
        </Box>
      );
    }

    if (error) {
      return (
        <Alert severity="error" sx={{ m: 2 }}>
          {error}
        </Alert>
      );
    }

    switch (widget.type) {
      case 'chart':
        return (
          <ChartWidget
            data={data}
            config={widget.config}
            height={widget.size.height - 60} // Account for header
          />
        );
      case 'table':
        return (
          <TableWidget
            data={data}
            config={widget.config}
            height={widget.size.height - 60}
          />
        );
      case 'metric':
        return (
          <MetricWidget
            data={data}
            config={widget.config}
          />
        );
      case 'text':
        return (
          <TextWidget
            data={data}
            config={widget.config}
          />
        );
      default:
        return (
          <Alert severity="warning" sx={{ m: 2 }}>
            Unknown widget type: {widget.type}
          </Alert>
        );
    }
  };

  const getRefreshRateLabel = (ms?: number): string => {
    if (!ms) return 'Manual';
    if (ms < 60000) return `${ms / 1000}s`;
    return `${Math.floor(ms / 60000)}m`;
  };

  const formatLastRefresh = (date: Date): string => {
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    
    if (diff < 60000) return 'Just now';
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    return date.toLocaleTimeString();
  };

  return (
    <Paper
      sx={{
        height: widget.size.height,
        display: 'flex',
        flexDirection: 'column',
        position: 'relative',
        ...(isEditing && {
          border: '2px dashed',
          borderColor: 'primary.main',
        }),
      }}
    >
      {/* Widget Header */}
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          p: 2,
          pb: 1,
          borderBottom: '1px solid',
          borderColor: 'divider',
        }}
      >
        <Box sx={{ flexGrow: 1, minWidth: 0 }}>
          <Typography variant="h6" noWrap>
            {widget.title}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {getRefreshRateLabel(widget.refresh_rate)} • Updated {formatLastRefresh(lastRefresh)}
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center' }}>
          <IconButton size="small" onClick={handleRefresh} disabled={loading}>
            <Refresh />
          </IconButton>
          <IconButton size="small" onClick={handleMenuOpen}>
            <MoreVert />
          </IconButton>
        </Box>
      </Box>

      {/* Widget Content */}
      <Box sx={{ flexGrow: 1, overflow: 'hidden' }}>
        {renderWidgetContent()}
      </Box>

      {/* Context Menu */}
      <Menu
        anchorEl={menuAnchor}
        open={Boolean(menuAnchor)}
        onClose={handleMenuClose}
      >
        <MenuItem onClick={handleRefresh}>
          <Refresh sx={{ mr: 1 }} /> Refresh
        </MenuItem>
        <MenuItem onClick={handleEdit}>
          <Edit sx={{ mr: 1 }} /> Edit
        </MenuItem>
        <MenuItem onClick={() => window.open(`/widget/${widget.id}/fullscreen`, '_blank')}>
          <Fullscreen sx={{ mr: 1 }} /> Fullscreen
        </MenuItem>
        <MenuItem onClick={handleDelete} sx={{ color: 'error.main' }}>
          <Delete sx={{ mr: 1 }} /> Delete
        </MenuItem>
      </Menu>

      {/* Loading Overlay */}
      {loading && (
        <Box
          sx={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            bgcolor: 'rgba(255, 255, 255, 0.7)',
            zIndex: 1,
          }}
        >
          <CircularProgress />
        </Box>
      )}
    </Paper>
  );
};