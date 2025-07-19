package query

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// SavedQuery represents a saved SQL query
type SavedQuery struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Query       string                 `json:"query"`
	Parameters  []QueryParameter       `json:"parameters,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by"`
	IsTemplate  bool                   `json:"is_template"`
	Category    string                 `json:"category,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// QueryParameter defines a parameter for a saved query
type QueryParameter struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // string, number, date, boolean
	Description  string      `json:"description,omitempty"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Required     bool        `json:"required"`
	Options      []string    `json:"options,omitempty"` // For enum-like parameters
}

// QueryStore manages saved queries
type QueryStore struct {
	queries map[string]*SavedQuery
	mu      sync.RWMutex
	storage StorageBackend
}

// StorageBackend interface for persistent storage
type StorageBackend interface {
	Save(query *SavedQuery) error
	Load(id string) (*SavedQuery, error)
	LoadAll() ([]*SavedQuery, error)
	Delete(id string) error
}

// InMemoryStorage is a simple in-memory storage backend
type InMemoryStorage struct {
	data map[string]*SavedQuery
	mu   sync.RWMutex
}

// NewInMemoryStorage creates a new in-memory storage
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		data: make(map[string]*SavedQuery),
	}
}

// Save saves a query to memory
func (s *InMemoryStorage) Save(query *SavedQuery) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[query.ID] = query
	return nil
}

// Load loads a query from memory
func (s *InMemoryStorage) Load(id string) (*SavedQuery, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query, exists := s.data[id]
	if !exists {
		return nil, fmt.Errorf("query not found: %s", id)
	}
	return query, nil
}

// LoadAll loads all queries from memory
func (s *InMemoryStorage) LoadAll() ([]*SavedQuery, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	queries := make([]*SavedQuery, 0, len(s.data))
	for _, query := range s.data {
		queries = append(queries, query)
	}
	return queries, nil
}

// Delete deletes a query from memory
func (s *InMemoryStorage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
	return nil
}

// NewQueryStore creates a new query store
func NewQueryStore() *QueryStore {
	store := &QueryStore{
		queries: make(map[string]*SavedQuery),
		storage: NewInMemoryStorage(), // Default to in-memory storage
	}
	
	// Load built-in templates
	store.loadBuiltInTemplates()
	
	return store
}

// SetStorage sets the storage backend
func (qs *QueryStore) SetStorage(storage StorageBackend) {
	qs.storage = storage
}

// Save saves a query
func (qs *QueryStore) Save(query *SavedQuery) error {
	if query.ID == "" {
		query.ID = uuid.New().String()
		query.CreatedAt = time.Now()
	}
	query.UpdatedAt = time.Now()
	
	// Validate query
	if err := qs.validateQuery(query); err != nil {
		return err
	}
	
	// Save to storage
	if err := qs.storage.Save(query); err != nil {
		return fmt.Errorf("failed to save query: %w", err)
	}
	
	// Update in-memory cache
	qs.mu.Lock()
	qs.queries[query.ID] = query
	qs.mu.Unlock()
	
	log.Info().Str("id", query.ID).Str("name", query.Name).Msg("Query saved")
	return nil
}

// Get retrieves a saved query by ID
func (qs *QueryStore) Get(id string) (*SavedQuery, error) {
	qs.mu.RLock()
	query, exists := qs.queries[id]
	qs.mu.RUnlock()
	
	if exists {
		return query, nil
	}
	
	// Try loading from storage
	query, err := qs.storage.Load(id)
	if err != nil {
		return nil, err
	}
	
	// Update cache
	qs.mu.Lock()
	qs.queries[id] = query
	qs.mu.Unlock()
	
	return query, nil
}

// List returns all saved queries
func (qs *QueryStore) List(filters ...QueryFilter) ([]*SavedQuery, error) {
	// Load all from storage
	allQueries, err := qs.storage.LoadAll()
	if err != nil {
		// Fall back to in-memory cache
		qs.mu.RLock()
		allQueries = make([]*SavedQuery, 0, len(qs.queries))
		for _, q := range qs.queries {
			allQueries = append(allQueries, q)
		}
		qs.mu.RUnlock()
	}
	
	// Apply filters
	filtered := allQueries
	for _, filter := range filters {
		filtered = filter.Apply(filtered)
	}
	
	return filtered, nil
}

