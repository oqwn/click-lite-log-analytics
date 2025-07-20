import React from 'react';
import {
  Box,
  AppBar,
  Toolbar,
  Typography,
  Drawer,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  ListItemButton,
  Divider,
  IconButton,
} from '@mui/material';
import {
  Dashboard as DashboardIcon,
  Search,
  Timeline,
  Menu as MenuIcon,
  Home,
  MonitorHeart,
} from '@mui/icons-material';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';

const drawerWidth = 240;

const navigation = [
  { name: 'Home', path: '/', icon: Home },
  { name: 'Live Logs', path: '/logs', icon: Timeline },
  { name: 'Query Builder', path: '/query-builder', icon: Search },
  { name: 'Dashboards', path: '/dashboards', icon: DashboardIcon },
  { name: 'Monitoring', path: '/monitoring', icon: MonitorHeart },
];

export const MainLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [mobileOpen, setMobileOpen] = React.useState(false);

  const handleDrawerToggle = () => {
    setMobileOpen(!mobileOpen);
  };

  const drawer = (
    <Box>
      <Toolbar>
        <Typography variant="h6" noWrap component="div">
          Click-Lite Analytics
        </Typography>
      </Toolbar>
      <Divider />
      <List>
        {navigation.map((item) => {
          const Icon = item.icon;
          const isActive = location.pathname === item.path;
          
          return (
            <ListItem key={item.name} disablePadding>
              <ListItemButton
                selected={isActive}
                onClick={() => navigate(item.path)}
              >
                <ListItemIcon>
                  <Icon color={isActive ? 'primary' : 'inherit'} />
                </ListItemIcon>
                <ListItemText primary={item.name} />
              </ListItemButton>
            </ListItem>
          );
        })}
      </List>
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', height: '100vh' }}>
      {/* Mobile drawer */}
      <Drawer
        variant="temporary"
        open={mobileOpen}
        onClose={handleDrawerToggle}
        ModalProps={{
          keepMounted: true, // Better open performance on mobile.
        }}
        sx={{
          display: { xs: 'block', sm: 'none' },
          '& .MuiDrawer-paper': { boxSizing: 'border-box', width: drawerWidth },
        }}
      >
        {drawer}
      </Drawer>

      {/* Desktop drawer */}
      <Drawer
        variant="permanent"
        sx={{
          display: { xs: 'none', sm: 'block' },
          '& .MuiDrawer-paper': { boxSizing: 'border-box', width: drawerWidth },
        }}
        open
      >
        {drawer}
      </Drawer>

      {/* Main content */}
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          ml: { sm: `${drawerWidth}px` },
          width: { sm: `calc(100% - ${drawerWidth}px)` },
          height: '100vh',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {/* Top app bar for mobile */}
        <AppBar
          position="fixed"
          sx={{
            width: { sm: `calc(100% - ${drawerWidth}px)` },
            ml: { sm: `${drawerWidth}px` },
            display: { xs: 'block', sm: 'none' },
          }}
        >
          <Toolbar>
            <IconButton
              color="inherit"
              aria-label="open drawer"
              edge="start"
              onClick={handleDrawerToggle}
              sx={{ mr: 2 }}
            >
              <MenuIcon />
            </IconButton>
            <Typography variant="h6" noWrap component="div">
              Click-Lite Analytics
            </Typography>
          </Toolbar>
        </AppBar>

        {/* Page content */}
        <Box
          sx={{
            flexGrow: 1,
            p: 3,
            pt: { xs: 11, sm: 3 }, // Account for mobile app bar
            overflow: 'auto',
          }}
        >
          <Outlet />
        </Box>
      </Box>
    </Box>
  );
};