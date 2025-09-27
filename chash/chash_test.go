package chash

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected int
	}{
		{
			name:     "default config",
			config:   Config{},
			expected: 150,
		},
		{
			name:     "custom replicas",
			config:   Config{Replicas: 100},
			expected: 100,
		},
		{
			name:     "zero replicas defaults to 150",
			config:   Config{Replicas: 0},
			expected: 150,
		},
		{
			name:     "negative replicas defaults to 150",
			config:   Config{Replicas: -5},
			expected: 150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ring := New(tt.config)
			if ring.replicas != tt.expected {
				t.Errorf("expected replicas %d, got %d", tt.expected, ring.replicas)
			}
			if ring.hashFunc == nil {
				t.Error("hash function should not be nil")
			}
		})
	}
}

func TestAddNode(t *testing.T) {
	ring := New(Config{Replicas: 3})

	// Test adding valid node
	err := ring.AddNode("server1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if ring.NodeCount() != 1 {
		t.Errorf("expected 1 node, got %d", ring.NodeCount())
	}

	if ring.VirtualNodeCount() != 3 {
		t.Errorf("expected 3 virtual nodes, got %d", ring.VirtualNodeCount())
	}

	// Test adding duplicate node
	err = ring.AddNode("server1")
	if err == nil {
		t.Error("expected error when adding duplicate node")
	}

	// Test adding empty node
	err = ring.AddNode("")
	if err != ErrEmptyKey {
		t.Errorf("expected ErrEmptyKey, got %v", err)
	}
}

func TestRemoveNode(t *testing.T) {
	ring := New(Config{Replicas: 3})
	ring.AddNode("server1")
	ring.AddNode("server2")

	// Test removing existing node
	err := ring.RemoveNode("server1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if ring.NodeCount() != 1 {
		t.Errorf("expected 1 node after removal, got %d", ring.NodeCount())
	}

	// Test removing non-existent node
	err = ring.RemoveNode("server3")
	if err != ErrNodeNotFound {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}

	// Test removing empty node
	err = ring.RemoveNode("")
	if err != ErrEmptyKey {
		t.Errorf("expected ErrEmptyKey, got %v", err)
	}
}

func TestGetNode(t *testing.T) {
	ring := New(Config{Replicas: 3})

	// Test with empty ring
	_, err := ring.GetNode("key1")
	if err != ErrNoNodes {
		t.Errorf("expected ErrNoNodes, got %v", err)
	}

	// Add nodes
	nodes := []string{"server1", "server2", "server3"}
	for _, node := range nodes {
		ring.AddNode(node)
	}

	// Test with empty key
	_, err = ring.GetNode("")
	if err != ErrEmptyKey {
		t.Errorf("expected ErrEmptyKey, got %v", err)
	}

	// Test key assignment
	node, err := ring.GetNode("user123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if node == "" {
		t.Error("expected non-empty node")
	}

	// Test consistency - same key should always return same node
	for i := 0; i < 10; i++ {
		n, err := ring.GetNode("user123")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if n != node {
			t.Errorf("expected consistent node assignment, got %s initially, %s later", node, n)
		}
	}
}

func TestGetNodes(t *testing.T) {
	ring := New(Config{Replicas: 3})

	// Test with empty ring
	_, err := ring.GetNodes("key1", 2)
	if err != ErrNoNodes {
		t.Errorf("expected ErrNoNodes, got %v", err)
	}

	// Add nodes
	nodes := []string{"server1", "server2", "server3", "server4"}
	for _, node := range nodes {
		ring.AddNode(node)
	}

	// Test with empty key
	_, err = ring.GetNodes("", 2)
	if err != ErrEmptyKey {
		t.Errorf("expected ErrEmptyKey, got %v", err)
	}

	// Test with invalid count
	_, err = ring.GetNodes("key1", 0)
	if err == nil {
		t.Error("expected error for zero count")
	}

	// Test normal case
	result, err := ring.GetNodes("user123", 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(result))
	}

	// Test requesting more nodes than available
	result, err = ring.GetNodes("user123", 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result) != 4 {
		t.Errorf("expected 4 nodes (all available), got %d", len(result))
	}

	// Ensure no duplicates
	seen := make(map[string]bool)
	for _, node := range result {
		if seen[node] {
			t.Errorf("duplicate node found: %s", node)
		}
		seen[node] = true
	}
}

func TestConsistentDistribution(t *testing.T) {
	ring := New(Config{Replicas: 150})

	// Add servers
	for i := 0; i < 5; i++ {
		ring.AddNode(fmt.Sprintf("server%d", i))
	}

	// Test key distribution
	distribution := make(map[string]int)
	numKeys := 10000

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key%d", i)
		node, err := ring.GetNode(key)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		distribution[node]++
	}

	// Check that distribution is reasonably uniform
	expectedPerNode := numKeys / 5
	tolerance := int(float64(expectedPerNode) * 0.3) // 30% tolerance

	for node, count := range distribution {
		if count < expectedPerNode-tolerance || count > expectedPerNode+tolerance {
			t.Errorf("Node %s has %d keys, expected around %d (±%d)", node, count, expectedPerNode, tolerance)
		}
	}
}

