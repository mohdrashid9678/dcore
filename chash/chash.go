// Package chash provides a thread-safe implementation of consistent hashing
// with virtual nodes for distributed systems.

package chash

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
)

var (
	// ErrNoNodes is returned when no nodes are available in the ring
	ErrNoNodes = errors.New("no nodes available in the hash ring")

	// ErrNodeNotFound is returned when attempting to remove a non-existent node
	ErrNodeNotFound = errors.New("node not found in the hash ring")

	// ErrEmptyKey is returned when an empty key is provided
	ErrEmptyKey = errors.New("key cannot be empty")
)

// HashFunc represents a hash function that takes a string and returns a uint64 hash
type HashFunc func(string) uint64

// DefaultHashFunc provides a default SHA-256 based hash function
func DefaultHashFunc(key string) uint64 {
	h := sha256.Sum256([]byte(key))
	return binary.BigEndian.Uint64(h[:8])
}

// Ring represents a consistent hash ring with virtual nodes
type Ring struct {
	// mu protects all fields below for concurrent access
	mu sync.RWMutex

	// hashFunc is the hash function used for generating hashes
	hashFunc HashFunc

	// replicas is the number of virtual nodes per physical node
	replicas int

	// ring stores the hash ring as sorted slice of hash values
	ring []uint64

	// nodes maps hash values to node names
	nodes map[uint64]string

	// nodeSet keeps track of all physical nodes for O(1) existence checks
	nodeSet map[string]struct{}
}

// Config holds configuration options for creating a new Ring
type Config struct {
	// Replicas specifies the number of virtual nodes per physical node
	// Higher values provide better distribution but use more memory
	// Default: 150
	Replicas int

	// HashFunc specifies the hash function to use
	// Default: DefaultHashFunc (SHA-256 based)
	HashFunc HashFunc
}

// New creates a new consistent hash ring with the given configuration
func New(config Config) *Ring {
	if config.Replicas <= 0 {
		config.Replicas = 150 // Default number of replicas
	}

	if config.HashFunc == nil {
		config.HashFunc = DefaultHashFunc
	}

	return &Ring{
		hashFunc: config.HashFunc,
		replicas: config.Replicas,
		nodes:    make(map[uint64]string),
		nodeSet:  make(map[string]struct{}),
	}
}

// NewWithNodes creates a new consistent hash ring with initial nodes
func NewWithNodes(config Config, nodes []string) *Ring {
	ring := New(config)
	for _, node := range nodes {
		ring.AddNode(node)
	}
	return ring
}

// AddNode adds a physical node to the hash ring with virtual nodes
// Returns an error if the node already exists
func (r *Ring) AddNode(node string) error {
	if node == "" {
		return ErrEmptyKey
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if node already exists
	if _, exists := r.nodeSet[node]; exists {
		return fmt.Errorf("node %s already exists", node)
	}

	// Add virtual nodes
	for i := 0; i < r.replicas; i++ {
		virtualNode := node + "#" + strconv.Itoa(i)
		hash := r.hashFunc(virtualNode)

		r.nodes[hash] = node
		r.ring = append(r.ring, hash)
	}

	// Sort ring to maintain order
	sort.Slice(r.ring, func(i, j int) bool {
		return r.ring[i] < r.ring[j]
	})

	r.nodeSet[node] = struct{}{}

	return nil
}

// RemoveNode removes a physical node and all its virtual nodes from the ring
// Returns an error if the node doesn't exist
func (r *Ring) RemoveNode(node string) error {
	if node == "" {
		return ErrEmptyKey
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if node exists
	if _, exists := r.nodeSet[node]; !exists {
		return ErrNodeNotFound
	}

	// Remove virtual nodes
	newRing := make([]uint64, 0, len(r.ring)-r.replicas)
	for _, hash := range r.ring {
		if r.nodes[hash] != node {
			newRing = append(newRing, hash)
		} else {
			delete(r.nodes, hash)
		}
	}

	r.ring = newRing
	delete(r.nodeSet, node)

	return nil
}

// GetNode returns the node responsible for the given key
// Uses clockwise traversal to find the closest node
func (r *Ring) GetNode(key string) (string, error) {
	if key == "" {
		return "", ErrEmptyKey
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.ring) == 0 {
		return "", ErrNoNodes
	}

	hash := r.hashFunc(key)

	// Binary search for the first node with hash >= key hash
	idx := sort.Search(len(r.ring), func(i int) bool {
		return r.ring[i] >= hash
	})

	// If no node found, wrap around to the first node
	if idx == len(r.ring) {
		idx = 0
	}

	return r.nodes[r.ring[idx]], nil
}

// GetNodes returns the top N nodes responsible for the given key
// Useful for replication scenarios where data should be stored on multiple nodes
func (r *Ring) GetNodes(key string, count int) ([]string, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}

	if count <= 0 {
		return nil, errors.New("count must be positive")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.ring) == 0 {
		return nil, ErrNoNodes
	}

	if count > len(r.nodeSet) {
		count = len(r.nodeSet)
	}

	hash := r.hashFunc(key)

	// Binary search for the first node with hash >= key hash
	idx := sort.Search(len(r.ring), func(i int) bool {
		return r.ring[i] >= hash
	})

	// If no node found, wrap around to the first node
	if idx == len(r.ring) {
		idx = 0
	}

	result := make([]string, 0, count)
	seen := make(map[string]struct{})

	// Traverse the ring clockwise until we have enough unique nodes
	for i := 0; i < len(r.ring) && len(result) < count; i++ {
		currentIdx := (idx + i) % len(r.ring)
		node := r.nodes[r.ring[currentIdx]]

		if _, exists := seen[node]; !exists {
			result = append(result, node)
			seen[node] = struct{}{}
		}
	}

	return result, nil
}

// Nodes returns a list of all physical nodes in the ring
func (r *Ring) Nodes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make([]string, 0, len(r.nodeSet))
	for node := range r.nodeSet {
		nodes = append(nodes, node)
	}

	// Sort for consistent ordering
	sort.Strings(nodes)
	return nodes
}

// IsEmpty returns true if the ring has no nodes
func (r *Ring) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.nodeSet) == 0
}

// NodeCount returns the number of physical nodes in the ring
func (r *Ring) NodeCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.nodeSet)
}

// VirtualNodeCount returns the total number of virtual nodes in the ring
func (r *Ring) VirtualNodeCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.ring)
}

// Stats returns statistical information about the ring
type Stats struct {
	PhysicalNodes int
	VirtualNodes  int
	Replicas      int
	LoadFactor    float64 // Average number of virtual nodes per physical node
}

// GetStats returns statistical information about the hash ring
func (r *Ring) GetStats() Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return Stats{
		PhysicalNodes: len(r.nodeSet),
		VirtualNodes:  len(r.ring),
		Replicas:      r.replicas,
		LoadFactor:    0,
	}
}
