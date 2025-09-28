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
Yet to be done. Any kind of contributions are welcome.
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

````go
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
````

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

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Add tests for new functionality
4. Ensure all tests pass (`go test -v ./...`)
5. Run `go fmt`
6. Update documentation if needed
7. Commit your changes (`git commit -am 'Add amazing feature'`)
8. Push to the branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

## Acknowledgments

- Based on the consistent hashing algorithm described in ["Consistent Hashing and Random Trees"](https://www.cs.princeton.edu/courses/archive/fall07/cos518/papers/chash.pdf)
- Inspired by implementations in Cassandra, DynamoDB, and other distributed systems

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
