package cluster

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

// Coordinator manages distributed cluster coordination
type Coordinator struct {
	nodes           []Node
	nodesMu         sync.RWMutex
	loadBalancer    LoadBalancer
	shardingStrategy ShardingStrategy
	healthChecker   *HealthChecker
	config          ClusterConfig
}

// Node represents a cluster node
type Node struct {
	ID              string
	Address         string
	Status          NodeStatus
	LastHealthCheck time.Time
	Load            float64
	Shards          []int
	Metadata        map[string]string
}

// NodeStatus represents node health status
type NodeStatus string

const (
	NodeStatusHealthy   NodeStatus = "healthy"
	NodeStatusDegraded  NodeStatus = "degraded"
	NodeStatusUnhealthy NodeStatus = "unhealthy"
)

// ClusterConfig configures cluster behavior
type ClusterConfig struct {
	ReplicationFactor   int
	ShardCount          int
	HealthCheckInterval time.Duration
	FailoverTimeout     time.Duration
	LoadBalancingPolicy string
}

// LoadBalancer interface for load balancing strategies
type LoadBalancer interface {
	SelectNode(nodes []Node, key string) (*Node, error)
	UpdateLoad(nodeID string, load float64)
}

// ShardingStrategy interface for data sharding
type ShardingStrategy interface {
	GetShard(key string, shardCount int) int
	GetNodesForShard(shard int, nodes []Node) []Node
}

// NewCoordinator creates a new cluster coordinator
func NewCoordinator(config ClusterConfig) *Coordinator {
	coordinator := &Coordinator{
		nodes:    []Node{},
		config:   config,
	}
	
	// Initialize load balancer
	switch config.LoadBalancingPolicy {
	case "least_loaded":
		coordinator.loadBalancer = NewLeastLoadedBalancer()
	case "consistent_hash":
		coordinator.loadBalancer = NewConsistentHashBalancer()
	default:
		coordinator.loadBalancer = NewRoundRobinBalancer()
	}
	
	// Initialize sharding strategy
	coordinator.shardingStrategy = NewHashSharding()
	
	// Initialize health checker
	coordinator.healthChecker = NewHealthChecker(config.HealthCheckInterval)
	
	return coordinator
}

// RegisterNode registers a new node in the cluster
func (c *Coordinator) RegisterNode(node Node) error {
	c.nodesMu.Lock()
	defer c.nodesMu.Unlock()
	
	// Check if node already exists
	for i, existing := range c.nodes {
		if existing.ID == node.ID {
			c.nodes[i] = node
			log.Info().Str("node_id", node.ID).Msg("Updated existing node")
			return nil
		}
	}
	
	// Add new node
	node.Status = NodeStatusHealthy
	node.LastHealthCheck = time.Now()
	c.nodes = append(c.nodes, node)
	
	// Rebalance shards
	c.rebalanceShards()
	
	log.Info().Str("node_id", node.ID).Msg("Registered new node")
	return nil
}

// RemoveNode removes a node from the cluster
func (c *Coordinator) RemoveNode(nodeID string) error {
	c.nodesMu.Lock()
	defer c.nodesMu.Unlock()
	
	for i, node := range c.nodes {
		if node.ID == nodeID {
			// Remove node
			c.nodes = append(c.nodes[:i], c.nodes[i+1:]...)
			
			// Rebalance shards
			c.rebalanceShards()
			
			log.Info().Str("node_id", nodeID).Msg("Removed node from cluster")
			return nil
		}
	}
	
	return fmt.Errorf("node not found: %s", nodeID)
}

// GetNode returns a node for the given key
func (c *Coordinator) GetNode(key string) (*Node, error) {
	c.nodesMu.RLock()
	defer c.nodesMu.RUnlock()
	
	healthyNodes := c.getHealthyNodes()
	if len(healthyNodes) == 0 {
		return nil, fmt.Errorf("no healthy nodes available")
	}
	
	return c.loadBalancer.SelectNode(healthyNodes, key)
}

// GetNodesForShard returns nodes responsible for a shard
func (c *Coordinator) GetNodesForShard(key string) ([]Node, error) {
	c.nodesMu.RLock()
	defer c.nodesMu.RUnlock()
	
	shard := c.shardingStrategy.GetShard(key, c.config.ShardCount)
	nodes := c.shardingStrategy.GetNodesForShard(shard, c.nodes)
	
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes available for shard %d", shard)
	}
	
	return nodes, nil
}

