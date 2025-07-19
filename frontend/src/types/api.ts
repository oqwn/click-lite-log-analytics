// API Types for Click-Lite Log Analytics

export interface LogFilter {
  field: string;
  operator: string;
  value: string;
}

export interface WebSocketMessage {
  type: string;
  action?: string;
  data?: any;
  filters?: LogFilter[];
}

// Query Builder Types
export interface QueryField {
  name: string;
  type: string;
  label: string;
  selected?: boolean;
}

export interface QueryBuilderFilter {
  id: string;
  field: string;
  operator: string;
  value: any;
  values?: any[];
  logical_op?: string;
}

export interface QueryAggregation {
  id: string;
  function: string;
  field?: string;
  alias?: string;
}

export interface QueryOrderBy {
  field: string;
  direction: 'ASC' | 'DESC';
}

export interface QueryTimeRange {
  start?: string;
  end?: string;
  relative?: string;
}

export interface QueryBuilder {
  id?: string;
  name: string;
  description?: string;
  fields: QueryField[];
  filters: QueryBuilderFilter[];
  aggregations: QueryAggregation[];
  group_by: string[];
  order_by: QueryOrderBy[];
  limit?: number;
  time_range?: QueryTimeRange;
  generated_sql?: string;
  created_at?: string;
  updated_at?: string;
  created_by?: string;
}

export interface QueryResultColumn {
  name: string;
  type: string;
  is_aggregated?: boolean;
}

export interface QueryBuilderResponse {
  sql: string;
  columns: QueryResultColumn[];
  rows: Record<string, any>[];
  row_count: number;
  execution_time_ms: number;
  error?: string;
}

export interface AvailableFields {
  fields: QueryField[];
}

// Dashboard Types
export interface WidgetPosition {
  x: number;
  y: number;
}

export interface WidgetSize {
  width: number;
  height: number;
}

export interface AxisConfig {
  label?: string;
  min?: number;
  max?: number;
  log_scale?: boolean;
}

export interface ThresholdConfig {
  value: number;
  color: string;
  label?: string;
  show_line: boolean;
}

export interface WidgetConfig {
  chart_type?: string;
  x_axis?: AxisConfig;
  y_axis?: AxisConfig;
  colors?: string[];
  show_legend?: boolean;
  show_grid?: boolean;
  stacked?: boolean;
  time_format?: string;
  value_format?: string;
  threshold?: ThresholdConfig;
  custom_options?: Record<string, any>;
}

export interface WidgetDataSource {
  type: 'query_builder' | 'saved_query' | 'custom_sql';
  query_id?: string;
  sql?: string;
  query_builder?: QueryBuilder;
  parameters?: Record<string, any>;
}

export interface DashboardWidget {
  id: string;
  type: 'chart' | 'table' | 'metric' | 'text';
  title: string;
  position: WidgetPosition;
  size: WidgetSize;
  config: WidgetConfig;
  data_source: WidgetDataSource;
  refresh_rate?: number;
}

export interface DashboardLayout {
  columns: number;
  row_height: number;
  grid_gap: number;
  auto_layout: boolean;
}

export interface DashboardVariable {
  name: string;
  type: string;
  label: string;
  default_value?: string;
  options?: string[];
  query?: string;
}

export interface DashboardSettings {
  refresh_interval: number;
  time_range?: QueryTimeRange;
  theme?: string;
  variables?: DashboardVariable[];
}

export interface Dashboard {
  id?: string;
  name: string;
  description?: string;
  widgets: DashboardWidget[];
  layout: DashboardLayout;
  settings: DashboardSettings;
  shared_with?: string[];
  is_public: boolean;
  created_at?: string;
  updated_at?: string;
  created_by?: string;
}

export interface DashboardShare {
  id: string;
  dashboard_id: string;
  share_token: string;
  expires_at?: string;
  permissions: string[];
  created_at: string;
  created_by: string;
}

export interface ChartDataset {
  label: string;
  data: number[];
  background_color?: string;
  border_color?: string;
  fill?: boolean;
}

export interface ChartData {
  labels: string[];
  datasets: ChartDataset[];
  metadata?: Record<string, any>;
}

export interface MetricData {
  value: number;
  label: string;
  unit?: string;
  change?: number;
  change_label?: string;
  status?: string;
  metadata?: Record<string, any>;
}

export interface WidgetDataResponse {
  widget_id: string;
  type: string;
  data: ChartData | MetricData | Record<string, any>[];
}

// API Response Types
export interface ApiResponse<T> {
  data?: T;
  error?: string;
  message?: string;
}

export interface DashboardListResponse {
  dashboards: Dashboard[];
  count: number;
}

// Filter and Search Types
export const FILTER_OPERATORS = [
  { value: 'equals', label: 'Equals' },
  { value: 'not_equals', label: 'Not Equals' },
  { value: 'contains', label: 'Contains' },
  { value: 'not_contains', label: 'Not Contains' },
  { value: 'greater_than', label: 'Greater Than' },
  { value: 'less_than', label: 'Less Than' },
  { value: 'greater_equal', label: 'Greater or Equal' },
  { value: 'less_equal', label: 'Less or Equal' },
  { value: 'between', label: 'Between' },
  { value: 'in', label: 'In' },
  { value: 'not_in', label: 'Not In' },
  { value: 'is_null', label: 'Is Null' },
  { value: 'is_not_null', label: 'Is Not Null' },
] as const;

export const AGGREGATION_FUNCTIONS = [
  { value: 'COUNT', label: 'Count' },
  { value: 'COUNT_DISTINCT', label: 'Count Distinct' },
  { value: 'SUM', label: 'Sum' },
  { value: 'AVG', label: 'Average' },
  { value: 'MIN', label: 'Minimum' },
  { value: 'MAX', label: 'Maximum' },
] as const;

export const TIME_RANGES = [
  { value: 'last_1m', label: 'Last 1 minute' },
  { value: 'last_5m', label: 'Last 5 minutes' },
  { value: 'last_15m', label: 'Last 15 minutes' },
  { value: 'last_1h', label: 'Last 1 hour' },
  { value: 'last_24h', label: 'Last 24 hours' },
  { value: 'last_7d', label: 'Last 7 days' },
  { value: 'last_30d', label: 'Last 30 days' },
] as const;

export const CHART_TYPES = [
  { value: 'line', label: 'Line Chart' },
  { value: 'bar', label: 'Bar Chart' },
  { value: 'area', label: 'Area Chart' },
  { value: 'pie', label: 'Pie Chart' },
  { value: 'scatter', label: 'Scatter Plot' },
] as const;