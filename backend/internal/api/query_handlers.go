package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/your-username/click-lite-log-analytics/backend/internal/database"
	"github.com/your-username/click-lite-log-analytics/backend/internal/query"
)

// ExecuteQuery handles SQL query execution
func ExecuteQuery(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req query.QueryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Execute query
		response, err := db.ExecuteQuery(r.Context(), &req)
		if err != nil {
			log.Error().Err(err).Str("query", req.Query).Msg("Query execution failed")
			// Return error in response rather than HTTP error
			response.Error = err.Error()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// SaveQuery saves a query
func SaveQuery(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var savedQuery query.SavedQuery
		if err := json.NewDecoder(r.Body).Decode(&savedQuery); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// TODO: Get user from context
		savedQuery.CreatedBy = "user"

		queryEngine := db.GetQueryEngine()
		if queryEngine == nil {
			http.Error(w, "Query engine not available", http.StatusInternalServerError)
			return
		}

		queryStore := queryEngine.GetQueryStore()
		if err := queryStore.Save(&savedQuery); err != nil {
			log.Error().Err(err).Msg("Failed to save query")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(savedQuery)
	}
}

// GetQuery retrieves a saved query
func GetQuery(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queryID := chi.URLParam(r, "id")
		if queryID == "" {
			http.Error(w, "Query ID required", http.StatusBadRequest)
			return
		}

		queryEngine := db.GetQueryEngine()
		if queryEngine == nil {
			http.Error(w, "Query engine not available", http.StatusInternalServerError)
			return
		}

		queryStore := queryEngine.GetQueryStore()
		savedQuery, err := queryStore.Get(queryID)
		if err != nil {
			http.Error(w, "Query not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(savedQuery)
	}
}

// ListQueries lists saved queries
func ListQueries(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queryEngine := db.GetQueryEngine()
		if queryEngine == nil {
			http.Error(w, "Query engine not available", http.StatusInternalServerError)
			return
		}

		queryStore := queryEngine.GetQueryStore()
		
		// Apply filters from query params
		var filters []query.QueryFilter
		
		// Tag filter
		if tags := r.URL.Query().Get("tags"); tags != "" {
			tagList := strings.Split(tags, ",")
			filters = append(filters, query.TagFilter{Tags: tagList})
		}
		
		// Category filter
		if category := r.URL.Query().Get("category"); category != "" {
			filters = append(filters, query.CategoryFilter{Category: category})
		}
		
		// Template filter
		if templatesOnly := r.URL.Query().Get("templates_only"); templatesOnly == "true" {
			filters = append(filters, query.TemplateFilter{TemplatesOnly: true})
		}

		queries, err := queryStore.List(filters...)
		if err != nil {
			log.Error().Err(err).Msg("Failed to list queries")
			http.Error(w, "Failed to list queries", http.StatusInternalServerError)
			return
		}

		// Ensure queries is not nil
		if queries == nil {
			queries = []*query.SavedQuery{}
		}
		
		response := map[string]interface{}{
			"queries": queries,
			"count":   len(queries),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// UpdateQuery updates a saved query
func UpdateQuery(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queryID := chi.URLParam(r, "id")
		if queryID == "" {
			http.Error(w, "Query ID required", http.StatusBadRequest)
			return
		}

		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		queryEngine := db.GetQueryEngine()
		if queryEngine == nil {
			http.Error(w, "Query engine not available", http.StatusInternalServerError)
			return
		}

		queryStore := queryEngine.GetQueryStore()
		if err := queryStore.Update(queryID, updates); err != nil {
			log.Error().Err(err).Str("id", queryID).Msg("Failed to update query")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Return updated query
		updatedQuery, _ := queryStore.Get(queryID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updatedQuery)
	}
}

// DeleteQuery deletes a saved query
func DeleteQuery(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queryID := chi.URLParam(r, "id")
		if queryID == "" {
			http.Error(w, "Query ID required", http.StatusBadRequest)
			return
		}

		queryEngine := db.GetQueryEngine()
		if queryEngine == nil {
			http.Error(w, "Query engine not available", http.StatusInternalServerError)
			return
		}

		queryStore := queryEngine.GetQueryStore()
		if err := queryStore.Delete(queryID); err != nil {
			log.Error().Err(err).Str("id", queryID).Msg("Failed to delete query")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// ExecuteSavedQuery executes a saved query with parameters
func ExecuteSavedQuery(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queryID := chi.URLParam(r, "id")
		if queryID == "" {
			http.Error(w, "Query ID required", http.StatusBadRequest)
			return
		}

		queryEngine := db.GetQueryEngine()
		if queryEngine == nil {
			http.Error(w, "Query engine not available", http.StatusInternalServerError)
			return
		}

		// Get saved query
		queryStore := queryEngine.GetQueryStore()
		savedQuery, err := queryStore.Get(queryID)
		if err != nil {
			http.Error(w, "Query not found", http.StatusNotFound)
			return
		}

		// Parse parameters from request
		var params map[string]interface{}
		if r.Method == "POST" {
			if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
				http.Error(w, "Invalid parameters", http.StatusBadRequest)
				return
			}
		} else {
			// GET request - parse from query string
			params = make(map[string]interface{})
			for key, values := range r.URL.Query() {
				if len(values) > 0 {
					params[key] = values[0]
				}
			}
		}

		// Create query request
		req := &query.QueryRequest{
			Query:      savedQuery.Query,
			Parameters: params,
		}

		// Execute query
		response, err := db.ExecuteQuery(r.Context(), req)
		if err != nil {
			log.Error().Err(err).Str("query_id", queryID).Msg("Failed to execute saved query")
			response.Error = err.Error()
		}

		// Add query metadata to response
		response.Query = savedQuery.Name // Show query name instead of SQL

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}