// getHealthyNodes returns only healthy nodes
func (c *Coordinator) getHealthyNodes() []Node {
	healthy := []Node{}
	for _, node := range c.nodes {
		if node.Status == NodeStatusHealthy {
			healthy = append(healthy, node)
		}
	}
	return healthy
}

// rebalanceShards redistributes shards among nodes
func (c *Coordinator) rebalanceShards() {
	if len(c.nodes) == 0 {
		return
	}
	
	shardsPerNode := c.config.ShardCount / len(c.nodes)
	extraShards := c.config.ShardCount % len(c.nodes)
	
	shard := 0
	for i := range c.nodes {
		c.nodes[i].Shards = []int{}
		
		// Assign base shards
		for j := 0; j < shardsPerNode; j++ {
			c.nodes[i].Shards = append(c.nodes[i].Shards, shard)
			shard++
		}
		
		// Distribute extra shards
		if i < extraShards {
			c.nodes[i].Shards = append(c.nodes[i].Shards, shard)
			shard++
		}
	}
}

// StartHealthChecking starts periodic health checking
func (c *Coordinator) StartHealthChecking(ctx context.Context) {
	go c.healthChecker.Start(ctx, c)
}

// UpdateNodeHealth updates node health status
func (c *Coordinator) UpdateNodeHealth(nodeID string, status NodeStatus) {
	c.nodesMu.Lock()
	defer c.nodesMu.Unlock()
	
	for i, node := range c.nodes {
		if node.ID == nodeID {
			c.nodes[i].Status = status
			c.nodes[i].LastHealthCheck = time.Now()
			break
		}
	}
}

// RoundRobinBalancer implements round-robin load balancing
type RoundRobinBalancer struct {
	current uint64
}

// NewRoundRobinBalancer creates a round-robin balancer
func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{}
}

// SelectNode selects next node in round-robin fashion
func (rb *RoundRobinBalancer) SelectNode(nodes []Node, key string) (*Node, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}
	
	index := atomic.AddUint64(&rb.current, 1) % uint64(len(nodes))
	return &nodes[index], nil
}

// UpdateLoad updates node load (not used in round-robin)
func (rb *RoundRobinBalancer) UpdateLoad(nodeID string, load float64) {}

// LeastLoadedBalancer selects least loaded node
type LeastLoadedBalancer struct {
	loads map[string]float64
	mu    sync.RWMutex
}

// NewLeastLoadedBalancer creates a least-loaded balancer
func NewLeastLoadedBalancer() *LeastLoadedBalancer {
	return &LeastLoadedBalancer{
		loads: make(map[string]float64),
	}
}

// SelectNode selects the least loaded node
func (lb *LeastLoadedBalancer) SelectNode(nodes []Node, key string) (*Node, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}
	
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	
	var selected *Node
	minLoad := float64(1.0)
	
	for i := range nodes {
		load, exists := lb.loads[nodes[i].ID]
		if !exists {
			load = 0.0
		}
		
		if load < minLoad {
			minLoad = load
			selected = &nodes[i]
		}
	}
	
	if selected == nil {
		selected = &nodes[0]
	}
	
	return selected, nil
}

// UpdateLoad updates node load
func (lb *LeastLoadedBalancer) UpdateLoad(nodeID string, load float64) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.loads[nodeID] = load
}

// ConsistentHashBalancer implements consistent hashing
type ConsistentHashBalancer struct {
	hashRing *ConsistentHashRing
}

// NewConsistentHashBalancer creates a consistent hash balancer
func NewConsistentHashBalancer() *ConsistentHashBalancer {
	return &ConsistentHashBalancer{
		hashRing: NewConsistentHashRing(150), // 150 virtual nodes per physical node
	}
}

// SelectNode selects node using consistent hashing
func (ch *ConsistentHashBalancer) SelectNode(nodes []Node, key string) (*Node, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}
	
	// Update hash ring if needed
	ch.hashRing.Update(nodes)
	
	// Get node for key
	nodeID := ch.hashRing.GetNode(key)
	
	// Find node by ID
	for i := range nodes {
		if nodes[i].ID == nodeID {
			return &nodes[i], nil
		}
	}
	
	return &nodes[0], nil
}

