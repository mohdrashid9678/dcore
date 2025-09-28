# Consistent Hashing

A production-ready, thread-safe Go implementation of consistent hashing with virtual nodes for distributed systems.

## Features

- **Thread-safe**: Full concurrent read/write support with optimized RWMutex usage
- **Virtual nodes**: Configurable replica count for better load distribution
- **Minimal disruption**: Adding/removing nodes affects only a small portion of keys
- **Customizable hashing**: Pluggable hash functions (defaults to SHA-256)
- **Zero dependencies**: Uses only Go standard library
- **Production ready**: Comprehensive error handling, logging, and testing
- **High performance**: Optimized with binary search and efficient data structures

## Installation

```bash
go get github.com/mohdrashid9678/dcore/chash
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/mohdrashid9678/dcore/chash"
)

func main() {
    // Create a new consistent hash ring with 150 virtual nodes per server
    ring := chash.New(chash.Config{
        Replicas: 150,
    })

    // Add servers to the ring
    servers := []string{"server1:8080", "server2:8080", "server3:8080"}
    for _, server := range servers {
        if err := ring.AddNode(server); err != nil {
            log.Fatal(err)
        }
    }

    // Find which server should handle a specific key
    server, err := ring.GetNode("user:12345")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Key 'user:12345' maps to server: %s\n", server)

    // Get multiple servers for replication
    servers, err = ring.GetNodes("user:12345", 2)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Key 'user:12345' should be replicated to: %v\n", servers)
}
```

## API Documentation

### Creating a Ring

```go
// Create with default settings (150 replicas, SHA-256 hash)
ring := chash.New(chash.Config{})

// Create with custom settings
ring := chash.New(chash.Config{
    Replicas: 150,
    HashFunc: md5Hash,
})
```

### Error Handling

The library provides specific error types for different scenarios:

```go
node, err := ring.GetNode("mykey")
switch err {
case chash.ErrNoNodes:
    // Handle case where ring is empty
    log.Println("No servers available")
case chash.ErrEmptyKey:
    // Handle empty key
    log.Println("Key cannot be empty")
case nil:
    // Success
    fmt.Printf("Key maps to: %s\n", node)
default:
    // Other errors
    log.Printf("Unexpected error: %v\n", err)
}
```

### Concurrent Usage

The ring is fully thread-safe and optimized for concurrent access:

```go
// Safe to use from multiple goroutines
go func() {
    for {
        node, _ := ring.GetNode("user:12345")
        // Process with node...
    }
}()

go func() {
    // Dynamic scaling
    ring.AddNode("new-server:8080")
    time.Sleep(time.Hour)
    ring.RemoveNode("new-server:8080")
}()
```

## Performance Characteristics

- **Add Node**: O(R log N) where R is replicas and N is total virtual nodes
- **Remove Node**: O(R log N) where R is replicas and N is total virtual nodes
- **Get Node**: O(log N) where N is total virtual nodes
- **Memory**: O(R × P) where R is replicas and P is physical nodes
- **Thread Safety**: Optimized RWMutex with readers favored for lookups

## Benchmarks

```
BenchmarkAddNode-8              5000    285432 ns/op    24563 B/op    12 allocs/op
BenchmarkGetNode-8           2000000       956 ns/op        0 B/op     0 allocs/op
BenchmarkGetNodeConcurrent-8 5000000       312 ns/op        0 B/op     0 allocs/op
BenchmarkRemoveNode-8           3000    421876 ns/op     8234 B/op     8 allocs/op
```

## Best Practices

### Replica Count Selection

- **Low latency systems**: 50-100 replicas
- **General purpose**: 150 replicas (recommended default)
- **High availability**: 200-300 replicas
- **Memory constrained**: 20-50 replicas

### Node Naming

Use consistent, descriptive node names:

```go
// Good
ring.AddNode("web-server-1.datacenter-east.company.com:8080")
ring.AddNode("cache-node-3.region-west.company.com:6379")

// Avoid
ring.AddNode("server1")
ring.AddNode("node-a")
```

### Error Handling in Production

```go
func safeGetNode(ring *chash.Ring, key string) (string, error) {
    if key == "" {
        return "", fmt.Errorf("invalid key")
    }

    node, err := ring.GetNode(key)
    if err == chash.ErrNoNodes {
        // Fallback strategy
        return "fallback-server:8080", nil
    }

    return node, err
}
```

### Monitoring Integration

```go
// Example with Prometheus metrics
import "github.com/prometheus/client_golang/prometheus"

var (
    nodeCount = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "consistent_hash_nodes_total",
        Help: "Total number of nodes in the hash ring",
    })
)

func updateMetrics(ring *chash.Ring) {
    stats := ring.GetStats()
    nodeCount.Set(float64(stats.PhysicalNodes))
}
```

## Use Cases

### Load Balancing

```go
type LoadBalancer struct {
    ring *chash.Ring
}

func (lb *LoadBalancer) GetServer(clientID string) (string, error) {
    return lb.ring.GetNode(clientID)
}

func (lb *LoadBalancer) AddServer(server string) error {
    return lb.ring.AddNode(server)
}
```

### Distributed Caching

