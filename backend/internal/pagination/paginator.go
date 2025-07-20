package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Paginator handles query result pagination
type Paginator struct {
	DefaultPageSize int
	MaxPageSize     int
}

// PageRequest represents pagination parameters
type PageRequest struct {
	PageSize  int    `json:"page_size"`
	PageToken string `json:"page_token,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

// PageResponse represents paginated results
type PageResponse struct {
	Data          interface{} `json:"data"`
	NextPageToken string      `json:"next_page_token,omitempty"`
	PrevPageToken string      `json:"prev_page_token,omitempty"`
	TotalCount    int64       `json:"total_count,omitempty"`
	PageSize      int         `json:"page_size"`
	HasMore       bool        `json:"has_more"`
}

// CursorToken represents pagination cursor information
type CursorToken struct {
	Offset    int               `json:"offset"`
	SortValue interface{}       `json:"sort_value,omitempty"`
	Filters   map[string]string `json:"filters,omitempty"`
}

// NewPaginator creates a new paginator
func NewPaginator(defaultSize, maxSize int) *Paginator {
	return &Paginator{
		DefaultPageSize: defaultSize,
		MaxPageSize:     maxSize,
	}
}

// ValidateRequest validates and normalizes pagination request
func (p *Paginator) ValidateRequest(req *PageRequest) error {
	// Set defaults
	if req.PageSize <= 0 {
		req.PageSize = p.DefaultPageSize
	}
	
	// Enforce max page size
	if req.PageSize > p.MaxPageSize {
		req.PageSize = p.MaxPageSize
	}
	
	// Set default sort order
	if req.SortOrder == "" {
		req.SortOrder = "DESC"
	} else {
		req.SortOrder = strings.ToUpper(req.SortOrder)
		if req.SortOrder != "ASC" && req.SortOrder != "DESC" {
			return fmt.Errorf("invalid sort order: %s", req.SortOrder)
		}
	}
	
	// Validate page token
	if req.PageToken != "" {
		if _, err := p.DecodeToken(req.PageToken); err != nil {
			return fmt.Errorf("invalid page token: %w", err)
		}
	}
	
	return nil
}

// ApplyPagination adds pagination to SQL query
func (p *Paginator) ApplyPagination(query string, req PageRequest) (string, error) {
	var offset int
	
	// Decode page token if provided
	if req.PageToken != "" {
		token, err := p.DecodeToken(req.PageToken)
		if err != nil {
			return "", err
		}
		offset = token.Offset
	}
	
	// Remove existing LIMIT/OFFSET if present
	query = p.stripPagination(query)
	
	// Add ORDER BY if not present
	if req.SortBy != "" && !strings.Contains(strings.ToUpper(query), "ORDER BY") {
		query += fmt.Sprintf(" ORDER BY %s %s", req.SortBy, req.SortOrder)
	}
	
	// Add LIMIT and OFFSET
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", req.PageSize+1, offset)
	
	return query, nil
}

// CreateResponse creates paginated response from results
func (p *Paginator) CreateResponse(results []interface{}, req PageRequest, totalCount int64) *PageResponse {
	hasMore := len(results) > req.PageSize
	
	// Trim results to requested page size
	if hasMore {
		results = results[:req.PageSize]
	}
	
	response := &PageResponse{
		Data:       results,
		PageSize:   req.PageSize,
		HasMore:    hasMore,
		TotalCount: totalCount,
	}
	
	// Create next page token if there are more results
	if hasMore {
		nextOffset := 0
		if req.PageToken != "" {
			token, _ := p.DecodeToken(req.PageToken)
			nextOffset = token.Offset
		}
		nextOffset += req.PageSize
		
		nextToken := &CursorToken{
			Offset: nextOffset,
		}
		response.NextPageToken = p.EncodeToken(nextToken)
	}
	
	// Create previous page token if not on first page
	if req.PageToken != "" {
		token, _ := p.DecodeToken(req.PageToken)
		if token.Offset > 0 {
			prevOffset := token.Offset - req.PageSize
			if prevOffset < 0 {
				prevOffset = 0
			}
			
			prevToken := &CursorToken{
				Offset: prevOffset,
			}
			response.PrevPageToken = p.EncodeToken(prevToken)
		}
	}
	
	return response
}

// EncodeToken encodes cursor token to string
func (p *Paginator) EncodeToken(token *CursorToken) string {
	data, _ := json.Marshal(token)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeToken decodes cursor token from string
func (p *Paginator) DecodeToken(tokenStr string) (*CursorToken, error) {
	data, err := base64.URLEncoding.DecodeString(tokenStr)
	if err != nil {
		return nil, err
	}
	
	var token CursorToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	
	return &token, nil
}

// stripPagination removes existing LIMIT/OFFSET from query
func (p *Paginator) stripPagination(query string) string {
	// Remove LIMIT clause
	limitRegex := `\s+LIMIT\s+\d+`
	query = regexp.MustCompile(limitRegex).ReplaceAllString(query, "")
	
	// Remove OFFSET clause
	offsetRegex := `\s+OFFSET\s+\d+`
	query = regexp.MustCompile(offsetRegex).ReplaceAllString(query, "")
	
	return strings.TrimSpace(query)
}

// KeysetPaginator implements keyset pagination for better performance
type KeysetPaginator struct {
	*Paginator
}

// NewKeysetPaginator creates a keyset paginator
func NewKeysetPaginator(defaultSize, maxSize int) *KeysetPaginator {
	return &KeysetPaginator{
		Paginator: NewPaginator(defaultSize, maxSize),
	}
}

// ApplyKeysetPagination applies keyset pagination to query
func (kp *KeysetPaginator) ApplyKeysetPagination(query string, req PageRequest, keyField string) (string, error) {
	// Decode cursor if provided
	var lastValue interface{}
	if req.PageToken != "" {
		token, err := kp.DecodeToken(req.PageToken)
		if err != nil {
			return "", err
		}
		lastValue = token.SortValue
	}
	
	// Remove existing pagination
	query = kp.stripPagination(query)
	
	// Add keyset condition if we have a cursor
	if lastValue != nil {
		whereClause := fmt.Sprintf("%s %s '%v'", 
			keyField, 
			map[string]string{"ASC": ">", "DESC": "<"}[req.SortOrder],
			lastValue)
		
		if strings.Contains(strings.ToUpper(query), "WHERE") {
			query = strings.Replace(query, "WHERE", fmt.Sprintf("WHERE %s AND", whereClause), 1)
		} else {
			// Find insertion point before ORDER BY, GROUP BY, etc.
			insertPoint := len(query)
			for _, keyword := range []string{"ORDER BY", "GROUP BY", "HAVING"} {
				if idx := strings.Index(strings.ToUpper(query), keyword); idx != -1 && idx < insertPoint {
					insertPoint = idx
				}
			}
			query = query[:insertPoint] + " WHERE " + whereClause + " " + query[insertPoint:]
		}
	}
	
	// Add ORDER BY if not present
	if !strings.Contains(strings.ToUpper(query), "ORDER BY") {
		query += fmt.Sprintf(" ORDER BY %s %s", keyField, req.SortOrder)
	}
	
	// Add LIMIT
	query += fmt.Sprintf(" LIMIT %d", req.PageSize+1)
	
	return query, nil
}

// CreateKeysetResponse creates response with keyset cursor
func (kp *KeysetPaginator) CreateKeysetResponse(results []map[string]interface{}, req PageRequest, keyField string) *PageResponse {
	hasMore := len(results) > req.PageSize
	
	// Trim results to requested page size
	if hasMore {
		results = results[:req.PageSize]
	}
	
	response := &PageResponse{
		Data:     results,
		PageSize: req.PageSize,
		HasMore:  hasMore,
	}
	
	// Create next page token if there are more results
	if hasMore && len(results) > 0 {
		lastResult := results[len(results)-1]
		if lastValue, ok := lastResult[keyField]; ok {
			nextToken := &CursorToken{
				SortValue: lastValue,
			}
			response.NextPageToken = kp.EncodeToken(nextToken)
		}
	}
	
	return response
}

// StreamingPaginator implements server-side streaming for large datasets
type StreamingPaginator struct {
	ChunkSize int
}

// NewStreamingPaginator creates a streaming paginator
func NewStreamingPaginator(chunkSize int) *StreamingPaginator {
	return &StreamingPaginator{
		ChunkSize: chunkSize,
	}
}

// StreamResults streams results in chunks
func (sp *StreamingPaginator) StreamResults(query string, writer func(chunk []interface{}) error) error {
	offset := 0
	
	for {
		// This would execute the query and get results
		// For now, we'll assume results are provided
		var results []interface{}
		
		if len(results) == 0 {
			break
		}
		
		// Write chunk
		if err := writer(results); err != nil {
			return err
		}
		
		// Check if we got less than chunk size (last chunk)
		if len(results) < sp.ChunkSize {
			break
		}
		
		offset += sp.ChunkSize
	}
	
	return nil
}