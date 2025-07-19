package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/your-username/click-lite-log-analytics/backend/internal/dashboard"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
)

// CreateDashboard creates a new dashboard
func CreateDashboard(service *dashboard.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var dashboardReq models.Dashboard
		if err := json.NewDecoder(r.Body).Decode(&dashboardReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		userID := getUserID(r) // TODO: Get from authentication context

		if err := service.CreateDashboard(r.Context(), &dashboardReq, userID); err != nil {
			log.Error().Err(err).Msg("Failed to create dashboard")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dashboardReq)
	}
}

// GetDashboard retrieves a dashboard by ID
func GetDashboard(service *dashboard.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dashboardID := chi.URLParam(r, "id")
		if dashboardID == "" {
			http.Error(w, "Dashboard ID required", http.StatusBadRequest)
			return
		}
		userID := getUserID(r)

		dashboard, err := service.GetDashboard(r.Context(), dashboardID, userID)
		if err != nil {
			log.Error().Err(err).Str("dashboard_id", dashboardID).Msg("Failed to get dashboard")
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dashboard)
	}
}

// UpdateDashboard updates an existing dashboard
func UpdateDashboard(service *dashboard.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dashboardID := chi.URLParam(r, "id")
		if dashboardID == "" {
			http.Error(w, "Dashboard ID required", http.StatusBadRequest)
			return
		}

		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		userID := getUserID(r)

		if err := service.UpdateDashboard(r.Context(), dashboardID, updates, userID); err != nil {
			log.Error().Err(err).Str("dashboard_id", dashboardID).Msg("Failed to update dashboard")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Return updated dashboard
		updatedDashboard, _ := service.GetDashboard(r.Context(), dashboardID, userID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updatedDashboard)
	}
}

// DeleteDashboard deletes a dashboard
func DeleteDashboard(service *dashboard.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dashboardID := chi.URLParam(r, "id")
		if dashboardID == "" {
			http.Error(w, "Dashboard ID required", http.StatusBadRequest)
			return
		}

		userID := getUserID(r)

		if err := service.DeleteDashboard(r.Context(), dashboardID, userID); err != nil {
			log.Error().Err(err).Str("dashboard_id", dashboardID).Msg("Failed to delete dashboard")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ListDashboards lists dashboards accessible to the user
func ListDashboards(service *dashboard.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)

		dashboards, err := service.ListDashboards(r.Context(), userID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to list dashboards")
			http.Error(w, "Failed to list dashboards", http.StatusInternalServerError)
			return
		}

		if dashboards == nil {
			dashboards = []*models.Dashboard{}
		}

		response := map[string]interface{}{
			"dashboards": dashboards,
			"count":      len(dashboards),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// ExecuteWidgetQuery executes a query for a specific widget
func ExecuteWidgetQuery(service *dashboard.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dashboardID := chi.URLParam(r, "dashboard_id")
		widgetID := chi.URLParam(r, "widget_id")

		if dashboardID == "" || widgetID == "" {
			http.Error(w, "Dashboard ID and Widget ID required", http.StatusBadRequest)
			return
		}

		userID := getUserID(r)

		// Get dashboard
		dashboardObj, err := service.GetDashboard(r.Context(), dashboardID, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Find widget
		var targetWidget *models.DashboardWidget
		for _, widget := range dashboardObj.Widgets {
			if widget.ID == widgetID {
				targetWidget = &widget
				break
			}
		}

		if targetWidget == nil {
			http.Error(w, "Widget not found", http.StatusNotFound)
			return
		}

		// Execute widget query
		result, err := service.ExecuteWidgetQuery(r.Context(), targetWidget)
		if err != nil {
			log.Error().Err(err).
				Str("dashboard_id", dashboardID).
				Str("widget_id", widgetID).
				Msg("Failed to execute widget query")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// GetWidgetData gets processed data for a widget (chart data, metrics, etc.)
func GetWidgetData(service *dashboard.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dashboardID := chi.URLParam(r, "dashboard_id")
		widgetID := chi.URLParam(r, "widget_id")

		if dashboardID == "" || widgetID == "" {
			http.Error(w, "Dashboard ID and Widget ID required", http.StatusBadRequest)
			return
		}

		userID := getUserID(r)

		// Get dashboard
		dashboardObj, err := service.GetDashboard(r.Context(), dashboardID, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Find widget
		var targetWidget *models.DashboardWidget
		for _, widget := range dashboardObj.Widgets {
			if widget.ID == widgetID {
				targetWidget = &widget
				break
			}
		}

		if targetWidget == nil {
			http.Error(w, "Widget not found", http.StatusNotFound)
			return
		}

		// Generate widget data
		data, err := service.GenerateWidgetData(r.Context(), targetWidget)
		if err != nil {
			log.Error().Err(err).
				Str("dashboard_id", dashboardID).
				Str("widget_id", widgetID).
				Msg("Failed to generate widget data")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"widget_id": widgetID,
			"type":      targetWidget.Type,
			"data":      data,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// ShareDashboard creates a share link for a dashboard
func ShareDashboard(service *dashboard.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dashboardID := chi.URLParam(r, "id")
		if dashboardID == "" {
			http.Error(w, "Dashboard ID required", http.StatusBadRequest)
			return
		}

		var shareReq struct {
			Permissions []string `json:"permissions"`
			ExpiresAt   *string  `json:"expires_at,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&shareReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		userID := getUserID(r)

		// Parse expiration time if provided
		var expiresAt *time.Time
		if shareReq.ExpiresAt != nil {
			if t, err := time.Parse(time.RFC3339, *shareReq.ExpiresAt); err == nil {
				expiresAt = &t
			}
		}

		share, err := service.ShareDashboard(r.Context(), dashboardID, shareReq.Permissions, expiresAt, userID)
		if err != nil {
			log.Error().Err(err).Str("dashboard_id", dashboardID).Msg("Failed to share dashboard")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(share)
	}
}

// GetSharedDashboard retrieves a dashboard by share token
func GetSharedDashboard(service *dashboard.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shareToken := chi.URLParam(r, "token")
		if shareToken == "" {
			http.Error(w, "Share token required", http.StatusBadRequest)
			return
		}


		dashboard, err := service.GetDashboardByShareToken(r.Context(), shareToken)
		if err != nil {
			log.Error().Err(err).Str("share_token", shareToken).Msg("Failed to get shared dashboard")
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dashboard)
	}
}

// getUserID extracts user ID from request context
// TODO: Implement proper authentication and extract from JWT/session
func getUserID(r *http.Request) string {
	// For now, return a default user ID
	// In production, this would extract from JWT token or session
	return "default-user"
}