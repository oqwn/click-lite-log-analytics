package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/your-username/click-lite-log-analytics/backend/internal/database"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
	"github.com/your-username/click-lite-log-analytics/backend/internal/query"
	"github.com/your-username/click-lite-log-analytics/backend/internal/querybuilder"
)

// Service handles dashboard operations
type Service struct {
	db              *database.DB
	queryBuilder    *querybuilder.Service
	dashboards      map[string]*models.Dashboard
	dashboardShares map[string]*models.DashboardShare
}

// NewService creates a new dashboard service
func NewService(db *database.DB) *Service {
	return &Service{
		db:              db,
		queryBuilder:    querybuilder.NewService(),
		dashboards:      make(map[string]*models.Dashboard),
		dashboardShares: make(map[string]*models.DashboardShare),
	}
}

// CreateDashboard creates a new dashboard
func (s *Service) CreateDashboard(ctx context.Context, dashboard *models.Dashboard, userID string) error {
	if dashboard.ID == "" {
		dashboard.ID = uuid.New().String()
	}

	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()
	dashboard.CreatedBy = userID

	// Validate dashboard
	if err := s.validateDashboard(dashboard); err != nil {
		return fmt.Errorf("dashboard validation failed: %w", err)
	}

	// Store in memory (in production, this would be in a database)
	s.dashboards[dashboard.ID] = dashboard

	log.Info().
		Str("dashboard_id", dashboard.ID).
		Str("name", dashboard.Name).
		Str("user_id", userID).
		Msg("Dashboard created")

	return nil
}

// GetDashboard retrieves a dashboard by ID
func (s *Service) GetDashboard(ctx context.Context, dashboardID string, userID string) (*models.Dashboard, error) {
	dashboard, exists := s.dashboards[dashboardID]
	if !exists {
		return nil, fmt.Errorf("dashboard not found: %s", dashboardID)
	}

	// Check permissions
	if !s.canAccessDashboard(dashboard, userID) {
		return nil, fmt.Errorf("access denied to dashboard: %s", dashboardID)
	}

	return dashboard, nil
}

// UpdateDashboard updates an existing dashboard
func (s *Service) UpdateDashboard(ctx context.Context, dashboardID string, updates map[string]interface{}, userID string) error {
	dashboard, exists := s.dashboards[dashboardID]
	if !exists {
		return fmt.Errorf("dashboard not found: %s", dashboardID)
	}

	// Check permissions
	if !s.canEditDashboard(dashboard, userID) {
		return fmt.Errorf("edit access denied to dashboard: %s", dashboardID)
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		dashboard.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		dashboard.Description = description
	}
	if widgets, ok := updates["widgets"]; ok {
		if widgetsData, err := json.Marshal(widgets); err == nil {
			var newWidgets []models.DashboardWidget
			if err := json.Unmarshal(widgetsData, &newWidgets); err == nil {
				dashboard.Widgets = newWidgets
			}
		}
	}
	if layout, ok := updates["layout"]; ok {
		if layoutData, err := json.Marshal(layout); err == nil {
			var newLayout models.DashboardLayout
			if err := json.Unmarshal(layoutData, &newLayout); err == nil {
				dashboard.Layout = newLayout
			}
		}
	}

	dashboard.UpdatedAt = time.Now()

	log.Info().
		Str("dashboard_id", dashboardID).
		Str("user_id", userID).
		Msg("Dashboard updated")

	return nil
}

// DeleteDashboard deletes a dashboard
func (s *Service) DeleteDashboard(ctx context.Context, dashboardID string, userID string) error {
	dashboard, exists := s.dashboards[dashboardID]
	if !exists {
		return fmt.Errorf("dashboard not found: %s", dashboardID)
	}

	// Check permissions
	if dashboard.CreatedBy != userID {
		return fmt.Errorf("delete access denied to dashboard: %s", dashboardID)
	}

	delete(s.dashboards, dashboardID)

	log.Info().
		Str("dashboard_id", dashboardID).
		Str("user_id", userID).
		Msg("Dashboard deleted")

	return nil
}