// Update updates an existing query
func (qs *QueryStore) Update(id string, updates map[string]interface{}) error {
	query, err := qs.Get(id)
	if err != nil {
		return err
	}
	
	// Apply updates
	if name, ok := updates["name"].(string); ok {
		query.Name = name
	}
	if desc, ok := updates["description"].(string); ok {
		query.Description = desc
	}
	if q, ok := updates["query"].(string); ok {
		query.Query = q
	}
	if tags, ok := updates["tags"].([]string); ok {
		query.Tags = tags
	}
	if params, ok := updates["parameters"].([]QueryParameter); ok {
		query.Parameters = params
	}
	if category, ok := updates["category"].(string); ok {
		query.Category = category
	}
	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		query.Metadata = metadata
	}
	
	query.UpdatedAt = time.Now()
	
	return qs.Save(query)
}

// Delete deletes a saved query
func (qs *QueryStore) Delete(id string) error {
	// Don't allow deletion of built-in templates
	query, err := qs.Get(id)
	if err != nil {
		return err
	}
	
	if query.IsTemplate && query.CreatedBy == "system" {
		return fmt.Errorf("cannot delete built-in template")
	}
	
	// Delete from storage
	if err := qs.storage.Delete(id); err != nil {
		return fmt.Errorf("failed to delete query: %w", err)
	}
	
	// Remove from cache
	qs.mu.Lock()
	delete(qs.queries, id)
	qs.mu.Unlock()
	
	log.Info().Str("id", id).Msg("Query deleted")
	return nil
}

// validateQuery validates a saved query
func (qs *QueryStore) validateQuery(query *SavedQuery) error {
	if query.Name == "" {
		return fmt.Errorf("query name is required")
	}
	
	if query.Query == "" {
		return fmt.Errorf("query text is required")
	}
	
	// Validate parameter names
	paramNames := make(map[string]bool)
	for _, param := range query.Parameters {
		if param.Name == "" {
			return fmt.Errorf("parameter name is required")
		}
		
		if paramNames[param.Name] {
			return fmt.Errorf("duplicate parameter name: %s", param.Name)
		}
		paramNames[param.Name] = true
		
		// Validate parameter type
		validTypes := []string{"string", "number", "date", "boolean"}
		validType := false
		for _, t := range validTypes {
			if param.Type == t {
				validType = true
				break
			}
		}
		if !validType {
			return fmt.Errorf("invalid parameter type: %s", param.Type)
		}
	}
	
	return nil
}