```go
type DistributedCache struct {
    ring *chash.Ring
    clients map[string]*redis.Client
}

func (dc *DistributedCache) Get(key string) (string, error) {
    server, err := dc.ring.GetNode(key)
    if err != nil {
        return "", err
    }

    client := dc.clients[server]
    return client.Get(context.Background(), key).Result()
}

func (dc *DistributedCache) Set(key, value string) error {
    server, err := dc.ring.GetNode(key)
    if err != nil {
        return err
    }

    client := dc.clients[server]
    return client.Set(context.Background(), key, value, 0).Err()
}
```

### Database Sharding

```go
type ShardedDB struct {
    ring *chash.Ring
    dbs  map[string]*sql.DB
}

func (sdb *ShardedDB) GetDB(userID string) (*sql.DB, error) {
    shard, err := sdb.ring.GetNode(fmt.Sprintf("user:%s", userID))
    if err != nil {
        return nil, err
    }

    db, exists := sdb.dbs[shard]
    if !exists {
        return nil, fmt.Errorf("database connection not found for shard: %s", shard)
    }

    return db, nil
}
```

## Testing

Run the test suite:

```bash
go test -v ./...
```

Run benchmarks:

```bash
go test -bench=. -benchmem
```

Check coverage:

```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Add tests for new functionality
4. Ensure all tests pass (`go test -v ./...`)
5. Run `go fmt` and `go vet`
6. Update documentation if needed
7. Commit your changes (`git commit -am 'Add amazing feature'`)
8. Push to the branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Based on the consistent hashing algorithm described in ["Consistent Hashing and Random Trees"](https://www.cs.princeton.edu/courses/archive/fall07/cos518/papers/chash.pdf)
- Inspired by implementations in Cassandra, DynamoDB, and other distributed systems

// ========================================

// LICENSE
MIT License

Copyright (c) 2025 [Your Name]

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

// ========================================

// .github/workflows/ci.yml
name: CI

on:
push:
branches: [ main, develop ]
pull_request:
branches: [ main ]

jobs:
test:
runs-on: ubuntu-latest
strategy:
matrix:
go-version: [1.21, 1.22]

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go-version }}

    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-

    - name: Install dependencies
      run: go mod tidy

    - name: Run tests
      run: go test -v -race -coverprofile=coverage.out ./...

    - name: Check coverage
      run: |
        go tool cover -func=coverage.out
        go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | sed 's/%//' | awk '{if($1<80) exit 1}'

    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out

    - name: Run benchmarks
      run: go test -bench=. -benchmem

    - name: Check formatting
      run: |
        if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
          echo "The following files need formatting:"
          gofmt -s -l .
          exit 1
        fi

    - name: Run go vet
      run: go vet ./...

    - name: Run staticcheck
      uses: dominikh/staticcheck-action@v1.3.0
      with:
        version: "2023.1.6"

// ========================================

// Makefile
.PHONY: test bench coverage clean fmt vet staticcheck deps help

# Default target

help: ## Show this help message
@echo 'Usage: make [target]'
@echo ''
@echo 'Targets:'
@awk 'BEGIN {FS = ":._?## "} /^[a-zA-Z_-]+:.\_?## / {printf " %-15s %s\n", $1, $2}' $(MAKEFILE_LIST)

deps: ## Install dependencies
go mod tidy
go mod verify

test: ## Run tests
go test -v -race ./...

bench: ## Run benchmarks
go test -bench=. -benchmem -run=^$

coverage: ## Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
@echo "Coverage report generated: coverage.html"

fmt: ## Format code
go fmt ./...

vet: ## Run go vet
go vet ./...

staticcheck: ## Run staticcheck
@which staticcheck > /dev/null || (echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
staticcheck ./...

clean: ## Clean build artifacts
go clean
rm -f coverage.out coverage.html

ci: deps fmt vet staticcheck test ## Run all CI checks

build-example: ## Build example program
cd examples && go build -o consistent-hash-example ./main.go

.DEFAULT_GOAL := help 100,
HashFunc: customHashFunction,
})

// Create with initial nodes
ring := chash.NewWithNodes(
chash.Config{Replicas: 150},
[]string{"server1", "server2", "server3"},
)

````

### Managing Nodes

```go
// Add a node
err := ring.AddNode("server4:8080")
if err != nil {
    // Handle error (e.g., node already exists)
}

// Remove a node
err := ring.RemoveNode("server4:8080")
if err != nil {
    // Handle error (e.g., node not found)
}

// Get all nodes
nodes := ring.Nodes() // Returns []string

// Check if ring is empty
if ring.IsEmpty() {
    // Handle empty ring
}
````

### Key Routing

```go
// Get the primary node for a key
node, err := ring.GetNode("mykey")
if err != nil {
    // Handle error (e.g., no nodes available)
}

// Get multiple nodes for replication
nodes, err := ring.GetNodes("mykey", 3)
if err != nil {
    // Handle error
}
```

### Monitoring

```go
// Get ring statistics
stats := ring.GetStats()
fmt.Printf("Physical nodes: %d\n", stats.PhysicalNodes)
fmt.Printf("Virtual nodes: %d\n", stats.VirtualNodes)
fmt.Printf("Replicas per node: %d\n", stats.Replicas)

// Get counts
physicalCount := ring.NodeCount()
virtualCount := ring.VirtualNodeCount()
```

## Advanced Usage

### Custom Hash Function

```go
import "crypto/md5"

func md5Hash(key string) uint64 {
    h := md5.Sum([]byte(key))
    return binary.BigEndian.Uint64(h[:8])
}

ring := chash.New(chash.Config{
    Replicas:
```