// ListDashboards lists dashboards accessible to a user
func (s *Service) ListDashboards(ctx context.Context, userID string) ([]*models.Dashboard, error) {
	var dashboards []*models.Dashboard

	for _, dashboard := range s.dashboards {
		if s.canAccessDashboard(dashboard, userID) {
			dashboards = append(dashboards, dashboard)
		}
	}

	return dashboards, nil
}

// ExecuteWidgetQuery executes a query for a specific widget
func (s *Service) ExecuteWidgetQuery(ctx context.Context, widget *models.DashboardWidget) (*models.QueryBuilderResponse, error) {
	var sql string
	var err error

	switch widget.DataSource.Type {
	case "query_builder":
		if widget.DataSource.QueryBuilder != nil {
			sql, err = s.queryBuilder.GenerateSQL(widget.DataSource.QueryBuilder)
			if err != nil {
				return nil, fmt.Errorf("failed to generate SQL from query builder: %w", err)
			}
		} else {
			return nil, fmt.Errorf("query builder configuration missing")
		}

	case "saved_query":
		// Get saved query from query engine
		queryEngine := s.db.GetQueryEngine()
		if queryEngine == nil {
			return nil, fmt.Errorf("query engine not available")
		}

		savedQuery, err := queryEngine.GetQueryStore().Get(widget.DataSource.QueryID)
		if err != nil {
			return nil, fmt.Errorf("failed to get saved query: %w", err)
		}
		sql = savedQuery.Query

	case "custom_sql":
		sql = widget.DataSource.SQL

	default:
		return nil, fmt.Errorf("unsupported data source type: %s", widget.DataSource.Type)
	}

	// Execute the query
	queryEngine := s.db.GetQueryEngine()
	if queryEngine == nil {
		return nil, fmt.Errorf("query engine not available")
	}

	req := &query.QueryRequest{
		Query:      sql,
		Parameters: widget.DataSource.Parameters,
		Timeout:    30, // 30 seconds
	}

	result, err := queryEngine.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	// Convert to QueryBuilderResponse
	response := &models.QueryBuilderResponse{
		SQL:           sql,
		Rows:          result.Rows,
		RowCount:      result.RowCount,
		ExecutionTime: result.ExecutionTime,
		Error:         result.Error,
	}

	// Convert columns
	for _, col := range result.Columns {
		response.Columns = append(response.Columns, models.QueryResultColumn{
			Name: col.Name,
			Type: col.Type,
		})
	}

	return response, nil
}

// GenerateWidgetData generates chart data for a widget
func (s *Service) GenerateWidgetData(ctx context.Context, widget *models.DashboardWidget) (interface{}, error) {
	queryResult, err := s.ExecuteWidgetQuery(ctx, widget)
	if err != nil {
		return nil, err
	}

	if queryResult.Error != "" {
		return nil, fmt.Errorf("query error: %s", queryResult.Error)
	}

	switch widget.Type {
	case "chart":
		return s.generateChartData(widget, queryResult)
	case "metric":
		return s.generateMetricData(widget, queryResult)
	case "table":
		return queryResult.Rows, nil
	default:
		return queryResult.Rows, nil
	}
}

// ShareDashboard creates a share link for a dashboard
func (s *Service) ShareDashboard(ctx context.Context, dashboardID string, permissions []string, expiresAt *time.Time, userID string) (*models.DashboardShare, error) {
	dashboard, exists := s.dashboards[dashboardID]
	if !exists {
		return nil, fmt.Errorf("dashboard not found: %s", dashboardID)
	}

	// Check permissions
	if dashboard.CreatedBy != userID {
		return nil, fmt.Errorf("share access denied to dashboard: %s", dashboardID)
	}

	share := &models.DashboardShare{
		ID:          uuid.New().String(),
		DashboardID: dashboardID,
		ShareToken:  uuid.New().String(),
		ExpiresAt:   expiresAt,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		CreatedBy:   userID,
	}

	s.dashboardShares[share.ShareToken] = share

	return share, nil
}

