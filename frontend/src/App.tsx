import React from 'react';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import { Box, Container, AppBar, Toolbar, Typography } from '@mui/material';
import LogStream from './components/LogStream';

const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#1976d2',
    },
    secondary: {
      main: '#dc004e',
    },
  },
});

function App() {
  // Get WebSocket URL from environment or use default
  const wsUrl = process.env.REACT_APP_WS_URL || 'ws://localhost:8080/api/v1/ws';

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
        <AppBar position="static">
          <Toolbar>
            <Typography variant="h6" component="div" sx={{ flexGrow: 1 }}>
              Click-Lite Log Analytics
            </Typography>
          </Toolbar>
        </AppBar>
        
        <Container maxWidth={false} sx={{ flexGrow: 1, py: 3, display: 'flex' }}>
          <LogStream wsUrl={wsUrl} />
        </Container>
      </Box>
    </ThemeProvider>
  );
}

export default App;
