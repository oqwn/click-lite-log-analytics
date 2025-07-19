package api

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/your-username/click-lite-log-analytics/backend/internal/database"
	"github.com/your-username/click-lite-log-analytics/backend/internal/models"
	"github.com/your-username/click-lite-log-analytics/backend/internal/query"
	"github.com/your-username/click-lite-log-analytics/backend/internal/querybuilder"
)

// GetAvailableFields returns the available fields for query building
func GetAvailableFields(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service := querybuilder.NewService()
		fields := service.GetAvailableFields()

		response := models.AvailableFields{
			Fields: fields,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GenerateSQL generates SQL from a query builder configuration
func GenerateSQL(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var qb models.QueryBuilder
		if err := json.NewDecoder(r.Body).Decode(&qb); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		service := querybuilder.NewService()

		// Validate query builder
		if err := service.ValidateQueryBuilder(&qb); err != nil {
			log.Error().Err(err).Msg("Query builder validation failed")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Generate SQL
		sql, err := service.GenerateSQL(&qb)
		if err != nil {
			log.Error().Err(err).Msg("SQL generation failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"sql":           sql,
			"query_builder": qb,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// ExecuteQueryBuilder executes a query builder configuration
func ExecuteQueryBuilder(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var qb models.QueryBuilder
		if err := json.NewDecoder(r.Body).Decode(&qb); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		service := querybuilder.NewService()

		// Validate query builder
		if err := service.ValidateQueryBuilder(&qb); err != nil {
			log.Error().Err(err).Msg("Query builder validation failed")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Generate SQL
		sql, err := service.GenerateSQL(&qb)
		if err != nil {
			log.Error().Err(err).Msg("SQL generation failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Execute query
		queryEngine := db.GetQueryEngine()
		if queryEngine == nil {
			http.Error(w, "Query engine not available", http.StatusInternalServerError)
			return
		}

		req := &query.QueryRequest{
			Query:   sql,
			Timeout: 30,
		}

		result, err := queryEngine.Execute(r.Context(), req)
		if err != nil {
			log.Error().Err(err).Str("sql", sql).Msg("Query execution failed")
			response := &models.QueryBuilderResponse{
				SQL:   sql,
				Error: err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// ValidateQueryBuilder validates a query builder configuration
func ValidateQueryBuilder(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var qb models.QueryBuilder
		if err := json.NewDecoder(r.Body).Decode(&qb); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		service := querybuilder.NewService()

		// Validate query builder
		err := service.ValidateQueryBuilder(&qb)
		
		response := map[string]interface{}{
			"valid": err == nil,
		}

		if err != nil {
			response["error"] = err.Error()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}