import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Button,
  Grid,
  IconButton,
  Alert,
  CircularProgress,
  AppBar,
  Toolbar,
  Breadcrumbs,
  Link,
  Chip,
} from '@mui/material';
import {
  Edit,
  Share,
  Refresh,
  Add,
  Home,
} from '@mui/icons-material';
import { useParams, useNavigate } from 'react-router-dom';
import { DashboardWidget } from './DashboardWidget';
import { AddWidgetDialog } from './AddWidgetDialog';
import { dashboardApi, handleApiError } from '../../services/api';
import { Dashboard, DashboardWidget as WidgetType } from '../../types/api';

export const DashboardView: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [dashboard, setDashboard] = useState<Dashboard | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [addWidgetOpen, setAddWidgetOpen] = useState(false);

  useEffect(() => {
    if (id) {
      loadDashboard();
    }
  }, [id, loadDashboard]);

  const loadDashboard = async () => {
    if (!id) return;

    try {
      setLoading(true);
      const dashboardData = await dashboardApi.get(id);
      setDashboard(dashboardData);
      setError(null);
    } catch (err) {
      setError(handleApiError(err));
    } finally {
      setLoading(false);
    }
  };

  const refreshAllWidgets = async () => {
    setRefreshing(true);
    // Trigger refresh on all widgets
    // This would be implemented by each widget component
    setTimeout(() => setRefreshing(false), 1000);
  };

  const handleShare = async () => {
    if (!dashboard?.id) return;

    try {
      const share = await dashboardApi.share(dashboard.id, ['view'], undefined);
      const shareUrl = `${window.location.origin}/shared/dashboard/${share.share_token}`;
      
      await navigator.clipboard.writeText(shareUrl);
      alert('Share URL copied to clipboard!');
    } catch (err) {
      setError(handleApiError(err));
    }
  };

  const handleAddWidget = async (widget: Omit<WidgetType, 'id'>) => {
    if (!dashboard) return;

    const newWidget: WidgetType = {
      ...widget,
      id: `widget-${Date.now()}`,
    };

    try {
      const updatedDashboard = await dashboardApi.update(dashboard.id!, {
        widgets: [...dashboard.widgets, newWidget],
      });
      setDashboard(updatedDashboard);
      setAddWidgetOpen(false);
    } catch (err) {
      setError(handleApiError(err));
    }
  };

  const handleUpdateWidget = async (widgetId: string, updates: Partial<WidgetType>) => {
    if (!dashboard) return;

    const updatedWidgets = dashboard.widgets.map(widget =>
      widget.id === widgetId ? { ...widget, ...updates } : widget
    );

    try {
      const updatedDashboard = await dashboardApi.update(dashboard.id!, {
        widgets: updatedWidgets,
      });
      setDashboard(updatedDashboard);
    } catch (err) {
      setError(handleApiError(err));
    }
  };

  const handleDeleteWidget = async (widgetId: string) => {
    if (!dashboard) return;

    const updatedWidgets = dashboard.widgets.filter(widget => widget.id !== widgetId);

    try {
      const updatedDashboard = await dashboardApi.update(dashboard.id!, {
        widgets: updatedWidgets,
      });
      setDashboard(updatedDashboard);
    } catch (err) {
      setError(handleApiError(err));
    }
  };

  const formatDate = (dateString?: string): string => {
    if (!dateString) return 'Unknown';
    return new Date(dateString).toLocaleString();
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !dashboard) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">
          {error || 'Dashboard not found'}
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <AppBar position="static" color="default" elevation={1}>
        <Toolbar>
          <Box sx={{ flexGrow: 1 }}>
            <Breadcrumbs aria-label="breadcrumb">
              <Link
                color="inherit"
                href="#"
                onClick={() => navigate('/dashboards')}
                sx={{ display: 'flex', alignItems: 'center' }}
              >
                <Home sx={{ mr: 0.5 }} fontSize="inherit" />
                Dashboards
              </Link>
              <Typography color="text.primary">{dashboard.name}</Typography>
            </Breadcrumbs>
            <Typography variant="h6" sx={{ mt: 1 }}>
              {dashboard.name}
            </Typography>
            {dashboard.description && (
              <Typography variant="body2" color="text.secondary">
                {dashboard.description}
              </Typography>
            )}
          </Box>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Chip
              label={`${dashboard.widgets.length} widgets`}
              size="small"
              variant="outlined"
            />
            <Chip
              label={`Updated: ${formatDate(dashboard.updated_at)}`}
              size="small"
              variant="outlined"
            />
            
            <IconButton onClick={refreshAllWidgets} disabled={refreshing}>
              <Refresh />
            </IconButton>
            
            <Button
              startIcon={<Add />}
              onClick={() => setAddWidgetOpen(true)}
              variant="outlined"
              size="small"
            >
              Add Widget
            </Button>
            
            <IconButton onClick={handleShare}>
              <Share />
            </IconButton>
            
            <Button
              startIcon={<Edit />}
              onClick={() => navigate(`/dashboard/${dashboard.id}/edit`)}
              variant="contained"
              size="small"
            >
              Edit
            </Button>
          </Box>
        </Toolbar>
      </AppBar>

      {/* Content */}
      <Box sx={{ flexGrow: 1, p: 3, overflow: 'auto' }}>
        {error && (
          <Alert severity="error" sx={{ mb: 3 }}>
            {error}
          </Alert>
        )}

        {dashboard.widgets.length === 0 ? (
          <Paper sx={{ p: 6, textAlign: 'center' }}>
            <Typography variant="h6" gutterBottom>
              No widgets in this dashboard
            </Typography>
            <Typography color="text.secondary" sx={{ mb: 3 }}>
              Add your first widget to start visualizing your data
            </Typography>
            <Button
              variant="contained"
              startIcon={<Add />}
              onClick={() => setAddWidgetOpen(true)}
            >
              Add Your First Widget
            </Button>
          </Paper>
        ) : (
          <Grid container spacing={dashboard.layout.grid_gap / 8}>
            {dashboard.widgets.map((widget) => (
              <Grid
                item
                xs={12}
                sm={6}
                md={widget.size.width}
                key={widget.id}
              >
                <DashboardWidget
                  widget={widget}
                  dashboardId={dashboard.id!}
                  onUpdate={(updates) => handleUpdateWidget(widget.id, updates)}
                  onDelete={() => handleDeleteWidget(widget.id)}
                />
              </Grid>
            ))}
          </Grid>
        )}
      </Box>

      {/* Add Widget Dialog */}
      <AddWidgetDialog
        open={addWidgetOpen}
        onClose={() => setAddWidgetOpen(false)}
        onAdd={handleAddWidget}
      />
    </Box>
  );
};