// loadBuiltInTemplates loads built-in query templates
func (qs *QueryStore) loadBuiltInTemplates() {
	templates := []*SavedQuery{
		{
			ID:          "template-errors-by-service",
			Name:        "Errors by Service",
			Description: "Count of errors grouped by service",
			Query: `SELECT 
    service,
    COUNT() as error_count,
    COUNT(DISTINCT trace_id) as unique_requests
FROM logs
WHERE level = 'error'
    AND timestamp >= now() - INTERVAL :time_range HOUR
GROUP BY service
ORDER BY error_count DESC
LIMIT :limit`,
			Parameters: []QueryParameter{
				{
					Name:         "time_range",
					Type:         "number",
					Description:  "Time range in hours",
					DefaultValue: 24,
					Required:     true,
				},
				{
					Name:         "limit",
					Type:         "number",
					Description:  "Maximum number of services to return",
					DefaultValue: 20,
					Required:     false,
				},
			},
			Tags:       []string{"errors", "service", "aggregation"},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			CreatedBy:  "system",
			IsTemplate: true,
			Category:   "Error Analysis",
		},
		{
			ID:          "template-logs-by-level",
			Name:        "Log Level Distribution",
			Description: "Distribution of log levels over time",
			Query: `SELECT 
    toStartOfInterval(timestamp, INTERVAL :interval MINUTE) as time_bucket,
    level,
    COUNT() as count
FROM logs
WHERE timestamp >= now() - INTERVAL :time_range HOUR
GROUP BY time_bucket, level
ORDER BY time_bucket DESC, level`,
			Parameters: []QueryParameter{
				{
					Name:         "time_range",
					Type:         "number",
					Description:  "Time range in hours",
					DefaultValue: 6,
					Required:     true,
				},
				{
					Name:         "interval",
					Type:         "number",
					Description:  "Bucket interval in minutes",
					DefaultValue: 5,
					Required:     true,
				},
			},
			Tags:       []string{"levels", "time-series", "distribution"},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			CreatedBy:  "system",
			IsTemplate: true,
			Category:   "Overview",
		},
		{
			ID:          "template-slow-requests",
			Name:        "Slow Requests Analysis",
			Description: "Find requests with high response times",
			Query: `SELECT 
    timestamp,
    service,
    trace_id,
    message,
    CAST(attributes['response_time'] AS Float64) as response_time_ms
FROM logs
WHERE attributes['response_time'] != ''
    AND CAST(attributes['response_time'] AS Float64) > :threshold
    AND timestamp >= now() - INTERVAL :time_range HOUR
ORDER BY response_time_ms DESC
LIMIT :limit`,
			Parameters: []QueryParameter{
				{
					Name:         "threshold",
					Type:         "number",
					Description:  "Response time threshold in milliseconds",
					DefaultValue: 1000,
					Required:     true,
				},
				{
					Name:         "time_range",
					Type:         "number",
					Description:  "Time range in hours",
					DefaultValue: 1,
					Required:     true,
				},
				{
					Name:         "limit",
					Type:         "number",
					Description:  "Maximum results",
					DefaultValue: 100,
					Required:     false,
				},
			},
			Tags:       []string{"performance", "slow-queries", "analysis"},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			CreatedBy:  "system",
			IsTemplate: true,
			Category:   "Performance",
		},
		{
			ID:          "template-search-by-trace",
			Name:        "Search by Trace ID",
			Description: "Find all logs for a specific trace",
			Query: `SELECT 
    timestamp,
    level,
    service,
    message,
    span_id,
    attributes
FROM logs
WHERE trace_id = :trace_id
ORDER BY timestamp ASC`,
			Parameters: []QueryParameter{
				{
					Name:        "trace_id",
					Type:        "string",
					Description: "Trace ID to search for",
					Required:    true,
				},
			},
			Tags:       []string{"tracing", "debugging", "correlation"},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			CreatedBy:  "system",
			IsTemplate: true,
			Category:   "Tracing",
		},
	}
	
	// Save templates
	for _, template := range templates {
		qs.queries[template.ID] = template
		// Also save to storage
		if err := qs.storage.Save(template); err != nil {
			log.Error().Err(err).Str("template", template.Name).Msg("Failed to save built-in template")
		}
	}
}

// QueryFilter interface for filtering queries
type QueryFilter interface {
	Apply([]*SavedQuery) []*SavedQuery
}

// TagFilter filters queries by tags
type TagFilter struct {
	Tags []string
}

// Apply applies the tag filter
func (f TagFilter) Apply(queries []*SavedQuery) []*SavedQuery {
	if len(f.Tags) == 0 {
		return queries
	}
	
	filtered := []*SavedQuery{}
	for _, q := range queries {
		for _, tag := range f.Tags {
			found := false
			for _, qTag := range q.Tags {
				if qTag == tag {
					found = true
					break
				}
			}
			if found {
				filtered = append(filtered, q)
				break
			}
		}
	}
	
	return filtered
}

// CategoryFilter filters queries by category
type CategoryFilter struct {
	Category string
}

// Apply applies the category filter
func (f CategoryFilter) Apply(queries []*SavedQuery) []*SavedQuery {
	if f.Category == "" {
		return queries
	}
	
	filtered := []*SavedQuery{}
	for _, q := range queries {
		if q.Category == f.Category {
			filtered = append(filtered, q)
		}
	}
	
	return filtered
}

// TemplateFilter filters only templates or non-templates
type TemplateFilter struct {
	TemplatesOnly bool
}

// Apply applies the template filter
func (f TemplateFilter) Apply(queries []*SavedQuery) []*SavedQuery {
	filtered := []*SavedQuery{}
	for _, q := range queries {
		if q.IsTemplate == f.TemplatesOnly {
			filtered = append(filtered, q)
		}
	}
	
	return filtered
}