// UpdateLoad updates node load (not used in consistent hashing)
func (ch *ConsistentHashBalancer) UpdateLoad(nodeID string, load float64) {}

// ConsistentHashRing implements consistent hashing ring
type ConsistentHashRing struct {
	virtualNodes int
	ring         map[uint32]string
	sortedKeys   []uint32
	mu           sync.RWMutex
}

// NewConsistentHashRing creates a new consistent hash ring
func NewConsistentHashRing(virtualNodes int) *ConsistentHashRing {
	return &ConsistentHashRing{
		virtualNodes: virtualNodes,
		ring:         make(map[uint32]string),
		sortedKeys:   []uint32{},
	}
}

// Update updates the hash ring with current nodes
func (chr *ConsistentHashRing) Update(nodes []Node) {
	chr.mu.Lock()
	defer chr.mu.Unlock()
	
	// Clear existing ring
	chr.ring = make(map[uint32]string)
	chr.sortedKeys = []uint32{}
	
	// Add nodes to ring
	for _, node := range nodes {
		for i := 0; i < chr.virtualNodes; i++ {
			virtualKey := fmt.Sprintf("%s-%d", node.ID, i)
			hash := chr.hash(virtualKey)
			chr.ring[hash] = node.ID
			chr.sortedKeys = append(chr.sortedKeys, hash)
		}
	}
	
	// Sort keys
	chr.sortKeys()
}

// GetNode returns node for given key
func (chr *ConsistentHashRing) GetNode(key string) string {
	chr.mu.RLock()
	defer chr.mu.RUnlock()
	
	if len(chr.sortedKeys) == 0 {
		return ""
	}
	
	hash := chr.hash(key)
	
	// Binary search for first key >= hash
	idx := 0
	for i := 0; i < len(chr.sortedKeys); i++ {
		if chr.sortedKeys[i] >= hash {
			idx = i
			break
		}
	}
	
	// Wrap around if necessary
	if idx == len(chr.sortedKeys) {
		idx = 0
	}
	
	return chr.ring[chr.sortedKeys[idx]]
}

// hash generates hash for key
func (chr *ConsistentHashRing) hash(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// sortKeys sorts the hash ring keys
func (chr *ConsistentHashRing) sortKeys() {
	// Simple bubble sort for now
	for i := 0; i < len(chr.sortedKeys); i++ {
		for j := i + 1; j < len(chr.sortedKeys); j++ {
			if chr.sortedKeys[i] > chr.sortedKeys[j] {
				chr.sortedKeys[i], chr.sortedKeys[j] = chr.sortedKeys[j], chr.sortedKeys[i]
			}
		}
	}
}

// HashSharding implements hash-based sharding
type HashSharding struct{}

// NewHashSharding creates hash-based sharding
func NewHashSharding() *HashSharding {
	return &HashSharding{}
}

// GetShard returns shard for given key
func (hs *HashSharding) GetShard(key string, shardCount int) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() % uint32(shardCount))
}

// GetNodesForShard returns nodes responsible for shard
func (hs *HashSharding) GetNodesForShard(shard int, nodes []Node) []Node {
	responsible := []Node{}
	
	for _, node := range nodes {
		for _, s := range node.Shards {
			if s == shard {
				responsible = append(responsible, node)
				break
			}
		}
	}
	
	return responsible
}

// HealthChecker performs periodic health checks
type HealthChecker struct {
	interval time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(interval time.Duration) *HealthChecker {
	return &HealthChecker{
		interval: interval,
	}
}

// Start starts health checking routine
func (hc *HealthChecker) Start(ctx context.Context, coordinator *Coordinator) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			hc.checkNodes(coordinator)
		case <-ctx.Done():
			return
		}
	}
}

// checkNodes checks health of all nodes
func (hc *HealthChecker) checkNodes(coordinator *Coordinator) {
	coordinator.nodesMu.RLock()
	nodes := make([]Node, len(coordinator.nodes))
	copy(nodes, coordinator.nodes)
	coordinator.nodesMu.RUnlock()
	
	for _, node := range nodes {
		status := hc.checkNodeHealth(node)
		coordinator.UpdateNodeHealth(node.ID, status)
	}
}

// checkNodeHealth checks individual node health
func (hc *HealthChecker) checkNodeHealth(node Node) NodeStatus {
	// In real implementation, this would make health check request
	// For now, return healthy
	return NodeStatusHealthy
}