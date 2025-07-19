// API Service Layer for Click-Lite Log Analytics

import axios, { AxiosResponse } from 'axios';
import {
  Dashboard,
  DashboardListResponse,
  QueryBuilder,
  QueryBuilderResponse,
  AvailableFields,
  WidgetDataResponse,
  DashboardShare,
  ApiResponse,
} from '../types/api';

// Base API configuration
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:20002/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Query Builder API
export const queryBuilderApi = {
  // Get available fields for query building
  getFields: async (): Promise<AvailableFields> => {
    const response: AxiosResponse<AvailableFields> = await api.get('/querybuilder/fields');
    return response.data;
  },

  // Generate SQL from query builder configuration
  generateSQL: async (queryBuilder: QueryBuilder): Promise<string> => {
    const response: AxiosResponse<{ sql: string }> = await api.post('/querybuilder/sql', queryBuilder);
    return response.data.sql;
  },

  // Execute query builder and get results
  execute: async (queryBuilder: QueryBuilder): Promise<QueryBuilderResponse> => {
    const response: AxiosResponse<QueryBuilderResponse> = await api.post('/querybuilder/execute', queryBuilder);
    return response.data;
  },

  // Save query builder configuration
  save: async (queryBuilder: QueryBuilder): Promise<QueryBuilder> => {
    const response: AxiosResponse<QueryBuilder> = await api.post('/querybuilder/save', queryBuilder);
    return response.data;
  },

  // Get saved query builders
  list: async (): Promise<QueryBuilder[]> => {
    const response: AxiosResponse<{ query_builders: QueryBuilder[] }> = await api.get('/querybuilder/saved');
    return response.data.query_builders || [];
  },

  // Get specific saved query builder
  get: async (id: string): Promise<QueryBuilder> => {
    const response: AxiosResponse<QueryBuilder> = await api.get(`/querybuilder/saved/${id}`);
    return response.data;
  },

  // Update saved query builder
  update: async (id: string, queryBuilder: Partial<QueryBuilder>): Promise<QueryBuilder> => {
    const response: AxiosResponse<QueryBuilder> = await api.put(`/querybuilder/saved/${id}`, queryBuilder);
    return response.data;
  },

  // Delete saved query builder
  delete: async (id: string): Promise<void> => {
    await api.delete(`/querybuilder/saved/${id}`);
  },
};

// Dashboard API
export const dashboardApi = {
  // List all dashboards
  list: async (): Promise<DashboardListResponse> => {
    const response: AxiosResponse<DashboardListResponse> = await api.get('/dashboards');
    return response.data;
  },

  // Create new dashboard
  create: async (dashboard: Omit<Dashboard, 'id' | 'created_at' | 'updated_at'>): Promise<Dashboard> => {
    const response: AxiosResponse<Dashboard> = await api.post('/dashboards', dashboard);
    return response.data;
  },

  // Get specific dashboard
  get: async (id: string): Promise<Dashboard> => {
    const response: AxiosResponse<Dashboard> = await api.get(`/dashboards/${id}`);
    return response.data;
  },

  // Update dashboard
  update: async (id: string, updates: Partial<Dashboard>): Promise<Dashboard> => {
    const response: AxiosResponse<Dashboard> = await api.put(`/dashboards/${id}`, updates);
    return response.data;
  },

  // Delete dashboard
  delete: async (id: string): Promise<void> => {
    await api.delete(`/dashboards/${id}`);
  },

  // Execute widget query
  executeWidgetQuery: async (dashboardId: string, widgetId: string): Promise<any> => {
    const response: AxiosResponse<any> = await api.get(`/dashboards/${dashboardId}/widgets/${widgetId}/query`);
    return response.data;
  },

  // Get widget data (formatted for charts/metrics)
  getWidgetData: async (dashboardId: string, widgetId: string): Promise<WidgetDataResponse> => {
    const response: AxiosResponse<WidgetDataResponse> = await api.get(`/dashboards/${dashboardId}/widgets/${widgetId}/data`);
    return response.data;
  },

  // Share dashboard
  share: async (
    id: string,
    permissions: string[],
    expiresAt?: string
  ): Promise<DashboardShare> => {
    const response: AxiosResponse<DashboardShare> = await api.post(`/dashboards/${id}/share`, {
      permissions,
      expires_at: expiresAt,
    });
    return response.data;
  },

  // Get shared dashboard by token
  getShared: async (token: string): Promise<Dashboard> => {
    const response: AxiosResponse<Dashboard> = await api.get(`/dashboards/shared/${token}`);
    return response.data;
  },
};

// Query Engine API
export const queryApi = {
  // Execute SQL query
  execute: async (sql: string, parameters?: Record<string, any>): Promise<any> => {
    const response: AxiosResponse<any> = await api.post('/query/execute', {
      query: sql,
      parameters: parameters || {},
    });
    return response.data;
  },

  // Validate SQL query
  validate: async (sql: string): Promise<{ valid: boolean; error?: string }> => {
    try {
      const response: AxiosResponse<{ valid: boolean; error?: string }> = await api.post('/query/validate', {
        query: sql,
      });
      return response.data;
    } catch (error) {
      return { valid: false, error: 'Query validation failed' };
    }
  },
};

// Saved Queries API
export const savedQueriesApi = {
  // List saved queries
  list: async (): Promise<any[]> => {
    const response: AxiosResponse<{ queries: any[] }> = await api.get('/query/saved');
    return response.data.queries || [];
  },

  // Save query
  save: async (query: {
    name: string;
    description?: string;
    query: string;
    parameters?: Record<string, any>;
  }): Promise<any> => {
    const response: AxiosResponse<any> = await api.post('/query/saved', query);
    return response.data;
  },

  // Get saved query
  get: async (id: string): Promise<any> => {
    const response: AxiosResponse<any> = await api.get(`/query/saved/${id}`);
    return response.data;
  },

  // Update saved query
  update: async (id: string, updates: any): Promise<any> => {
    const response: AxiosResponse<any> = await api.put(`/query/saved/${id}`, updates);
    return response.data;
  },

  // Delete saved query
  delete: async (id: string): Promise<void> => {
    await api.delete(`/query/saved/${id}`);
  },

  // Execute saved query with parameters
  execute: async (id: string, parameters?: Record<string, any>): Promise<any> => {
    const response: AxiosResponse<any> = await api.post(`/query/saved/${id}/execute`, {
      parameters: parameters || {},
    });
    return response.data;
  },
};

// WebSocket URL for real-time features
export const getWebSocketUrl = (): string => {
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsHost = import.meta.env.VITE_WS_URL || `${wsProtocol}//${window.location.host}/api/v1/ws`;
  return wsHost;
};

// Error handling utility
export const handleApiError = (error: any): string => {
  if (error.response?.data?.error) {
    return error.response.data.error;
  }
  if (error.response?.data?.message) {
    return error.response.data.message;
  }
  if (error.message) {
    return error.message;
  }
  return 'An unexpected error occurred';
};

// Request interceptor for authentication (when implemented)
api.interceptors.request.use(
  (config) => {
    // TODO: Add authentication token when auth is implemented
    // const token = getAuthToken();
    // if (token) {
    //   config.headers.Authorization = `Bearer ${token}`;
    // }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  (error) => {
    // Handle common errors
    if (error.response?.status === 401) {
      // TODO: Handle authentication errors
      // Authentication required
    } else if (error.response?.status >= 500) {
      // Server error
    }
    return Promise.reject(error);
  }
);

export default api;