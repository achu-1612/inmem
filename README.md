# inmem

`inmem` is an in-memory cache library for Go, designed to provide a simple and efficient caching mechanism.


## Features

- **In-Memory Caching**: Store key-value pairs in memory for fast access.
- **Sharding**: Distribute cache items across multiple shards for improved performance.
- **Transactions**: Support for atomic and optimistic transactions.
- **Persistence**: Periodically save cache data to disk and load it on startup.
- **Eviction**: TTL based, LFU, LRU eviction policy for the keys.

## Installation

To install the package, run:

```sh
go get github.com/achu-1612/inmem
```

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/achu-1612/inmem"
)

func main() {
    cache, err := inmem.New(context.Background(), inmem.Options{})
    if err != nil {
        panic(err)
    }

    cache.Set("key", "value", 0)
    value, found := cache.Get("key")
    if found {
        fmt.Println("Found value:", value)
    } else {
        fmt.Println("Value not found")
    }
}
```

### Sharded Cache

```go
package main

import (
    "context"
    "github.com/achu-1612/inmem"
)

func main() {
    cache, err := inmem.New(context.Background(), inmem.Options{
        Sharding:   true,
        ShardCount: 4,
    })
    if err != nil {
        panic(err)
    }

    cache.Set("key", "value", 0)
    value, found := cache.Get("key")
    if found {
        fmt.Println("Found value:", value)
    } else {
        fmt.Println("Value not found")
    }
}
```

### Transactions

```go
package main

import (
    "context"
    "github.com/achu-1612/inmem"
)

func main() {
    cache, err := inmem.New(context.Background(), inmem.Options{
        TransactionType: inmem.TransactionTypeOptimistic,
    })
    if err != nil {
        panic(err)
    }

    cache.Begin()
    cache.Set("key", "value", 0)
    cache.Commit()

    value, found := cache.Get("key")
    if found {
        fmt.Println("Found value:", value)
    } else {
        fmt.Println("Value not found")
    }
}
```

### Persistence

```go
package main

import (
    "context"
    "github.com/achu-1612/inmem"
    "time"
)

func main() {
    cache, err := inmem.New(context.Background(), inmem.Options{
        Sync:           true,
        SyncInterval:   time.Minute,
        SyncFolderPath: "cache_data",
    })
    if err != nil {
        panic(err)
    }

    cache.Set("key", "value", 0)
    value, found := cache.Get("key")
    if found {
        fmt.Println("Found value:", value)
    } else {
        fmt.Println("Value not found")
    }
}
```

### Eviction

```go
package main

import (
    "context"
    "github.com/achu-1612/inmem"
    "github.com/achu-1612/inmem/eviction"
)

func main() {
    cache, err := inmem.New(context.Background(), inmem.Options{
        EvictionPolicy: eviction.PolicyLRU,,
        MaxSize: 2,
    })
    if err != nil {
        panic(err)
    }

    cache.Set("key1", "value", 0)
    cache.Set("key2", "value", 0)
    cache.Get("key2")
    cache.Set("key3", "value", 0) // key1 will be evicted as key2 has access frequency as 2 and key1 has 1
}
```

### Just the LFU/LRU cache

```go

package main

import (
    "context"
    "github.com/achu-1612/inmem/eviction"
)

func main() {
    cache, err := eviction.New(eviction.Options{
        Policy: eviction.PolicyLFU, // eviction.PolicyLRU can be used to LRU cache
        Capacity: 2,
        DeleteFinalizer: nil, // optional
        EvictFinalizer: nil,
    })
    if err != nil {
        panic(err)
    }

    cache.Set("key1", "value")
    cache.Set("key2", "value")
    cache.Get("key2")
    cache.Set("key3", "value" // key1 will be evicted as key2 has access frequency as 2 and key1 has 1
}
// Implment LFUResource, an entry for the LFU cache.
// If not default eviction entry will be use to deal with the cache data.

```


## License

This project is licensed under the MIT License.

## Contributing

Contributions are welcome! Open an issue or submit a pull request.

## Contact

For questions or support, reach out to [me](https://github.com/achu-1612)
Read the [blog post](https://dev.to/achu1612/introducing-inmem-lightweight-go-cache-engine-with-built-in-sharding-transaction-and-eviction-42f3) for inmem
