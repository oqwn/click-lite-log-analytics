import { ThemeProvider, createTheme } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MainLayout } from './components/Layout/MainLayout';
import { HomePage } from './pages/HomePage';
import { QueryBuilder } from './components/QueryBuilder';
import { DashboardList, DashboardView } from './components/Dashboard';
import { LogStreamEnhanced } from './components/LogStream/LogStreamEnhanced';

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

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

function App() {
  const wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:20002/api/v1/ws';

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<MainLayout />}>
              <Route index element={<HomePage />} />
              <Route path="logs" element={<LogStreamEnhanced wsUrl={wsUrl} />} />
              <Route path="query-builder" element={<QueryBuilder />} />
              <Route path="dashboards" element={<DashboardList />} />
              <Route path="dashboard/:id" element={<DashboardView />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

export default App;
