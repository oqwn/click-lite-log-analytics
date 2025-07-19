package models

import (
	"time"
)

// Dashboard represents a dashboard configuration
type Dashboard struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Widgets     []DashboardWidget `json:"widgets"`
	Layout      DashboardLayout   `json:"layout"`
	Settings    DashboardSettings `json:"settings"`
	SharedWith  []string          `json:"shared_with,omitempty"`
	IsPublic    bool              `json:"is_public"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	CreatedBy   string            `json:"created_by"`
}

// DashboardWidget represents a widget on the dashboard
type DashboardWidget struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"` // chart, table, metric, text
	Title      string            `json:"title"`
	Position   WidgetPosition    `json:"position"`
	Size       WidgetSize        `json:"size"`
	Config     WidgetConfig      `json:"config"`
	DataSource WidgetDataSource  `json:"data_source"`
	RefreshRate int              `json:"refresh_rate,omitempty"` // seconds, 0 = no auto-refresh
}

// WidgetPosition represents widget position on the dashboard
type WidgetPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// WidgetSize represents widget dimensions
type WidgetSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// WidgetConfig contains widget-specific configuration
type WidgetConfig struct {
	ChartType     string                 `json:"chart_type,omitempty"` // line, bar, pie, area, scatter
	XAxis         AxisConfig             `json:"x_axis,omitempty"`
	YAxis         AxisConfig             `json:"y_axis,omitempty"`
	Colors        []string               `json:"colors,omitempty"`
	ShowLegend    bool                   `json:"show_legend"`
	ShowGrid      bool                   `json:"show_grid"`
	Stacked       bool                   `json:"stacked,omitempty"`
	TimeFormat    string                 `json:"time_format,omitempty"`
	ValueFormat   string                 `json:"value_format,omitempty"`
	Threshold     *ThresholdConfig       `json:"threshold,omitempty"`
	CustomOptions map[string]interface{} `json:"custom_options,omitempty"`
}

// AxisConfig represents chart axis configuration
type AxisConfig struct {
	Label     string `json:"label,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	LogScale  bool   `json:"log_scale,omitempty"`
}

// ThresholdConfig represents threshold lines/alerts
type ThresholdConfig struct {
	Value     float64 `json:"value"`
	Color     string  `json:"color"`
	Label     string  `json:"label,omitempty"`
	ShowLine  bool    `json:"show_line"`
}

// WidgetDataSource represents the data source for a widget
type WidgetDataSource struct {
	Type         string                 `json:"type"` // query_builder, saved_query, custom_sql
	QueryID      string                 `json:"query_id,omitempty"`
	SQL          string                 `json:"sql,omitempty"`
	QueryBuilder *QueryBuilder          `json:"query_builder,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
}

// DashboardLayout represents overall dashboard layout
type DashboardLayout struct {
	Columns    int    `json:"columns"`
	RowHeight  int    `json:"row_height"`
	GridGap    int    `json:"grid_gap"`
	AutoLayout bool   `json:"auto_layout"`
}

// DashboardSettings represents dashboard-wide settings
type DashboardSettings struct {
	RefreshInterval int               `json:"refresh_interval"` // seconds
	TimeRange       *QueryTimeRange   `json:"time_range,omitempty"`
	Theme           string            `json:"theme,omitempty"` // light, dark
	Variables       []DashboardVariable `json:"variables,omitempty"`
}

// DashboardVariable represents a dashboard variable
type DashboardVariable struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"` // text, select, multi_select, time_range
	Label        string   `json:"label"`
	DefaultValue string   `json:"default_value,omitempty"`
	Options      []string `json:"options,omitempty"`
	Query        string   `json:"query,omitempty"` // for dynamic options
}

// DashboardShare represents sharing configuration
type DashboardShare struct {
	ID           string    `json:"id"`
	DashboardID  string    `json:"dashboard_id"`
	ShareToken   string    `json:"share_token"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Permissions  []string  `json:"permissions"` // view, edit
	CreatedAt    time.Time `json:"created_at"`
	CreatedBy    string    `json:"created_by"`
}

// ChartData represents data for chart widgets
type ChartData struct {
	Labels   []string                 `json:"labels"`
	Datasets []ChartDataset           `json:"datasets"`
	Metadata map[string]interface{}   `json:"metadata,omitempty"`
}

// ChartDataset represents a data series
type ChartDataset struct {
	Label           string    `json:"label"`
	Data            []float64 `json:"data"`
	BackgroundColor string    `json:"background_color,omitempty"`
	BorderColor     string    `json:"border_color,omitempty"`
	Fill            bool      `json:"fill,omitempty"`
}

// MetricData represents data for metric widgets
type MetricData struct {
	Value       float64                `json:"value"`
	Label       string                 `json:"label"`
	Unit        string                 `json:"unit,omitempty"`
	Change      *float64               `json:"change,omitempty"`
	ChangeLabel string                 `json:"change_label,omitempty"`
	Status      string                 `json:"status,omitempty"` // normal, warning, critical
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}