func TestNodeRemovalConsistency(t *testing.T) {
	ring := New(Config{Replicas: 150})

	// Add servers
	servers := []string{"server1", "server2", "server3", "server4", "server5"}
	for _, server := range servers {
		ring.AddNode(server)
	}

	// Record initial assignments
	initialAssignments := make(map[string]string)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key%d", i)
		node, _ := ring.GetNode(key)
		initialAssignments[key] = node
	}

	// Remove one server
	ring.RemoveNode("server3")

	// Check how many keys were reassigned
	reassigned := 0
	for key, originalNode := range initialAssignments {
		newNode, _ := ring.GetNode(key)
		if originalNode == "server3" {
			// Keys from removed server should be reassigned
			if newNode == "server3" {
				t.Error("key still assigned to removed server")
			}
		} else if originalNode != newNode {
			// Keys from other servers shouldn't move much
			reassigned++
		}
	}

	// Only keys from the removed server plus minimal disruption should be reassigned
	maxExpectedReassigned := int(float64(len(initialAssignments)) * 0.3) // Max 30% disruption
	if reassigned > maxExpectedReassigned {
		t.Errorf("too many keys reassigned: %d (expected < %d)", reassigned, maxExpectedReassigned)
	}
}

func TestConcurrentAccess(t *testing.T) {
	ring := New(Config{Replicas: 10})

	// Add initial nodes
	for i := 0; i < 5; i++ {
		ring.AddNode(fmt.Sprintf("server%d", i))
	}

	var wg sync.WaitGroup
	errors := make(chan error, 1000)

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := fmt.Sprintf("key%d_%d", id, j)
				_, err := ring.GetNode(key)
				if err != nil && err != ErrNoNodes {
					errors <- err
					return
				}
			}
		}(i)
	}

	// Concurrent writes (add/remove nodes)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nodeName := fmt.Sprintf("temp_server%d", id)
			ring.AddNode(nodeName)
			ring.RemoveNode(nodeName)
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("concurrent access error: %v", err)
	}
}

func TestNewWithNodes(t *testing.T) {
	nodes := []string{"server1", "server2", "server3"}
	ring := NewWithNodes(Config{Replicas: 3}, nodes)

	if ring.NodeCount() != 3 {
		t.Errorf("expected 3 nodes, got %d", ring.NodeCount())
	}

	if ring.VirtualNodeCount() != 9 {
		t.Errorf("expected 9 virtual nodes, got %d", ring.VirtualNodeCount())
	}

	// Verify all nodes are present
	ringNodes := ring.Nodes()
	if len(ringNodes) != 3 {
		t.Errorf("expected 3 nodes in ring, got %d", len(ringNodes))
	}

	for _, expectedNode := range nodes {
		found := false
		for _, ringNode := range ringNodes {
			if ringNode == expectedNode {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("node %s not found in ring", expectedNode)
		}
	}
}

func TestIsEmpty(t *testing.T) {
	ring := New(Config{})

	if !ring.IsEmpty() {
		t.Error("new ring should be empty")
	}

	ring.AddNode("server1")

	if ring.IsEmpty() {
		t.Error("ring with nodes should not be empty")
	}

	ring.RemoveNode("server1")

	if !ring.IsEmpty() {
		t.Error("ring should be empty after removing all nodes")
	}
}

func TestGetStats(t *testing.T) {
	ring := New(Config{Replicas: 5})
	ring.AddNode("server1")
	ring.AddNode("server2")

	stats := ring.GetStats()

	if stats.PhysicalNodes != 2 {
		t.Errorf("expected 2 physical nodes, got %d", stats.PhysicalNodes)
	}

	if stats.VirtualNodes != 10 {
		t.Errorf("expected 10 virtual nodes, got %d", stats.VirtualNodes)
	}

	if stats.Replicas != 5 {
		t.Errorf("expected 5 replicas, got %d", stats.Replicas)
	}
}

func TestCustomHashFunction(t *testing.T) {
	// Create a simple hash function for testing
	simpleHash := func(key string) uint64 {
		var hash uint64
		for _, c := range key {
			hash = hash*31 + uint64(c)
		}
		return hash
	}

	ring := New(Config{
		Replicas: 3,
		HashFunc: simpleHash,
	})

	ring.AddNode("server1")

	// Test that our custom hash function is being used
	node, err := ring.GetNode("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if node != "server1" {
		t.Errorf("expected server1, got %s", node)
	}
}

// Benchmarks
func BenchmarkAddNode(b *testing.B) {
	ring := New(Config{Replicas: 150})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ring.AddNode("server" + strconv.Itoa(i))
	}
}

func BenchmarkGetNode(b *testing.B) {
	ring := New(Config{Replicas: 150})

	// Setup
	for i := 0; i < 100; i++ {
		ring.AddNode("server" + strconv.Itoa(i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ring.GetNode("key" + strconv.Itoa(i))
	}
}

func BenchmarkGetNodeConcurrent(b *testing.B) {
	ring := New(Config{Replicas: 150})

	// Setup
	for i := 0; i < 100; i++ {
		ring.AddNode("server" + strconv.Itoa(i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ring.GetNode("key" + strconv.Itoa(i))
			i++
		}
	})
}

func BenchmarkRemoveNode(b *testing.B) {
	// Setup - create nodes first
	nodes := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		nodes[i] = "server" + strconv.Itoa(i)
	}

	ring := NewWithNodes(Config{Replicas: 150}, nodes)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ring.RemoveNode(nodes[i])
	}
}
