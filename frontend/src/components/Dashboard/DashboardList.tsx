import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Button,
  Grid,
  Card,
  CardContent,
  CardActions,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Alert,
  Chip,
  Menu,
  MenuItem,
} from '@mui/material';
import {
  Add,
  Dashboard as DashboardIcon,
  Edit,
  Delete,
  Share,
  MoreVert,
  Visibility,
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { dashboardApi, handleApiError } from '../../services/api';
import { Dashboard } from '../../types/api';

export const DashboardList: React.FC = () => {
  const navigate = useNavigate();
  const [dashboards, setDashboards] = useState<Dashboard[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedDashboard, setSelectedDashboard] = useState<Dashboard | null>(null);
  const [newDashboard, setNewDashboard] = useState({
    name: '',
    description: '',
  });
  const [menuAnchor, setMenuAnchor] = useState<{ element: HTMLElement; dashboard: Dashboard } | null>(null);

  useEffect(() => {
    loadDashboards();
  }, []);

  const loadDashboards = async () => {
    try {
      setLoading(true);
      const response = await dashboardApi.list();
      setDashboards(response.dashboards || []);
      setError(null);
    } catch (err) {
      setError(handleApiError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleCreateDashboard = async () => {
    if (!newDashboard.name.trim()) {
      setError('Dashboard name is required');
      return;
    }

    try {
      const dashboard = await dashboardApi.create({
        name: newDashboard.name,
        description: newDashboard.description,
        widgets: [],
        layout: {
          columns: 12,
          row_height: 100,
          grid_gap: 16,
          auto_layout: true,
        },
        settings: {
          refresh_interval: 30000,
          theme: 'light',
        },
        is_public: false,
      });
      
      setDashboards(prev => [...prev, dashboard]);
      setCreateDialogOpen(false);
      setNewDashboard({ name: '', description: '' });
      setError(null);
      
      // Navigate to the new dashboard
      navigate(`/dashboard/${dashboard.id}`);
    } catch (err) {
      setError(handleApiError(err));
    }
  };

  const handleDeleteDashboard = async () => {
    if (!selectedDashboard?.id) return;

    try {
      await dashboardApi.delete(selectedDashboard.id);
      setDashboards(prev => prev.filter(d => d.id !== selectedDashboard.id));
      setDeleteDialogOpen(false);
      setSelectedDashboard(null);
      setError(null);
    } catch (err) {
      setError(handleApiError(err));
    }
  };

  const handleMenuClick = (event: React.MouseEvent<HTMLElement>, dashboard: Dashboard) => {
    setMenuAnchor({ element: event.currentTarget, dashboard });
  };

  const handleMenuClose = () => {
    setMenuAnchor(null);
  };

  const handleShare = async (dashboard: Dashboard) => {
    try {
      const share = await dashboardApi.share(dashboard.id!, ['view'], undefined);
      const shareUrl = `${window.location.origin}/shared/dashboard/${share.share_token}`;
      
      await navigator.clipboard.writeText(shareUrl);
      alert('Share URL copied to clipboard!');
    } catch (err) {
      setError(handleApiError(err));
    }
    handleMenuClose();
  };

  const formatDate = (dateString?: string): string => {
    if (!dateString) return 'Unknown';
    return new Date(dateString).toLocaleDateString();
  };

  const getWidgetTypeColor = (type: string) => {
    switch (type) {
      case 'chart': return 'primary';
      case 'table': return 'success';
      case 'metric': return 'warning';
      case 'text': return 'info';
      default: return 'default';
    }
  };

  if (loading) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <Typography>Loading dashboards...</Typography>
      </Paper>
    );
  }

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">Dashboards</Typography>
        <Button
          variant="contained"
          startIcon={<Add />}
          onClick={() => setCreateDialogOpen(true)}
        >
          Create Dashboard
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      {dashboards.length === 0 ? (
        <Paper sx={{ p: 6, textAlign: 'center' }}>
          <DashboardIcon sx={{ fontSize: 64, color: 'text.secondary', mb: 2 }} />
          <Typography variant="h6" gutterBottom>
            No dashboards yet
          </Typography>
          <Typography color="text.secondary" sx={{ mb: 3 }}>
            Create your first dashboard to start visualizing your log data
          </Typography>
          <Button
            variant="contained"
            startIcon={<Add />}
            onClick={() => setCreateDialogOpen(true)}
          >
            Create Your First Dashboard
          </Button>
        </Paper>
      ) : (
        <Grid container spacing={3}>
          {dashboards.map((dashboard) => (
            <Grid item xs={12} sm={6} md={4} key={dashboard.id}>
              <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
                <CardContent sx={{ flexGrow: 1 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                    <Typography variant="h6" noWrap>
                      {dashboard.name}
                    </Typography>
                    <IconButton
                      size="small"
                      onClick={(e) => handleMenuClick(e, dashboard)}
                    >
                      <MoreVert />
                    </IconButton>
                  </Box>

                  {dashboard.description && (
                    <Typography
                      variant="body2"
                      color="text.secondary"
                      sx={{ mb: 2, overflow: 'hidden', textOverflow: 'ellipsis' }}
                    >
                      {dashboard.description}
                    </Typography>
                  )}

                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 2 }}>
                    <Chip
                      label={`${dashboard.widgets.length} widgets`}
                      size="small"
                      variant="outlined"
                    />
                    {dashboard.is_public && (
                      <Chip
                        label="Public"
                        size="small"
                        color="success"
                        variant="outlined"
                      />
                    )}
                  </Box>

                  {dashboard.widgets.length > 0 && (
                    <Box sx={{ mb: 2 }}>
                      <Typography variant="caption" color="text.secondary" gutterBottom>
                        Widget Types:
                      </Typography>
                      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 0.5 }}>
                        {[...new Set(dashboard.widgets.map(w => w.type))].map((type) => (
                          <Chip
                            key={type}
                            label={type}
                            size="small"
                            color={getWidgetTypeColor(type) as any}
                          />
                        ))}
                      </Box>
                    </Box>
                  )}

                  <Typography variant="caption" color="text.secondary">
                    Updated: {formatDate(dashboard.updated_at)}
                  </Typography>
                </CardContent>

                <CardActions>
                  <Button
                    size="small"
                    startIcon={<Visibility />}
                    onClick={() => navigate(`/dashboard/${dashboard.id}`)}
                  >
                    View
                  </Button>
                  <Button
                    size="small"
                    startIcon={<Edit />}
                    onClick={() => navigate(`/dashboard/${dashboard.id}/edit`)}
                  >
                    Edit
                  </Button>
                </CardActions>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}

      {/* Context Menu */}
      <Menu
        anchorEl={menuAnchor?.element}
        open={Boolean(menuAnchor)}
        onClose={handleMenuClose}
      >
        <MenuItem onClick={() => {
          navigate(`/dashboard/${menuAnchor?.dashboard.id}`);
          handleMenuClose();
        }}>
          <Visibility sx={{ mr: 1 }} /> View
        </MenuItem>
        <MenuItem onClick={() => {
          navigate(`/dashboard/${menuAnchor?.dashboard.id}/edit`);
          handleMenuClose();
        }}>
          <Edit sx={{ mr: 1 }} /> Edit
        </MenuItem>
        <MenuItem onClick={() => menuAnchor && handleShare(menuAnchor.dashboard)}>
          <Share sx={{ mr: 1 }} /> Share
        </MenuItem>
        <MenuItem onClick={() => {
          setSelectedDashboard(menuAnchor?.dashboard || null);
          setDeleteDialogOpen(true);
          handleMenuClose();
        }}>
          <Delete sx={{ mr: 1 }} /> Delete
        </MenuItem>
      </Menu>

      {/* Create Dashboard Dialog */}
      <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Create New Dashboard</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Dashboard Name"
            fullWidth
            variant="outlined"
            value={newDashboard.name}
            onChange={(e) => setNewDashboard(prev => ({ ...prev, name: e.target.value }))}
            sx={{ mb: 2 }}
          />
          <TextField
            margin="dense"
            label="Description (optional)"
            fullWidth
            multiline
            rows={3}
            variant="outlined"
            value={newDashboard.description}
            onChange={(e) => setNewDashboard(prev => ({ ...prev, description: e.target.value }))}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleCreateDashboard} variant="contained">
            Create
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onClose={() => setDeleteDialogOpen(false)}>
        <DialogTitle>Delete Dashboard</DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to delete "{selectedDashboard?.name}"? This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleDeleteDashboard} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};