// examples/main.go
package main

import (
	"fmt"
	"log"

	"github.com/mohdrashid9678/dcore/chash"
)

func main() {
	fmt.Println("Consistent Hashing Demo")
	fmt.Println("==========================")

	// Demo 1: Basic usage
	basicDemo()

	fmt.Println()

	// Demo 2: Load distribution
	loadDistributionDemo()

	fmt.Println()

	// Demo 3: Node addition/removal impact
	nodeChangesDemo()

	fmt.Println()

	// Demo 4: Replication scenario
	replicationDemo()
}

func basicDemo() {
	fmt.Println("Basic Usage Demo")
	fmt.Println("-------------------")

	// Create a consistent hash ring
	ring := chash.New(chash.Config{
		Replicas: 150, // 150 virtual nodes per physical node
	})

	// Add some servers
	servers := []string{
		"web-server-1:8080",
		"web-server-2:8080",
		"web-server-3:8080",
		"web-server-4:8080",
	}

	for _, server := range servers {
		if err := ring.AddNode(server); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Added server: %s\n", server)
	}

	// Test key routing
	testKeys := []string{
		"user:alice",
		"user:bob",
		"user:charlie",
		"session:abc123",
		"cache:popular-posts",
	}

	fmt.Println("\n Key Routing:")
	for _, key := range testKeys {
		server, err := ring.GetNode(key)
		if err != nil {
			log.Printf("Error getting node for %s: %v", key, err)
			continue
		}
		fmt.Printf("  %s → %s\n", key, server)
	}

	// Show statistics
	stats := ring.GetStats()
	fmt.Printf("\nRing Statistics:\n")
	fmt.Printf("  Physical nodes: %d\n", stats.PhysicalNodes)
	fmt.Printf("  Virtual nodes: %d\n", stats.VirtualNodes)
	fmt.Printf("  Replicas per node: %d\n", stats.Replicas)
}

func loadDistributionDemo() {
	fmt.Println(" Load Distribution Demo")
	fmt.Println("-------------------------")

	ring := chash.New(chash.Config{Replicas: 150})

	// Add servers
	numServers := 5
	for i := 0; i < numServers; i++ {
		server := fmt.Sprintf("server-%d:8080", i+1)
		ring.AddNode(server)
	}

	// Generate many keys and see distribution
	numKeys := 10000
	distribution := make(map[string]int)

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		server, err := ring.GetNode(key)
		if err != nil {
			continue
		}
		distribution[server]++
	}

	fmt.Printf("Distribution of %d keys across %d servers:\n", numKeys, numServers)
	expectedPerServer := numKeys / numServers

	for server, count := range distribution {
		percentage := float64(count) / float64(numKeys) * 100
		deviation := float64(count-expectedPerServer) / float64(expectedPerServer) * 100
		fmt.Printf("  %s: %d keys (%.1f%%, %+.1f%% from expected)\n",
			server, count, percentage, deviation)
	}
}

func nodeChangesDemo() {
	fmt.Println("Node Addition/Removal Impact Demo")
	fmt.Println("------------------------------------")

	ring := chash.New(chash.Config{Replicas: 150})

	// Add initial servers
	initialServers := []string{"server-1", "server-2", "server-3", "server-4"}
	for _, server := range initialServers {
		ring.AddNode(server)
	}

	// Generate test keys and record initial assignments
	numKeys := 1000
	initialAssignments := make(map[string]string)

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		server, _ := ring.GetNode(key)
		initialAssignments[key] = server
	}

	fmt.Printf("Initial state: %d keys distributed across %d servers\n",
		numKeys, len(initialServers))

	// Add a new server
	newServer := "server-5"
	ring.AddNode(newServer)
	fmt.Printf("\n Added new server: %s\n", newServer)

	// Check how many keys moved
	movedAfterAdd := 0
	movedToNewServer := 0

	for key, originalServer := range initialAssignments {
		currentServer, _ := ring.GetNode(key)
		if originalServer != currentServer {
			movedAfterAdd++
			if currentServer == newServer {
				movedToNewServer++
			}
		}
	}

	fmt.Printf("Impact of adding server:\n")
	fmt.Printf("  Keys moved: %d (%.1f%%)\n", movedAfterAdd,
		float64(movedAfterAdd)/float64(numKeys)*100)
	fmt.Printf("  Keys moved to new server: %d\n", movedToNewServer)

	// Remove a server
	removedServer := "server-2"
	ring.RemoveNode(removedServer)
	fmt.Printf("\n➖ Removed server: %s\n", removedServer)

	// Check impact of removal
	movedAfterRemove := 0
	for key, serverAfterAdd := range getAssignments(ring, numKeys) {
		if initialAssignments[key] == removedServer {
			// This key was on the removed server, so it should move
			continue
		}
		if serverAfterAdd != initialAssignments[key] {
			movedAfterRemove++
		}
	}

	keysOnRemovedServer := countKeysOnServer(initialAssignments, removedServer)
	fmt.Printf("Impact of removing server:\n")
	fmt.Printf("  Keys that were on removed server: %d\n", keysOnRemovedServer)
	fmt.Printf("  Additional keys moved from other servers: %d (%.1f%%)\n",
		movedAfterRemove, float64(movedAfterRemove)/float64(numKeys)*100)
}

func replicationDemo() {
	fmt.Println("Replication Demo")
	fmt.Println("------------------")

	ring := chash.New(chash.Config{Replicas: 150})

	// Add servers
	servers := []string{"cache-1", "cache-2", "cache-3", "cache-4", "cache-5"}
	for _, server := range servers {
		ring.AddNode(server)
	}

	// Simulate storing data with replication
	testKeys := []string{
		"user-profile:12345",
		"shopping-cart:67890",
		"session:abcdef",
		"popular-posts",
		"trending-topics",
	}

	replicationFactor := 3

	fmt.Printf("Storing data with replication factor of %d:\n\n", replicationFactor)

	for _, key := range testKeys {
		replicas, err := ring.GetNodes(key, replicationFactor)
		if err != nil {
			log.Printf("Error getting replicas for %s: %v", key, err)
			continue
		}

		fmt.Printf("Key: %s\n", key)
		fmt.Printf("  Primary: %s\n", replicas[0])
		fmt.Printf("  Replicas: %v\n", replicas[1:])
		fmt.Println()
	}

	// Simulate server failure
	failedServer := "cache-3"
	fmt.Printf("Simulating failure of server: %s\n", failedServer)
	ring.RemoveNode(failedServer)

	fmt.Println("\nData availability after server failure:")
	for _, key := range testKeys {
		replicas, err := ring.GetNodes(key, replicationFactor)
		if err != nil {
			continue
		}

		fmt.Printf("Key %s: still available on %d servers: %v\n",
			key, len(replicas), replicas)
	}
}

// Helper functions
func getAssignments(ring *chash.Ring, numKeys int) map[string]string {
	assignments := make(map[string]string)
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		server, _ := ring.GetNode(key)
		assignments[key] = server
	}
	return assignments
}

func countKeysOnServer(assignments map[string]string, server string) int {
	count := 0
	for _, s := range assignments {
		if s == server {
			count++
		}
	}
	return count
}