// GetDashboardByShareToken retrieves a dashboard by share token
func (s *Service) GetDashboardByShareToken(ctx context.Context, shareToken string) (*models.Dashboard, error) {
	share, exists := s.dashboardShares[shareToken]
	if !exists {
		return nil, fmt.Errorf("invalid share token")
	}

	// Check expiration
	if share.ExpiresAt != nil && time.Now().After(*share.ExpiresAt) {
		return nil, fmt.Errorf("share link has expired")
	}

	dashboard, exists := s.dashboards[share.DashboardID]
	if !exists {
		return nil, fmt.Errorf("dashboard not found")
	}

	return dashboard, nil
}

// Helper methods

func (s *Service) validateDashboard(dashboard *models.Dashboard) error {
	if dashboard.Name == "" {
		return fmt.Errorf("dashboard name is required")
	}

	for _, widget := range dashboard.Widgets {
		if err := s.validateWidget(&widget); err != nil {
			return fmt.Errorf("widget validation failed: %w", err)
		}
	}

	return nil
}

func (s *Service) validateWidget(widget *models.DashboardWidget) error {
	if widget.Title == "" {
		return fmt.Errorf("widget title is required")
	}

	validTypes := []string{"chart", "table", "metric", "text"}
	validType := false
	for _, t := range validTypes {
		if widget.Type == t {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("invalid widget type: %s", widget.Type)
	}

	return nil
}

func (s *Service) canAccessDashboard(dashboard *models.Dashboard, userID string) bool {
	// Owner can always access
	if dashboard.CreatedBy == userID {
		return true
	}

	// Check if dashboard is public
	if dashboard.IsPublic {
		return true
	}

	// Check if user is in shared list
	for _, sharedUser := range dashboard.SharedWith {
		if sharedUser == userID {
			return true
		}
	}

	return false
}

func (s *Service) canEditDashboard(dashboard *models.Dashboard, userID string) bool {
	// Only owner can edit
	return dashboard.CreatedBy == userID
}

func (s *Service) generateChartData(widget *models.DashboardWidget, queryResult *models.QueryBuilderResponse) (*models.ChartData, error) {
	if len(queryResult.Rows) == 0 {
		return &models.ChartData{
			Labels:   []string{},
			Datasets: []models.ChartDataset{},
		}, nil
	}

	// Simple chart data generation - assumes first column is labels, rest are data
	var labels []string
	var dataValues []float64

	for _, row := range queryResult.Rows {
		// Get first column as label
		for _, value := range row {
			labels = append(labels, fmt.Sprintf("%v", value))
			break
		}

		// Get second column as data (if exists)
		i := 0
		for _, value := range row {
			if i == 1 {
				if numValue, ok := value.(float64); ok {
					dataValues = append(dataValues, numValue)
				} else if intValue, ok := value.(int64); ok {
					dataValues = append(dataValues, float64(intValue))
				} else {
					dataValues = append(dataValues, 0)
				}
				break
			}
			i++
		}
	}

	chartData := &models.ChartData{
		Labels: labels,
		Datasets: []models.ChartDataset{
			{
				Label: widget.Title,
				Data:  dataValues,
				BackgroundColor: "rgba(54, 162, 235, 0.2)",
				BorderColor:     "rgba(54, 162, 235, 1)",
			},
		},
	}

	return chartData, nil
}

func (s *Service) generateMetricData(widget *models.DashboardWidget, queryResult *models.QueryBuilderResponse) (*models.MetricData, error) {
	if len(queryResult.Rows) == 0 {
		return &models.MetricData{
			Value: 0,
			Label: widget.Title,
		}, nil
	}

	// Get first value from first row
	var value float64
	for _, v := range queryResult.Rows[0] {
		if numValue, ok := v.(float64); ok {
			value = numValue
		} else if intValue, ok := v.(int64); ok {
			value = float64(intValue)
		}
		break
	}

	return &models.MetricData{
		Value: value,
		Label: widget.Title,
	}, nil
}