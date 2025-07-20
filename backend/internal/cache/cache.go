package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// Cache interface defines cache operations
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
	Size() int
}

// MemoryCache implements an in-memory cache with TTL support
type MemoryCache struct {
	mu      sync.RWMutex
	items   map[string]*CacheItem
	maxSize int
}

// CacheItem represents a cached item
type CacheItem struct {
	Value      interface{}
	Expiration time.Time
	AccessTime time.Time
	AccessCount int64
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache(maxSize int) *MemoryCache {
	cache := &MemoryCache{
		items:   make(map[string]*CacheItem),
		maxSize: maxSize,
	}
	
	// Start cleanup routine
	go cache.cleanupExpired()
	
	return cache
}

// Get retrieves a value from cache
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()
	
	if !exists {
		return nil, false
	}
	
	// Check expiration
	if time.Now().After(item.Expiration) {
		c.Delete(key)
		return nil, false
	}
	
	// Update access statistics
	c.mu.Lock()
	item.AccessTime = time.Now()
	item.AccessCount++
	c.mu.Unlock()
	
	return item.Value, true
}

// Set stores a value in cache with TTL
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Check cache size and evict if necessary
	if len(c.items) >= c.maxSize {
		c.evictLRU()
	}
	
	c.items[key] = &CacheItem{
		Value:       value,
		Expiration:  time.Now().Add(ttl),
		AccessTime:  time.Now(),
		AccessCount: 0,
	}
}

// Delete removes a key from cache
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear removes all items from cache
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*CacheItem)
}

// Size returns the number of items in cache
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// evictLRU removes the least recently used item
func (c *MemoryCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, item := range c.items {
		if oldestKey == "" || item.AccessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.AccessTime
		}
	}
	
	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// cleanupExpired removes expired items periodically
func (c *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.Expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// QueryCache wraps a cache for query results
type QueryCache struct {
	cache Cache
	ttl   time.Duration
}

// NewQueryCache creates a new query result cache
func NewQueryCache(cache Cache, ttl time.Duration) *QueryCache {
	return &QueryCache{
		cache: cache,
		ttl:   ttl,
	}
}

// GetQueryResult retrieves cached query result
func (qc *QueryCache) GetQueryResult(query string, params map[string]interface{}) (interface{}, bool) {
	key := qc.generateKey(query, params)
	return qc.cache.Get(key)
}

// SetQueryResult caches query result
func (qc *QueryCache) SetQueryResult(query string, params map[string]interface{}, result interface{}) {
	key := qc.generateKey(query, params)
	qc.cache.Set(key, result, qc.ttl)
}

// InvalidatePattern invalidates cache entries matching a pattern
func (qc *QueryCache) InvalidatePattern(pattern string) {
	// For now, clear all cache on invalidation
	// TODO: Implement pattern-based invalidation
	qc.cache.Clear()
}

// generateKey creates a cache key from query and parameters
func (qc *QueryCache) generateKey(query string, params map[string]interface{}) string {
	data := map[string]interface{}{
		"query":  query,
		"params": params,
	}
	
	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits       int64     `json:"hits"`
	Misses     int64     `json:"misses"`
	Evictions  int64     `json:"evictions"`
	Size       int       `json:"size"`
	MaxSize    int       `json:"max_size"`
	HitRate    float64   `json:"hit_rate"`
	LastReset  time.Time `json:"last_reset"`
}

// StatsCache wraps a cache with statistics tracking
type StatsCache struct {
	cache  Cache
	mu     sync.Mutex
	stats  CacheStats
}

// NewStatsCache creates a cache with statistics
func NewStatsCache(cache Cache, maxSize int) *StatsCache {
	return &StatsCache{
		cache: cache,
		stats: CacheStats{
			MaxSize:   maxSize,
			LastReset: time.Now(),
		},
	}
}

// Get retrieves value and updates statistics
func (sc *StatsCache) Get(key string) (interface{}, bool) {
	value, found := sc.cache.Get(key)
	
	sc.mu.Lock()
	if found {
		sc.stats.Hits++
	} else {
		sc.stats.Misses++
	}
	sc.updateHitRate()
	sc.mu.Unlock()
	
	return value, found
}

// Set stores value and updates statistics
func (sc *StatsCache) Set(key string, value interface{}, ttl time.Duration) {
	sc.cache.Set(key, value, ttl)
}

// Delete removes key from cache
func (sc *StatsCache) Delete(key string) {
	sc.cache.Delete(key)
	
	sc.mu.Lock()
	sc.stats.Evictions++
	sc.mu.Unlock()
}

// Clear removes all items from cache
func (sc *StatsCache) Clear() {
	sc.cache.Clear()
	
	sc.mu.Lock()
	sc.stats.Hits = 0
	sc.stats.Misses = 0
	sc.stats.Evictions = 0
	sc.stats.LastReset = time.Now()
	sc.mu.Unlock()
}

// Size returns cache size
func (sc *StatsCache) Size() int {
	return sc.cache.Size()
}

// GetStats returns cache statistics
func (sc *StatsCache) GetStats() CacheStats {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	
	sc.stats.Size = sc.cache.Size()
	return sc.stats
}

// updateHitRate calculates the cache hit rate
func (sc *StatsCache) updateHitRate() {
	total := sc.stats.Hits + sc.stats.Misses
	if total > 0 {
		sc.stats.HitRate = float64(sc.stats.Hits) / float64(total)
	}
}

// LayeredCache implements a multi-layer cache (L1, L2)
type LayeredCache struct {
	l1Cache Cache        // Fast, small cache
	l2Cache Cache        // Slower, larger cache
	l1TTL   time.Duration
	l2TTL   time.Duration
}

// NewLayeredCache creates a two-layer cache
func NewLayeredCache(l1Size, l2Size int, l1TTL, l2TTL time.Duration) *LayeredCache {
	return &LayeredCache{
		l1Cache: NewMemoryCache(l1Size),
		l2Cache: NewMemoryCache(l2Size),
		l1TTL:   l1TTL,
		l2TTL:   l2TTL,
	}
}

// Get retrieves from L1, then L2
func (lc *LayeredCache) Get(key string) (interface{}, bool) {
	// Check L1
	if value, found := lc.l1Cache.Get(key); found {
		return value, true
	}
	
	// Check L2
	if value, found := lc.l2Cache.Get(key); found {
		// Promote to L1
		lc.l1Cache.Set(key, value, lc.l1TTL)
		return value, true
	}
	
	return nil, false
}

// Set stores in both L1 and L2
func (lc *LayeredCache) Set(key string, value interface{}, ttl time.Duration) {
	lc.l1Cache.Set(key, value, lc.l1TTL)
	lc.l2Cache.Set(key, value, lc.l2TTL)
}

// Delete removes from both caches
func (lc *LayeredCache) Delete(key string) {
	lc.l1Cache.Delete(key)
	lc.l2Cache.Delete(key)
}

// Clear clears both caches
func (lc *LayeredCache) Clear() {
	lc.l1Cache.Clear()
	lc.l2Cache.Clear()
}

// Size returns combined size
func (lc *LayeredCache) Size() int {
	return lc.l1Cache.Size() + lc.l2Cache.Size()
}